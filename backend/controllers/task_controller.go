package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskController struct {
	Repo             *repositories.TaskRepo
	UserRepo         *repositories.UserRepo
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
		pushNotification(c.Request.Context(), t.NotificationRepo, &models.Notification{
			OrganizationID: orgID,
			UserID:         assigneeID,
			TaskID:         task.ID,
			ProjectID:      projectID,
			Title:          "New Task Assigned",
			Message:        "You have been assigned to: " + task.Title,
		})
	}

	ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex(), ProjectID: projectID.Hex()})

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

	// Tell the assignee (if it is someone else) that the status changed
	if task, _ := t.Repo.FindByID(c.Request.Context(), orgID, taskID); task != nil &&
		!task.AssigneeID.IsZero() && task.AssigneeID != userID {
		pushNotification(c.Request.Context(), t.NotificationRepo, &models.Notification{
			OrganizationID: orgID,
			UserID:         task.AssigneeID,
			TaskID:         task.ID,
			ProjectID:      task.ProjectID,
			Title:          "Task Status Updated",
			Message:        "\"" + task.Title + "\" moved to " + statusLabel(in.Status),
		})
	}

	ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex(), ProjectID: c.Param("projectId")})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (t *TaskController) ListMyTasks(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	tasks, total, err := t.Repo.ListByAssignee(c.Request.Context(), orgID, userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []models.Task{}
	}
	c.JSON(http.StatusOK, gin.H{
		"tasks":      tasks,
		"totalCount": total,
		"page":       page,
		"limit":      limit,
	})
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

	ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex(), ProjectID: c.Param("projectId")})

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

func statusLabel(status string) string {
	switch status {
	case models.StatusTodo:
		return "To Do"
	case models.StatusInProgress:
		return "In Progress"
	case models.StatusUnderReview:
		return "Under Review"
	case models.StatusDone:
		return "Done"
	}
	return status
}

// Get returns one task (used when opening a task from a notification or activity link)
func (t *TaskController) Get(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := t.Repo.FindByID(c.Request.Context(), orgID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if task == nil || task.ProjectID.Hex() != c.Param("projectId") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// Update edits a task's title, description, priority, assignee and due date.
// Every field is optional: only the fields that are sent AND really changed are saved.
//   - assigneeId "" removes the assignee
//   - dueDate    "" removes the due date
func (t *TaskController) Update(c *gin.Context) {
	ctx := c.Request.Context()
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var in struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
		AssigneeID  *string `json:"assigneeId"`
		DueDate     *string `json:"dueDate"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	current, err := t.Repo.FindByID(ctx, orgID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if current == nil || current.ProjectID.Hex() != c.Param("projectId") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	set := bson.M{}
	unset := bson.M{}
	changes := []string{}
	var newAssignee primitive.ObjectID
	assigneeChanged := false

	// ---- title ----
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
			return
		}
		if len([]rune(title)) > 120 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title must be 120 characters or less"})
			return
		}
		if title != current.Title {
			set["title"] = title
			changes = append(changes, "changed the title")
		}
	}

	// ---- description ----
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if len([]rune(desc)) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Description must be 2000 characters or less"})
			return
		}
		if desc != current.Description {
			set["description"] = desc
			changes = append(changes, "updated the description")
		}
	}

	// ---- priority ----
	if in.Priority != nil {
		p := strings.ToLower(strings.TrimSpace(*in.Priority))
		if p != models.PriorityLow && p != models.PriorityMedium && p != models.PriorityHigh {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Priority must be low, medium or high"})
			return
		}
		if p != current.Priority {
			set["priority"] = p
			changes = append(changes, "changed priority to "+p)
		}
	}

	// ---- assignee ----
	if in.AssigneeID != nil {
		v := strings.TrimSpace(*in.AssigneeID)
		if v == "" {
			if !current.AssigneeID.IsZero() {
				unset["assigneeId"] = ""
				changes = append(changes, "removed the assignee")
			}
		} else {
			id, err := primitive.ObjectIDFromHex(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignee"})
				return
			}
			// The assignee must be a member of THIS workspace
			user, _ := t.UserRepo.FindByID(ctx, id)
			isMember := false
			if user != nil {
				for _, m := range user.Memberships {
					if m.OrganizationID == orgID {
						isMember = true
						break
					}
				}
			}
			if !isMember {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee must be a member of this workspace"})
				return
			}
			if id != current.AssigneeID {
				set["assigneeId"] = id
				changes = append(changes, "assigned the task to "+user.Name)
				assigneeChanged = true
				newAssignee = id
			}
		}
	}

	// ---- due date ----
	if in.DueDate != nil {
		v := strings.TrimSpace(*in.DueDate)
		if v == "" {
			if current.DueDate != nil {
				unset["dueDate"] = ""
				changes = append(changes, "removed the due date")
			}
		} else {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Due date must look like 2026-12-31"})
				return
			}
			if current.DueDate == nil || current.DueDate.UTC().Format("2006-01-02") != v {
				set["dueDate"] = d
				changes = append(changes, "set the due date to "+v)
			}
		}
	}

	// Nothing really changed: return the task as it is
	if len(changes) == 0 {
		c.JSON(http.StatusOK, current)
		return
	}

	if err := t.Repo.UpdateFields(ctx, orgID, taskID, set, unset); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := t.Repo.FindByID(ctx, orgID, taskID)
	if updated == nil {
		updated = current
	}

	// Activity log
	t.ActivityRepo.Create(ctx, &models.Activity{
		OrganizationID: orgID,
		TaskID:         taskID,
		UserID:         userID,
		Action:         models.ActionUpdatedTask,
		Details:        strings.Join(changes, ", "),
	})

	// Tell a newly assigned person
	if assigneeChanged && newAssignee != userID {
		pushNotification(ctx, t.NotificationRepo, &models.Notification{
			OrganizationID: orgID,
			UserID:         newAssignee,
			TaskID:         taskID,
			ProjectID:      updated.ProjectID,
			Title:          "New Task Assigned",
			Message:        fmt.Sprintf("You have been assigned to: %s", updated.Title),
		})
	}

	ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex(), ProjectID: c.Param("projectId")})

	c.JSON(http.StatusOK, updated)
}