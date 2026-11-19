package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/controllers"
	"github.com/yourname/multi-tenant-pm/middleware"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/services"
)

func Setup(r *gin.Engine) {
	// ==========================================
	// 1. Initialize Dependencies (Wiring)
	// ==========================================

	// Auth
	authSvc := &services.AuthService{Users: &repositories.UserRepo{}}
	authCtrl := &controllers.AuthController{Svc: authSvc}

	// Users
	userRepo := &repositories.UserRepo{}
	userCtrl := &controllers.UserController{Repo: userRepo}

	// Projects
	projectRepo := &repositories.ProjectRepo{}
	projectCtrl := &controllers.ProjectController{Repo: projectRepo}

	// Shared Repositories
	activityRepo := &repositories.ActivityRepo{}
	notificationRepo := &repositories.NotificationRepo{}
	taskRepo := &repositories.TaskRepo{}

	// Tasks
	taskCtrl := &controllers.TaskController{
		Repo:             taskRepo,
		ActivityRepo:     activityRepo,
		NotificationRepo: notificationRepo,
	}

	// Comments
	commentCtrl := &controllers.CommentController{
		Repo:             &repositories.CommentRepo{},
		ActivityRepo:     activityRepo,
		NotificationRepo: notificationRepo,
		TaskRepo:         taskRepo,
	}

	// Activities
	activityCtrl := &controllers.ActivityController{Repo: activityRepo}

	// Notifications
	notifCtrl := &controllers.NotificationController{Repo: notificationRepo}

	// Analytics
	analyticsCtrl := &controllers.AnalyticsController{}

	// Team
	teamCtrl := &controllers.TeamController{
		UserRepo: userRepo,
		InvRepo:  &repositories.InvitationRepo{},
	}

	// AI
	aiCtrl := &controllers.AIController{
		TaskRepo:    taskRepo,
		UserRepo:    userRepo,
		ProjectRepo: projectRepo,
	}

	// Workspace (NEW)
	workspaceCtrl := &controllers.WorkspaceController{
		UserRepo: userRepo,
	}

	// ==========================================
	// 2. Define Routes
	// ==========================================

	api := r.Group("/api")
	{
		// --- PUBLIC ROUTES ---
		api.POST("/auth/register", authCtrl.Register)
		api.POST("/auth/login", authCtrl.Login)
		api.POST("/invites/accept", teamCtrl.AcceptInvite)
		api.GET("/ws", controllers.HandleWS)

		// --- PROTECTED ROUTES (Requires JWT) ---
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			// Auth Check
			protected.GET("/me", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"userId":  c.GetString("userId"),
					"orgId":   c.GetString("orgId"),
					"orgName": c.GetString("orgName"),
					"role":    c.GetString("role"),
				})
			})

			// Users
			protected.GET("/users", userCtrl.ListMembers)

			// Workspaces
			protected.POST("/workspaces/switch/:orgId", workspaceCtrl.SwitchWorkspace)

			// Projects
			projects := protected.Group("/projects")
			{
				projects.GET("", projectCtrl.List)
				projects.POST("", middleware.RequireRole("owner", "admin", "manager"), projectCtrl.Create)
				projects.DELETE("/:projectId", middleware.RequireRole("owner", "admin", "manager", "member"), projectCtrl.Delete)
			}

			// Tasks
			tasks := protected.Group("/projects/:projectId/tasks")
			{
				tasks.GET("", taskCtrl.ListByProject)
				tasks.POST("", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.Create)
				tasks.PATCH("/:taskId/status", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.UpdateStatus)
				tasks.DELETE("/:taskId", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.Delete)
			}

			// Subtasks
			subtasks := protected.Group("/tasks/:taskId/subtasks")
			{
				subtasks.POST("", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.AddSubtask)
				subtasks.PATCH("/:subtaskId", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.ToggleSubtask)
				subtasks.PATCH("/:subtaskId/details", middleware.RequireRole("owner", "admin", "manager", "member"), taskCtrl.UpdateSubtask)

				// Subtask comments
				subtasks.GET("/:subtaskId/comments", commentCtrl.ListByTask)
				subtasks.POST("/:subtaskId/comments", middleware.RequireRole("owner", "admin", "manager", "member"), commentCtrl.Create)
			}

			// My Tasks
			protected.GET("/my-tasks", taskCtrl.ListMyTasks)

			// Comments
			comments := protected.Group("/tasks/:taskId/comments")
			{
				comments.GET("", commentCtrl.ListByTask)
				comments.POST("", middleware.RequireRole("owner", "admin", "manager", "member"), commentCtrl.Create)
			}

			// Activities
			activities := protected.Group("/tasks/:taskId/activities")
			{
				activities.GET("", activityCtrl.ListByTask)
			}

			// Notifications
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notifCtrl.List)
				notifications.PATCH("/:id/read", notifCtrl.MarkAsRead)
			}

			// Analytics
			protected.GET("/analytics", analyticsCtrl.GetDashboardStats)

			// Team Management
			protected.GET("/team", teamCtrl.ListMembers)
			protected.POST("/invites", middleware.RequireRole("owner", "admin"), teamCtrl.Invite)
			protected.PATCH("/team/:userId/role", middleware.RequireRole("owner", "admin"), teamCtrl.UpdateRole)
			protected.DELETE("/team/:userId", middleware.RequireRole("owner", "admin"), teamCtrl.RemoveUser)

			// AI
			protected.POST("/ai/generate-task", aiCtrl.GenerateTask)
			protected.POST("/ai/ask", aiCtrl.AskAI)
		}
	}
}
