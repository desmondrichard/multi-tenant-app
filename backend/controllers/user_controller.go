package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserController struct {
	Repo *repositories.UserRepo
}

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

	type UserResponse struct {
		ID    primitive.ObjectID `json:"id"`
		Name  string             `json:"name"`
		Email string             `json:"email"`
		Role  string             `json:"role"`
	}

	var result []UserResponse
	for _, user := range users {
		var roleForThisOrg string

		// Look through ALL memberships to find the one for THIS org
		for _, m := range user.Memberships {
			if m.OrganizationID == orgID {
				roleForThisOrg = m.Role
				break
			}
		}

		// If no role is found, default to "member"
		if roleForThisOrg == "" {
			roleForThisOrg = "member"
		}

		result = append(result, UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  roleForThisOrg,
		})
	}

	if result == nil {
		result = []UserResponse{}
	}

	c.JSON(http.StatusOK, result)
}
