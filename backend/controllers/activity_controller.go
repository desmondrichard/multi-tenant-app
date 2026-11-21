package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityController struct {
	Repo     *repositories.ActivityRepo
	TaskRepo *repositories.TaskRepo
	UserRepo *repositories.UserRepo
}

func (a *ActivityController) ListByTask(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	if task, err := a.TaskRepo.FindByID(c.Request.Context(), orgID, taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	activities, err := a.Repo.ListByTask(c.Request.Context(), orgID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if activities == nil {
		activities = []models.Activity{}
	}
	c.JSON(http.StatusOK, activities)
}

// ActivityDTO is one row of the workspace activity feed, with the user's name
// and the task's title already filled in.
type ActivityDTO struct {
	ID          primitive.ObjectID `json:"id"`
	Action      string             `json:"action"`
	Details     string             `json:"details"`
	CreatedAt   time.Time          `json:"createdAt"`
	UserID      primitive.ObjectID `json:"userId"`
	UserName    string             `json:"userName"`
	TaskID      primitive.ObjectID `json:"taskId"`
	TaskTitle   string             `json:"taskTitle"`
	ProjectID   string             `json:"projectId"`
	TaskDeleted bool               `json:"taskDeleted"`
}

// ListRecent returns the workspace-wide activity feed.
// Query params: action (all|created_task|changed_status|added_comment|deleted_task),
// userId, page (default 1), limit (default 20, max 50)
func (a *ActivityController) ListRecent(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	ctx := c.Request.Context()

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	var filterUser primitive.ObjectID
	if v := c.Query("userId"); v != "" && v != "all" {
		id, err := primitive.ObjectIDFromHex(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		filterUser = id
	}

	activities, total, err := a.Repo.ListByOrg(ctx, orgID, c.Query("action"), filterUser, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Collect the IDs we need names/titles for
	taskIDs := []primitive.ObjectID{}
	userIDs := []primitive.ObjectID{}
	seenTask := map[primitive.ObjectID]bool{}
	seenUser := map[primitive.ObjectID]bool{}
	for _, act := range activities {
		if !seenTask[act.TaskID] {
			seenTask[act.TaskID] = true
			taskIDs = append(taskIDs, act.TaskID)
		}
		if !seenUser[act.UserID] {
			seenUser[act.UserID] = true
			userIDs = append(userIDs, act.UserID)
		}
	}

	tasks, _ := a.TaskRepo.FindByIDs(ctx, orgID, taskIDs)
	users, _ := a.UserRepo.FindByIDs(ctx, userIDs)

	taskByID := map[primitive.ObjectID]models.Task{}
	for _, t := range tasks {
		taskByID[t.ID] = t
	}
	userNames := map[primitive.ObjectID]string{}
	for _, u := range users {
		userNames[u.ID] = u.Name
	}

	items := make([]ActivityDTO, 0, len(activities))
	for _, act := range activities {
		dto := ActivityDTO{
			ID:        act.ID,
			Action:    act.Action,
			Details:   act.Details,
			CreatedAt: act.CreatedAt,
			UserID:    act.UserID,
			UserName:  userNames[act.UserID],
			TaskID:    act.TaskID,
		}
		if t, ok := taskByID[act.TaskID]; ok {
			dto.TaskTitle = t.Title
			dto.ProjectID = t.ProjectID.Hex()
			dto.TaskDeleted = t.DeletedAt != nil
		} else {
			dto.TaskDeleted = true
		}
		items = append(items, dto)
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": items,
		"totalCount": total,
		"page":       page,
		"limit":      limit,
		"hasMore":    page*limit < total,
	})
}