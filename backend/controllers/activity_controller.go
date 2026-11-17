package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityController struct {
	Repo *repositories.ActivityRepo
}

func (a *ActivityController) ListByTask(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	taskID, err := primitive.ObjectIDFromHex(c.Param("taskId"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
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
