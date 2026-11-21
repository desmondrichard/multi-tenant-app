package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationController struct {
	Repo *repositories.NotificationRepo
}

// List returns the newest notifications. ?limit=10 (default 10, max 100).
// The response also carries totalCount and unreadCount, so the bell badge is
// correct even when only some notifications are loaded.
func (n *NotificationController) List(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	notifications, total, unread, err := n.Repo.ListByUser(c.Request.Context(), orgID, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if notifications == nil {
		notifications = []models.Notification{}
	}
	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"totalCount":    total,
		"unreadCount":   unread,
		"hasMore":       int64(len(notifications)) < total,
	})
}

func (n *NotificationController) MarkAsRead(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	notifID, err := primitive.ObjectIDFromHex(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := n.Repo.MarkAsRead(c.Request.Context(), orgID, userID, notifID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MarkAllAsRead marks all of the user's notifications as read
func (n *NotificationController) MarkAllAsRead(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	if err := n.Repo.MarkAllAsRead(c.Request.Context(), orgID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// pushNotification saves a notification and instantly pushes it (through
// Redis pub/sub + WebSocket) to the receiving user's open tabs.
func pushNotification(ctx context.Context, repo *repositories.NotificationRepo, n *models.Notification) {
	if err := repo.Create(ctx, n); err != nil {
		return
	}
	ws.Publish(ws.Event{
		Type:       "NOTIFICATION",
		OrgID:      n.OrganizationID.Hex(),
		TargetUser: n.UserID.Hex(),
		Title:      n.Title,
		Message:    n.Message,
	})
}