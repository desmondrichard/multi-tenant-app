package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkspaceController struct {
	UserRepo *repositories.UserRepo
}

func (w *WorkspaceController) SwitchWorkspace(c *gin.Context) {
	orgID, err := primitive.ObjectIDFromHex(c.Param("orgId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	user, err := w.UserRepo.FindByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Find the membership for the requested org
	var targetMembership *models.Membership
	for _, m := range user.Memberships {
		if m.OrganizationID == orgID {
			// Create a copy to avoid pointer loop issues
			mCopy := m
			targetMembership = &mCopy
			break
		}
	}

	if targetMembership == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not belong to this workspace"})
		return
	}

	// Generate a new token for this workspace
	newToken, _ := utils.GenerateToken(user.ID.Hex(), targetMembership.OrganizationID.Hex(), targetMembership.OrganizationName, targetMembership.Role)

	c.JSON(http.StatusOK, gin.H{"token": newToken, "orgName": targetMembership.OrganizationName})
}
