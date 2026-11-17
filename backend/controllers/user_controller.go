package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserController struct {
	Repo *repositories.UserRepo
}

// ListMembers returns all users belonging to the logged-in user's organization
func (u *UserController) ListMembers(c *gin.Context) {
	orgID, err := primitive.ObjectIDFromHex(c.GetString("orgId"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid organization context"})
		return
	}

	users, err := u.Repo.ListByOrg(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure we return an empty array instead of null if no users are found
	if users == nil {
		users = []models.User{}
	}

	c.JSON(http.StatusOK, users)
}
