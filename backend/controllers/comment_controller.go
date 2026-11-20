package controllers

import (
	"net/http"
	"strings"

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
	if strings.TrimSpace(in.Content) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Comment cannot be empty"})
		return
	}

	// Present only on /tasks/:taskId/subtasks/:subtaskId/comments
	subtaskID := ctx.Param("subtaskId")
	var subtaskTitle string
	if subtaskID != "" {
		st, err := c.TaskRepo.HasSubtask(ctx.Request.Context(), orgID, taskID, subtaskID)
		if err != nil || st == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
			return
		}
		subtaskTitle = st.Title
	}

	comment := &models.Comment{
		OrganizationID: orgID,
		TaskID:         taskID,
		SubtaskID:      subtaskID,
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
		Details:        activityDetails(subtaskID, subtaskTitle),
	})

	// 2. Send Notification to Assignee
	task, _ := c.TaskRepo.FindByID(ctx.Request.Context(), orgID, taskID)
	if task != nil && !task.AssigneeID.IsZero() && task.AssigneeID != userID {
		c.NotificationRepo.Create(ctx.Request.Context(), &models.Notification{
			OrganizationID: orgID,
			UserID:         task.AssigneeID,
			Title:          "New Comment",
			Message:        notificationMessage(subtaskID, subtaskTitle, task.Title),
		})
	}

	// 3. Publish through Redis -> every server pushes it to its WebSocket clients
	ws.Publish(ws.Event{
		Type:      "NEW_COMMENT",
		OrgID:     orgID.Hex(),
		TaskID:    taskID.Hex(),
		SubtaskID: subtaskID,
	})

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

	comments, err := c.Repo.ListByTask(ctx.Request.Context(), orgID, taskID, ctx.Param("subtaskId"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if comments == nil {
		comments = []models.Comment{}
	}
	ctx.JSON(http.StatusOK, comments)
}

func activityDetails(subtaskID, subtaskTitle string) string {
	if subtaskID == "" {
		return "Added a comment"
	}
	return "Commented on subtask: " + subtaskTitle
}

func notificationMessage(subtaskID, subtaskTitle, taskTitle string) string {
	if subtaskID == "" {
		return "Someone commented on your task: " + taskTitle
	}
	return "Someone commented on subtask \"" + subtaskTitle + "\" in your task: " + taskTitle
}
