package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	ws "github.com/yourname/multi-tenant-pm/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genai"
)

type AIController struct {
	TaskRepo         *repositories.TaskRepo
	UserRepo         *repositories.UserRepo
	ProjectRepo      *repositories.ProjectRepo
	ActivityRepo     *repositories.ActivityRepo
	NotificationRepo *repositories.NotificationRepo
}

type AITaskRequest struct {
	Prompt    string `json:"prompt"`
	ProjectID string `json:"projectId"`
}

type AIAskRequest struct {
	Question string `json:"question"`
}

// AISubtask is one subtask suggested by the AI
type AISubtask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// AITaskResponse is the JSON the AI must return: every field that the
// "New Task" form has, plus the subtasks.
type AITaskResponse struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Priority    string      `json:"priority"`
	Status      string      `json:"status"`
	Assignee    string      `json:"assignee"`
	DueDate     string      `json:"dueDate"`
	Subtasks    []AISubtask `json:"subtasks"`
}

// taskSchema forces Gemini to answer in exactly this JSON shape
func taskSchema() *genai.Schema {
	str := func(desc string) *genai.Schema {
		return &genai.Schema{Type: genai.TypeString, Description: desc}
	}
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title":       str("Short task title, max 80 characters"),
			"description": str("1-3 sentences explaining what needs to be done"),
			"priority":    {Type: genai.TypeString, Enum: []string{"low", "medium", "high"}},
			"status":      {Type: genai.TypeString, Enum: []string{"todo", "in_progress", "under_review", "done"}},
			"assignee":    str("Exact name of a team member, or empty string"),
			"dueDate":     str("Due date as YYYY-MM-DD, or empty string"),
			"subtasks": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"title":       str("Short actionable subtask title"),
						"description": str("One sentence describing the subtask"),
					},
					Required: []string{"title", "description"},
				},
			},
		},
		Required: []string{"title", "description", "priority", "status", "assignee", "dueDate", "subtasks"},
	}
}

// findAssignee matches the name written by the AI (or the user) to a real team member
func findAssignee(users []models.User, name string) primitive.ObjectID {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return primitive.NilObjectID
	}
	// 1) exact match (ignoring case)
	for _, u := range users {
		if strings.ToLower(u.Name) == name {
			return u.ID
		}
	}
	// 2) partial match: "john" matches "John Smith"
	for _, u := range users {
		n := strings.ToLower(u.Name)
		if strings.Contains(n, name) || strings.Contains(name, n) {
			return u.ID
		}
	}
	return primitive.NilObjectID
}

