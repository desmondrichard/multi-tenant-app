package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ErrAccessRevoked means: the token is valid, but the user no longer exists
// or no longer belongs to the workspace written inside the token.
var ErrAccessRevoked = errors.New("access revoked")

var userRepo = &repositories.UserRepo{}

// VerifyMembership checks the DATABASE (not just the token signature):
// the user must still exist AND still be a member of the token's workspace.
// It returns the user's CURRENT membership, so role changes apply immediately.
func VerifyMembership(ctx context.Context, claims *utils.Claims) (*models.User, *models.Membership, error) {
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, nil, ErrAccessRevoked
	}
	orgID, err := primitive.ObjectIDFromHex(claims.OrganizationID)
	if err != nil {
		return nil, nil, ErrAccessRevoked
	}

	user, err := userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, err // database problem - NOT the same as "revoked"
	}
	if user == nil {
		return nil, nil, ErrAccessRevoked
	}

	for i := range user.Memberships {
		if user.Memberships[i].OrganizationID == orgID {
			m := user.Memberships[i]
			return user, &m, nil
		}
	}
	return nil, nil, ErrAccessRevoked
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := utils.ParseToken(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "INVALID_TOKEN"})
			return
		}

		// Re-check the database on every request
		_, membership, err := VerifyMembership(c.Request.Context(), claims)
		if err != nil {
			if errors.Is(err, ErrAccessRevoked) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Your access to this workspace has been removed",
					"code":  "ACCESS_REVOKED",
				})
				return
			}
			// Temporary DB error: don't log the user out
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session"})
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("orgId", claims.OrganizationID)
		c.Set("orgName", membership.OrganizationName) // from the database
		c.Set("role", membership.Role)                // from the database, not the token
		c.Next()
	}
}
