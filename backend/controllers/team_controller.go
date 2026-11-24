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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/utils"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TeamController struct {
	UserRepo *repositories.UserRepo
	InvRepo  *repositories.InvitationRepo
	TaskRepo *repositories.TaskRepo
}

// TeamMemberDTO is a flattened view of a user for ONE workspace,
// so the frontend gets the role for the current org directly.
type TeamMemberDTO struct {
	ID    primitive.ObjectID `json:"id"`
	Name  string             `json:"name"`
	Email string             `json:"email"`
	Role  string             `json:"role"`
}

// List Team Members
func (t *TeamController) ListMembers(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	users, _ := t.UserRepo.ListByOrg(c.Request.Context(), orgID)

	members := []TeamMemberDTO{}
	for _, u := range users {
		role := models.RoleMember
		for _, m := range u.Memberships {
			if m.OrganizationID == orgID {
				role = m.Role
				break
			}
		}
		members = append(members, TeamMemberDTO{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
			Role:  role,
		})
	}
	c.JSON(http.StatusOK, members)
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

	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	if in.Role != models.RoleAdmin && in.Role != models.RoleMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be admin or member"})
		return
	}

	ctx := c.Request.Context()

	// 0. The email service must be configured BEFORE we save anything
	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ BREVO_API_KEY is missing in .env")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email service not configured"})
		return
	}

	// 1. Duplicate checks, ONLY inside THIS workspace.
	//    The same email may still be invited to (or own) other workspaces.
	if existing, _ := t.UserRepo.FindByEmailAnyCase(ctx, in.Email); existing != nil {
		for _, m := range existing.Memberships {
			if m.OrganizationID == orgID {
				c.JSON(http.StatusConflict, gin.H{"error": "This person is already a member of this workspace"})
				return
			}
		}
	}
	if pending, _ := t.InvRepo.FindPending(ctx, orgID, in.Email); pending != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "An invitation for this email is already pending in this workspace"})
		return
	}

	// 2. Generate Token
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	// 3. Save Invitation to DB
	inv := &models.Invitation{
		OrganizationID: orgID,
		Email:          in.Email,
		Role:           in.Role,
		Token:          token,
	}

	if err := t.InvRepo.Create(ctx, inv); err != nil {
		// two admins inviting the same email at the same moment: the unique index stops the second one
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "An invitation for this email is already pending in this workspace"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Send Email via Brevo HTTP API

	frontendURL := os.Getenv("CLIENT_ORIGIN")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	inviteLink := fmt.Sprintf("%s/accept-invite?token=%s", frontendURL, token)

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "TaskFlow",
			"email": os.Getenv("SMTP_EMAIL"),
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
		t.InvRepo.Delete(ctx, inv.ID) // don't leave a pending invite that would block a retry
		c.JSON(http.StatusInternalServerError, gin.H{"error": "The invitation email could not be sent. Please try again."})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("❌ Brevo API Error:", string(body))
		t.InvRepo.Delete(ctx, inv.ID) // don't leave a pending invite that would block a retry
		c.JSON(http.StatusInternalServerError, gin.H{"error": "The invitation email could not be sent. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation email sent successfully"})
}

// Accept Invite (Public route)
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

	// Fetch Org Name
	var org models.Organization
	config.Database.Collection("organizations").FindOne(c.Request.Context(), bson.M{"_id": inv.OrganizationID}).Decode(&org)

	// Check if user already exists
	existingUser, _ := t.UserRepo.FindByEmail(c.Request.Context(), inv.Email)

	if existingUser != nil {
		// User exists! We need to check if they are ALREADY in this workspace
		isAlreadyMember := false
		currentRole := inv.Role
		for _, m := range existingUser.Memberships {
			if m.OrganizationID == inv.OrganizationID {
				isAlreadyMember = true
				currentRole = m.Role // keep their REAL role (e.g. owner), not the invitation's
				break
			}
		}

		if isAlreadyMember {
			// They are already in this workspace: close the invitation and just log them in
			t.InvRepo.MarkAccepted(c.Request.Context(), inv.ID)
			token, _ := utils.GenerateToken(existingUser.ID.Hex(), inv.OrganizationID.Hex(), org.Name, currentRole)
			c.JSON(http.StatusOK, gin.H{"token": token})
			return
		}

		// Add the new membership with the role from the INVITATION
		newMembership := models.Membership{
			OrganizationID:   inv.OrganizationID,
			OrganizationName: org.Name,
			Role:             inv.Role, // <-- THIS is the fix. It uses "member", not "owner"
		}

		existingUser.Memberships = append(existingUser.Memberships, newMembership)
		t.UserRepo.UpdateUser(c.Request.Context(), existingUser)
		t.InvRepo.MarkAccepted(c.Request.Context(), inv.ID)

		token, _ := utils.GenerateToken(existingUser.ID.Hex(), inv.OrganizationID.Hex(), org.Name, inv.Role)
		c.JSON(http.StatusOK, gin.H{"token": token})
		return
	}

	// User does not exist. Create new user with the INVITATION's role.
	hash, _ := utils.HashPassword(in.Password)
	user := &models.User{
		Name:         in.Name,
		Email:        inv.Email,
		PasswordHash: hash,
		Memberships: []models.Membership{
			{
				OrganizationID:   inv.OrganizationID,
				OrganizationName: org.Name,
				Role:             inv.Role, // <-- Uses "member"
			},
		},
	}

	if err := t.UserRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email might already be registered"})
		return
	}

	t.InvRepo.MarkAccepted(c.Request.Context(), inv.ID)
	token, _ := utils.GenerateToken(user.ID.Hex(), inv.OrganizationID.Hex(), org.Name, inv.Role)

	c.JSON(http.StatusOK, gin.H{"token": token})
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

	in.Role = strings.ToLower(strings.TrimSpace(in.Role))

	// There is exactly ONE owner per organization: the person who created it.
	if in.Role == models.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "An organization can have only one owner"})
		return
	}
	if in.Role != models.RoleAdmin && in.Role != models.RoleMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be admin or member"})
		return
	}

	// The owner's role can never be changed
	if target, _ := t.UserRepo.FindByID(c.Request.Context(), userID); target != nil {
		for _, m := range target.Memberships {
			if m.OrganizationID == orgID && m.Role == models.RoleOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "The owner's role cannot be changed"})
				return
			}
		}
	}

	if err := t.UserRepo.UpdateRoleInMemberships(c.Request.Context(), orgID, userID, in.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Tell that user's open tabs to reload their role immediately
	ws.Publish(ws.Event{Type: "ROLE_CHANGED", OrgID: orgID.Hex(), TargetUser: userID.Hex()})
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

	// An owner can never be removed
	if target, _ := t.UserRepo.FindByID(c.Request.Context(), userID); target != nil {
		for _, m := range target.Memberships {
			if m.OrganizationID == orgID && m.Role == models.RoleOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "The workspace owner cannot be removed"})
				return
			}
		}
	}

	if err := t.UserRepo.RemoveMembership(c.Request.Context(), orgID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Their tasks must not stay assigned to someone who can no longer see them
	if n, err := t.TaskRepo.UnassignUser(c.Request.Context(), orgID, userID); err == nil && n > 0 {
		ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex()})
	}

	// Log the removed user out IMMEDIATELY in every open tab / browser.
	// (Even without this push, their next API call returns 401 ACCESS_REVOKED.)
	ws.Publish(ws.Event{Type: "FORCE_LOGOUT", OrgID: orgID.Hex(), TargetUser: userID.Hex()})
	c.JSON(http.StatusOK, gin.H{"message": "User removed successfully"})
}
