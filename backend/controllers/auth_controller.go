package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/yourname/multi-tenant-pm/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate = validator.New()

type AuthController struct {
	Svc *services.AuthService
}

func (c *AuthController) Register(ctx *gin.Context) {
	var in services.RegisterInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validate.Struct(in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, _, err := c.Svc.Register(ctx.Request.Context(), in)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only the token is returned. The profile is loaded from GET /me
	ctx.JSON(http.StatusCreated, gin.H{"token": token})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var in services.LoginInput
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validate.Struct(in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, _, err := c.Svc.Login(ctx.Request.Context(), in)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

// Me returns the logged-in user's profile. The frontend keeps ONLY the token
// in localStorage and calls this endpoint to get name, email, role, workspaces.
// (AuthRequired already verified the user + workspace in the database.)
func (c *AuthController) Me(ctx *gin.Context) {
	userID, _ := primitive.ObjectIDFromHex(ctx.GetString("userId"))

	user, err := c.Svc.Users.FindByID(ctx.Request.Context(), userID)
	if err != nil || user == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not found", "code": "ACCESS_REVOKED"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"role":           ctx.GetString("role"),
		"organizationId": ctx.GetString("orgId"),
		"orgName":        ctx.GetString("orgName"),
		"memberships":    user.Memberships,
	})
}
