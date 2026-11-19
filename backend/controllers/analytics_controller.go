package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsController struct{}

func (a *AnalyticsController) GetDashboardStats(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	ctx := c.Request.Context()

	// 1. Count Projects
	projectsColl := config.Database.Collection("projects")
	totalProjects, _ := projectsColl.CountDocuments(ctx, bson.M{"organizationId": orgID, "deletedAt": nil})

	// 2. Count Tasks by Status
	tasksColl := config.Database.Collection("tasks")

	totalTasks, _ := tasksColl.CountDocuments(ctx, bson.M{"organizationId": orgID, "deletedAt": nil})

	completedTasks, _ := tasksColl.CountDocuments(ctx, bson.M{
		"organizationId": orgID,
		"deletedAt":      nil,
		"status":         "done",
	})

	underReviewTasks, _ := tasksColl.CountDocuments(ctx, bson.M{
		"organizationId": orgID,
		"deletedAt":      nil,
		"status":         "under_review", // <-- Added
	})

	inProgressTasks, _ := tasksColl.CountDocuments(ctx, bson.M{
		"organizationId": orgID,
		"deletedAt":      nil,
		"status":         "in_progress",
	})

	todoTasks, _ := tasksColl.CountDocuments(ctx, bson.M{
		"organizationId": orgID,
		"deletedAt":      nil,
		"status":         "todo",
	})

	// Return the aggregated data
	c.JSON(http.StatusOK, gin.H{
		"totalProjects": totalProjects,
		"totalTasks":    totalTasks,
		"tasksByStatus": []gin.H{
			{"name": "To Do", "value": todoTasks, "fill": "#94a3b8"},
			{"name": "In Progress", "value": inProgressTasks, "fill": "#3b82f6"},
			{"name": "Under Review", "value": underReviewTasks, "fill": "#f59e0b"}, // <-- Added
			{"name": "Done", "value": completedTasks, "fill": "#22c55e"},
		},
	})
}