// GenerateTask creates a COMPLETE task (and its subtasks) from a prompt,
// exactly like filling the "New Task" form and adding subtasks by hand.
func (a *AIController) GenerateTask(c *gin.Context) {
	var in AITaskRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please describe the task"})
		return
	}

	ctx := c.Request.Context()
	orgID, _ := primitive.ObjectIDFromHex(c.GetString("orgId"))
	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	projectID, err := primitive.ObjectIDFromHex(in.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// The project must exist in THIS workspace and not be deleted
	projects, _ := a.ProjectRepo.List(ctx, orgID)
	projectFound := false
	for _, p := range projects {
		if p.ID == projectID {
			projectFound = true
			break
		}
	}
	if !projectFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 1. Team members the AI can assign to
	users, _ := a.UserRepo.ListByOrg(ctx, orgID)
	userNames := make([]string, 0, len(users))
	for _, u := range users {
		userNames = append(userNames, u.Name)
	}

	// 2. Build the prompt
	now := time.Now()
	prompt := fmt.Sprintf(`You are an assistant inside a project management app. Turn the user's request
into ONE complete task with subtasks. Fill every field like a person using the "New Task" form would.

Today's date is %s (%s).
Team members you may assign to: %s

Rules:
- "title": short and clear (max 80 characters). Do not copy the whole prompt.
- "description": 1-3 sentences explaining what must be done and why.
- "priority": "low", "medium" or "high". Use "high" for words like urgent, critical, ASAP, bug, blocker. Default "medium".
- "status": "todo", "in_progress", "under_review" or "done". Default "todo". Only change it if the user clearly says so (e.g. "already started" = in_progress).
- "assignee": the EXACT name from the team members list if the user mentions someone (even by first name). Otherwise "".
- "dueDate": format YYYY-MM-DD. Work out relative dates ("tomorrow", "next Friday", "in 2 weeks", "end of month") from today's date. If the user gives no date, use "".
- "subtasks": if the user lists steps, use them. Otherwise create 3 to 6 short, actionable subtasks that break the task down. Each subtask needs a "title" and a one-sentence "description".

User request: %q`, now.Format("2006-01-02"), now.Weekday(), strings.Join(userNames, ", "), in.Prompt)

	// 3. Call Gemini (forced to return JSON in our schema)
	aiResponse, err := callGemini(prompt, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   taskSchema(),
	})
	if err != nil {
		fmt.Println("❌ AI error:", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "The AI is busy right now. Please try again in a moment."})
		return
	}

	// 4. Parse the AI response
	var parsed AITaskResponse
	if err := json.Unmarshal([]byte(aiResponse), &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI returned an invalid format. Please try again."})
		return
	}

	// 5. Validate / clean every field (never trust the AI blindly)
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The AI could not understand the request. Please add more detail."})
		return
	}
	if len(title) > 120 {
		title = title[:120]
	}

	priority := strings.ToLower(strings.TrimSpace(parsed.Priority))
	if priority != models.PriorityLow && priority != models.PriorityMedium && priority != models.PriorityHigh {
		priority = models.PriorityMedium
	}

	status := strings.ToLower(strings.TrimSpace(parsed.Status))
	switch status {
	case models.StatusTodo, models.StatusInProgress, models.StatusUnderReview, models.StatusDone:
	default:
		status = models.StatusTodo
	}

	var dueDate *time.Time
	if d, err := time.Parse("2006-01-02", strings.TrimSpace(parsed.DueDate)); err == nil {
		dueDate = &d
	}

	assigneeID := findAssignee(users, parsed.Assignee)

	subtasks := []models.Subtask{}
	for _, st := range parsed.Subtasks {
		stTitle := strings.TrimSpace(st.Title)
		if stTitle == "" {
			continue
		}
		subtasks = append(subtasks, models.Subtask{
			ID:          primitive.NewObjectID().Hex(),
			Title:       stTitle,
			Description: strings.TrimSpace(st.Description),
			Completed:   false,
		})
		if len(subtasks) == 10 {
			break
		}
	}

	// 6. Save the task (same fields as the manual "New Task" form)
	task := &models.Task{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Title:          title,
		Description:    strings.TrimSpace(parsed.Description),
		Status:         status,
		Priority:       priority,
		AssigneeID:     assigneeID,
		CreatedBy:      userID,
		DueDate:        dueDate,
		Subtasks:       subtasks,
	}
	if err := a.TaskRepo.Create(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save task"})
		return
	}

	// 7. Activity log + notification + live update (same as a manual task)
	a.ActivityRepo.Create(ctx, &models.Activity{
		OrganizationID: orgID,
		TaskID:         task.ID,
		UserID:         userID,
		Action:         models.ActionCreatedTask,
		Details:        "Created the task with AI",
	})

	if !assigneeID.IsZero() && assigneeID != userID {
		pushNotification(ctx, a.NotificationRepo, &models.Notification{
			OrganizationID: orgID,
			UserID:         assigneeID,
			TaskID:         task.ID,
			ProjectID:      projectID,
			Title:          "New Task Assigned",
			Message:        "You have been assigned to: " + task.Title,
		})
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
	return callGemini(prompt, nil)
}

// callGemini sends the prompt. Pass a config to force a JSON response schema.
func callGemini(prompt string, cfg *genai.GenerateContentConfig) (string, error) {
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
			}, cfg)

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
