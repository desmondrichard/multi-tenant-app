package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskController struct {
	Repo             *repositories.TaskRepo
	ActivityRepo     *repositories.ActivityRepo
	NotificationRepo *repositories.NotificationRepo
}

func (t *TaskController) Create(c *gin.Context) {
	var in struct {
		Title       string `json:"title" validate:"required,min=2"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		Status      string `json:"status"`
		AssigneeID  string `json:"assigneeId"`
		DueDate     string `json:"dueDate"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	projectID, err := primitive.ObjectIDFromHex(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var assigneeID primitive.ObjectID
	if in.AssigneeID != "" {
		assigneeID, _ = primitive.ObjectIDFromHex(in.AssigneeID)
	}

	// Parse Due Date
	var dueDate *time.Time
	if in.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", in.DueDate)
		if err == nil {
			dueDate = &parsedDate
		}
	}

	if in.Status == "" {
		in.Status = models.StatusTodo
	}
	if in.Priority == "" {
		in.Priority = models.PriorityMedium
	}

	task := &models.Task{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Title:          in.Title,
		Description:    in.Description,
		Status:         in.Status,
		Priority:       in.Priority,
		AssigneeID:     assigneeID,
		CreatedBy:      userID,
		DueDate:        dueDate,
	}

	if err := t.Repo.Create(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log Activity
	t.ActivityRepo.Create(c.Request.Context(), &models.Activity{
		OrganizationID: orgID,
		TaskID:         task.ID,
		UserID:         userID,
		Action:         models.ActionCreatedTask,
		Details:        "Created the task",
	})

	// Send Notification
	if !assigneeID.IsZero() && assigneeID != userID {
		t.NotificationRepo.Create(c.Request.Context(), &models.Notification{
			OrganizationID: orgID,
			UserID:         assigneeID,
			Title:          "New Task Assigned",
			Message:        "You have been assigned to: " + task.Title,
		})
	}

	c.JSON(http.StatusCreated, task)
}

func (t *TaskController) ListByProject(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	projectID, err := primitive.ObjectIDFromHex(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	search := c.Query("search")
	status := c.Query("status")
	priority := c.Query("priority")
	sortBy := c.Query("sort")

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)

	tasks, totalCount, err := t.Repo.ListByProject(
		c.Request.Context(), orgID, projectID,
		search, status, priority, sortBy, page, limit,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":      tasks,
		"totalCount": totalCount,
		"page":       page,
		"limit":      limit,
	})
}

func (t *TaskController) UpdateStatus(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var in struct {
		Status string `json:"status" validate:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := t.Repo.UpdateStatus(c.Request.Context(), orgID, taskID, in.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	t.ActivityRepo.Create(c.Request.Context(), &models.Activity{
		OrganizationID: orgID,
		TaskID:         taskID,
		UserID:         userID,
		Action:         models.ActionChangedStatus,
		Details:        "Changed status to " + in.Status,
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (t *TaskController) ListMyTasks(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	tasks, err := t.Repo.ListByAssignee(c.Request.Context(), orgID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}
	c.JSON(http.StatusOK, tasks)
}

// ==========================================
// SUBTASK CONTROLLERS
// ==========================================

func (t *TaskController) AddSubtask(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, _ := primitive.ObjectIDFromHex(c.Param("taskId"))

	var in struct {
		Title       string `json:"title" validate:"required,min=1"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create subtask with a string ID (fixes the 500 error)
	subtask := models.Subtask{
		ID:          primitive.NewObjectID().Hex(),
		Title:       in.Title,
		Description: in.Description,
		Completed:   false,
	}

	if err := t.Repo.AddSubtask(c.Request.Context(), orgID, taskID, subtask); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subtask)
}

func (t *TaskController) ToggleSubtask(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, _ := primitive.ObjectIDFromHex(c.Param("taskId"))

	// Get subtask ID as a string (matches the model)
	subtaskID := c.Param("subtaskId")

	var in struct {
		Completed bool `json:"completed"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := t.Repo.ToggleSubtask(c.Request.Context(), orgID, taskID, subtaskID, in.Completed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UpdateSubtask edits the title and description of a subtask
func (t *TaskController) UpdateSubtask(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}
	subtaskID := c.Param("subtaskId")

	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
		return
	}

	if err := t.Repo.UpdateSubtaskDetails(c.Request.Context(), orgID, taskID, subtaskID, in.Title, in.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": subtaskID, "title": in.Title, "description": in.Description})
}

// Delete soft-deletes a task: it is hidden from the app but stays in the database
func (t *TaskController) Delete(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	if err := t.Repo.SoftDelete(c.Request.Context(), orgID, taskID, userID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	t.ActivityRepo.Create(c.Request.Context(), &models.Activity{
		OrganizationID: orgID,
		TaskID:         taskID,
		UserID:         userID,
		Action:         models.ActionDeletedTask,
		Details:        "Deleted the task",
	})

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}
