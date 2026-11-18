package controllers

import (
	"fmt" // <-- Added this
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CommentController struct {
	Repo             *repositories.CommentRepo
	ActivityRepo     *repositories.ActivityRepo
	NotificationRepo *repositories.NotificationRepo
	TaskRepo         *repositories.TaskRepo
}

func (c *CommentController) Create(ctx *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(ctx.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(ctx.GetString("userId"))
	taskID, err := primitive.ObjectIDFromHex(ctx.Param("taskId"))

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var in struct {
		Content string `json:"content" validate:"required,min=1"`
	}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &models.Comment{
		OrganizationID: orgID,
		TaskID:         taskID,
		UserID:         userID,
		Content:        in.Content,
	}

	if err := c.Repo.Create(ctx.Request.Context(), comment); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 1. Log Activity
	c.ActivityRepo.Create(ctx.Request.Context(), &models.Activity{
		OrganizationID: orgID,
		TaskID:         taskID,
		UserID:         userID,
		Action:         models.ActionAddedComment,
		Details:        "Added a comment",
	})

	// 2. Send Notification to Assignee
	task, _ := c.TaskRepo.FindByID(ctx.Request.Context(), orgID, taskID)
	if task != nil && !task.AssigneeID.IsZero() && task.AssigneeID != userID {
		c.NotificationRepo.Create(ctx.Request.Context(), &models.Notification{
			OrganizationID: orgID,
			UserID:         task.AssigneeID,
			Title:          "New Comment",
			Message:        "Someone commented on your task: " + task.Title,
		})
	}

	// 3. Broadcast to WebSocket clients in the same Org
	// 3. Broadcast to WebSocket clients in the same Org (Run in background)
	broadcastMsg := fmt.Sprintf(`{"type":"NEW_COMMENT","orgId":"%s","taskId":"%s"}`, orgID.Hex(), taskID.Hex())
	go func() {
		ws.GlobalHub.Broadcast <- []byte(broadcastMsg)
	}()

	// 4. Send the final HTTP response ONLY ONCE
	ctx.JSON(http.StatusCreated, comment)
}

func (c *CommentController) ListByTask(ctx *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(ctx.GetString("orgId"))
	taskID, err := primitive.ObjectIDFromHex(ctx.Param("taskId"))

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	comments, err := c.Repo.ListByTask(ctx.Request.Context(), orgID, taskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if comments == nil {
		comments = []models.Comment{}
	}
	ctx.JSON(http.StatusOK, comments)
}
