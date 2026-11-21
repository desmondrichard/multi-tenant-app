package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectController struct {
	Repo *repositories.ProjectRepo
}

func (p *ProjectController) Create(c *gin.Context) {
	var in struct {
		Name        string `json:"name" validate:"required,min=2"`
		Key         string `json:"key" validate:"required,min=2,max=8"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	proj := &models.Project{
		OrganizationID: orgID,
		Name:           in.Name,
		Key:            in.Key,
		Description:    in.Description,
		CreatedBy:      userID,
	}

	if err := p.Repo.Create(c.Request.Context(), proj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ws.Publish(ws.Event{Type: "PROJECTS_UPDATED", OrgID: orgID.Hex()})

	c.JSON(http.StatusCreated, proj)
}

// List returns the workspace's projects.
//   - without ?page  -> a plain array of ALL projects (used by dropdowns / name lookups)
//   - with ?page&limit -> {projects, totalCount, page, limit} (used by the Projects page)
func (p *ProjectController) List(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	ctx := c.Request.Context()

	if c.Query("page") == "" {
		projects, err := p.Repo.List(ctx, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if projects == nil {
			projects = []models.Project{}
		}
		c.JSON(http.StatusOK, projects)
		return
	}

	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "9"), 10, 64)
	if limit < 1 {
		limit = 9
	}
	if limit > 50 {
		limit = 50
	}

	projects, total, err := p.Repo.ListPaged(ctx, orgID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []models.Project{}
	}
	c.JSON(http.StatusOK, gin.H{
		"projects":   projects,
		"totalCount": total,
		"page":       page,
		"limit":      limit,
	})
}

// Delete soft-deletes a project and its tasks: hidden from the app, kept in the database
func (p *ProjectController) Delete(c *gin.Context) {
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	projectID, err := primitive.ObjectIDFromHex(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := p.Repo.SoftDelete(c.Request.Context(), orgID, projectID, userID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ws.Publish(ws.Event{Type: "PROJECTS_UPDATED", OrgID: orgID.Hex()})

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}