package controllers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/config" // <-- Added
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/utils"
	"go.mongodb.org/mongo-driver/bson" // <-- Added
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TeamController struct {
	UserRepo *repositories.UserRepo
	InvRepo  *repositories.InvitationRepo
}

// List Team Members
func (t *TeamController) ListMembers(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	users, _ := t.UserRepo.ListByOrg(c.Request.Context(), orgID)
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

// Invite generates a token and sends an email via Brevo HTTP API
func (t *TeamController) Invite(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	var in struct {
		Email string `json:"email" validate:"required,email"`
		Role  string `json:"role" validate:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Generate Token
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	// 2. Save Invitation to DB
	inv := &models.Invitation{
		OrganizationID: orgID,
		Email:          in.Email,
		Role:           in.Role,
		Token:          token,
	}

	if err := t.InvRepo.Create(c.Request.Context(), inv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Send Email via Brevo HTTP API
	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ BREVO_API_KEY is missing in .env")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email service not configured"})
		return
	}

	// Use the frontend URL from environment variable (fallback to localhost)
	frontendURL := os.Getenv("CLIENT_ORIGIN")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	inviteLink := fmt.Sprintf("%s/accept-invite?token=%s", frontendURL, token)

	// Brevo API payload
	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "TaskFlow",
			"email": os.Getenv("SMTP_EMAIL"), // Your verified email in Brevo
		},
		"to": []map[string]string{
			{"email": in.Email},
		},
		"subject": "Invitation to join TaskFlow",
		"htmlContent": fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 500px; margin: 0 auto; padding: 20px; border: 1px solid #e2e8f0; border-radius: 8px;">
				<h2 style="color: #1e293b;">You've been invited to TaskFlow!</h2>
				<p style="color: #475569;">You have been invited to join the workspace as a <strong>%s</strong>.</p>
				<p style="color: #475569;">Click the button below to create your account and join the team.</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">Accept Invitation</a>
				</div>
				<p style="margin-top: 20px; font-size: 12px; color: #94a3b8;">Or copy and paste this link: <br/> %s</p>
			</div>
		`, in.Role, inviteLink, inviteLink),
	}

	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ Failed to reach Brevo API:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invite created, but email failed to send."})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("❌ Brevo API Error:", string(body))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invite created, but email failed to send."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation email sent successfully"})
}

// Accept Invite (Public route)
func (t *TeamController) AcceptInvite(c *gin.Context) {
	var in struct {
		Token    string `json:"token" validate:"required"`
		Name     string `json:"name" validate:"required"`
		Password string `json:"password" validate:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inv, err := t.InvRepo.FindByToken(c.Request.Context(), in.Token)
	if err != nil || inv == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired invite token"})
		return
	}

	// Hash password
	hash, _ := utils.HashPassword(in.Password)

	// Create User
	user := &models.User{
		OrganizationID: inv.OrganizationID,
		Name:           in.Name,
		Email:          inv.Email,
		PasswordHash:   hash,
		Role:           inv.Role,
	}

	if err := t.UserRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email might already be registered"})
		return
	}

	t.InvRepo.MarkAccepted(c.Request.Context(), inv.ID)

	// --- FIX: Fetch org name to pass to GenerateToken ---
	var org models.Organization
	config.Database.Collection("organizations").FindOne(c.Request.Context(), bson.M{"_id": user.OrganizationID}).Decode(&org)

	token, _ := utils.GenerateToken(user.ID.Hex(), user.OrganizationID.Hex(), org.Name, user.Role)

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// UpdateRole updates a user's role (Promote/Demote)
func (t *TeamController) UpdateRole(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var in struct {
		Role string `json:"role" validate:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := t.UserRepo.UpdateRole(c.Request.Context(), orgID, userID, in.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// RemoveUser deletes a user from the organization
func (t *TeamController) RemoveUser(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Prevent owners from deleting themselves
	currentUserID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	if userID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot remove yourself"})
		return
	}

	if err := t.UserRepo.RemoveUser(c.Request.Context(), orgID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User removed successfully"})
}
