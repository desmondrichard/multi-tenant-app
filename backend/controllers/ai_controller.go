package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genai"
)

type AIController struct {
	TaskRepo    *repositories.TaskRepo
	UserRepo    *repositories.UserRepo
	ProjectRepo *repositories.ProjectRepo
}

type AITaskRequest struct {
	Prompt    string `json:"prompt"`
	ProjectID string `json:"projectId"`
}

type AIAskRequest struct {
	Question string `json:"question"`
}

type AITaskResponse struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Assignee string `json:"assignee"`
}

// GenerateTask creates a task based on a prompt
func (a *AIController) GenerateTask(c *gin.Context) {
	var in AITaskRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	projectID, _ := primitive.ObjectIDFromHex(in.ProjectID)

	// 1. Fetch users for AI context
	users, _ := a.UserRepo.ListByOrg(c.Request.Context(), orgID)
	var userNames []string
	for _, u := range users {
		userNames = append(userNames, u.Name)
	}

	// 2. Build Prompt
	prompt := fmt.Sprintf(`
		You are an AI assistant for a project management app.
		The user will give you a prompt. You must extract:
		- "title": A short title for the task.
		- "priority": Must be exactly "low", "medium", or "high".
		- "assignee": The name of the person to assign. Choose from: %v. If no one matches, leave it empty.
		
		Respond ONLY with a valid JSON object. Do not include markdown formatting.
		User Prompt: "%s"
	`, userNames, in.Prompt)

	// 3. Call Gemini
	aiResponse, err := callGeminiWithSDK(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI failed: " + err.Error()})
		return
	}

	// 4. Parse AI Response
	var parsedResult AITaskResponse
	if err := json.Unmarshal([]byte(aiResponse), &parsedResult); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI returned invalid format", "raw": aiResponse})
		return
	}

	// 5. Find Assignee
	var assigneeID primitive.ObjectID
	for _, u := range users {
		if u.Name == parsedResult.Assignee {
			assigneeID = u.ID
			break
		}
	}

	// 6. Save Task
	task := &models.Task{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Title:          parsedResult.Title,
		Priority:       parsedResult.Priority,
		AssigneeID:     assigneeID,
		Status:         models.StatusTodo,
		CreatedBy:      userID,
		Subtasks:       []models.Subtask{},
	}

	if err := a.TaskRepo.Create(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save task"})
		return
	}

	ws.Publish(ws.Event{Type: "TASKS_UPDATED", OrgID: orgID.Hex(), ProjectID: projectID.Hex()})

	c.JSON(http.StatusOK, task)
}

// AskAI answers questions based on workspace data
func (a *AIController) AskAI(c *gin.Context) {
	var in AIAskRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))

	// 1. Fetch Projects, Tasks, and Users for context
	projects, _ := a.ProjectRepo.List(c.Request.Context(), orgID)
	tasksResult, _, _ := a.TaskRepo.ListByProject(c.Request.Context(), orgID, primitive.NilObjectID, "", "", "", "", 1, 100)
	users, _ := a.UserRepo.ListByOrg(c.Request.Context(), orgID)

	// 2. Build Context String
	contextStr := "--- WORKSPACE DATA ---\n"
	contextStr += "PROJECTS:\n"
	for _, p := range projects {
		contextStr += fmt.Sprintf("- %s (Key: %s): %s\n", p.Name, p.Key, p.Description)
	}

	contextStr += "\nTEAM MEMBERS:\n"
	for _, u := range users {
		// Fix: Find the role from the memberships array for the current org
		role := "member"
		for _, m := range u.Memberships {
			if m.OrganizationID == orgID {
				role = m.Role
				break
			}
		}
		contextStr += fmt.Sprintf("- %s (%s)\n", u.Name, role)
	}

	contextStr += "\nTASKS:\n"
	if tasksResult != nil {
		for _, t := range tasksResult {
			assigneeName := "Unassigned"
			for _, u := range users {
				if u.ID == t.AssigneeID {
					assigneeName = u.Name
					break
				}
			}
			contextStr += fmt.Sprintf("- [%s] %s | Status: %s | Priority: %s | Assignee: %s\n", t.ID.Hex(), t.Title, t.Status, t.Priority, assigneeName)
		}
	}

	// 3. Build Prompt
	prompt := fmt.Sprintf(`
		You are a helpful AI assistant for a project management app.
		Here is the current data of the workspace:
		%s
		
		User Question: "%s"
		
		Answer the user's question based ONLY on the data provided above. Be concise and professional. If the answer is not in the data, say "I don't have that information in the workspace."
	`, contextStr, in.Question)

	// 4. Call Gemini
	answer, err := callGeminiWithSDK(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI is busy, please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"answer": answer})
}

// --- Gemini API Helper ---

func callGeminiWithSDK(prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is missing in .env")
	}

	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("client init failed: %v", err)
	}

	modelsList := []string{"gemini-flash-latest", "gemini-2.5-flash", "gemini-3.8-flash"}
	var lastErr error

	for _, modelName := range modelsList {
		fmt.Printf("🔄 Trying model: %s\n", modelName)

		resp, err := client.Models.GenerateContent(ctx, modelName,
			[]*genai.Content{
				{Parts: []*genai.Part{{Text: prompt}}},
			}, nil)

		if err == nil && len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			text := resp.Candidates[0].Content.Parts[0].Text
			fmt.Printf("✅ Success with model: %s\n", modelName)
			return cleanJSON(text), nil
		}

		lastErr = err
		fmt.Printf("⚠️ Model %s failed: %v\n", modelName, err)
	}

	return "", fmt.Errorf("all AI models failed. Last error: %v", lastErr)
}

func cleanJSON(s string) string {
	if len(s) > 7 && s[:7] == "```json" {
		s = s[7:]
	}
	if len(s) > 3 && s[len(s)-3:] == "```" {
		s = s[:len(s)-3]
	}
	return s
}
