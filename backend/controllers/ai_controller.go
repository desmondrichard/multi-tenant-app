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
	memberLines := make([]string, 0, len(users))
	for _, u := range users {
		role := models.RoleMember
		for _, m := range u.Memberships {
			if m.OrganizationID == orgID {
				role = m.Role
				break
			}
		}
		memberLines = append(memberLines, fmt.Sprintf("%s (role: %s)", u.Name, role))
	}

	// 2. Build the prompt
	now := time.Now()
	prompt := fmt.Sprintf(`You are an assistant inside a project management app. Turn the user's request
into ONE complete task with subtasks. Fill every field like a person using the "New Task" form would.

Today's date is %s (%s).
Team members you may assign to (name and role): %s

Rules:
- "title": short and clear (max 80 characters). Do not copy the whole prompt.
- "description": 1-3 sentences explaining what must be done and why.
- "priority": "low", "medium" or "high". Use "high" for words like urgent, critical, ASAP, bug, blocker. Default "medium".
- "status": "todo", "in_progress", "under_review" or "done". Default "todo". Only change it if the user clearly says so (e.g. "already started" = in_progress).
- "assignee": the EXACT name from the team members list if the user mentions someone (even by first name). If the user says a ROLE (e.g. "owner", "admin"), pick the member with that role. Otherwise "".
- "dueDate": format YYYY-MM-DD. Work out relative dates ("tomorrow", "next Friday", "in 2 weeks", "end of month") from today's date. If the user gives no date, use "".
- "subtasks": if the user says they do NOT want subtasks (e.g. "no subtasks", "without sub tasks"), return an empty list []. If the user lists steps, use them. Otherwise create 3 to 6 short, actionable subtasks that break the task down. Each subtask needs a "title" and a one-sentence "description".
- If the user gives explicit values (like "title: ...", "description: ...", "priority: ...", "due date: ..."), use those values exactly.

User request: %q`, now.Format("2006-01-02"), now.Weekday(), strings.Join(memberLines, ", "), in.Prompt)

	// 3. Call Gemini (forced to return JSON in our schema)
	aiResponse, err := callGemini(prompt, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   taskSchema(),
	})
	// If the model rejected the schema (HTTP 400), retry with plain JSON mode
	if err != nil && isBadRequestError(err) {
		fmt.Println("↩️ Retrying without response schema:", err)
		aiResponse, err = callGemini(prompt, &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
		})
	}
	if err != nil {
		fmt.Println("❌ AI error:", err)
		status, msg := aiErrorMessage(err)
		c.JSON(status, gin.H{"error": msg})
		return
	}

	// 4. Parse the AI response
	aiResponse = extractJSON(aiResponse)
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

func callGeminiWithSDK(prompt string) (string, error) {
	return callGemini(prompt, nil)
}

// geminiModels returns the models to try, in order.
// Override with GEMINI_MODELS=model-a,model-b in .env if Google renames models.
func geminiModels() []string {
	if v := strings.TrimSpace(os.Getenv("GEMINI_MODELS")); v != "" {
		list := []string{}
		for _, m := range strings.Split(v, ",") {
			if m = strings.TrimSpace(m); m != "" {
				list = append(list, m)
			}
		}
		if len(list) > 0 {
			return list
		}
	}
	return []string{"gemini-2.5-flash", "gemini-flash-latest", "gemini-2.5-flash-lite", "gemini-flash-lite-latest"}
}

func isRetryableAIError(err error) bool {
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "503") || strings.Contains(e, "unavailable") ||
		strings.Contains(e, "overloaded") || strings.Contains(e, "429") ||
		strings.Contains(e, "resource_exhausted")
}

func isBadRequestError(err error) bool {
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "400") || strings.Contains(e, "invalid_argument")
}

// aiErrorMessage turns a Gemini error into a clear message for the user
func aiErrorMessage(err error) (int, string) {
	e := strings.ToLower(err.Error())
	switch {
	case strings.Contains(e, "gemini_api_key is missing"):
		return http.StatusInternalServerError, "AI is not configured on the server: GEMINI_API_KEY is missing."
	case strings.Contains(e, "api key not valid") || strings.Contains(e, "api_key_invalid") ||
		strings.Contains(e, "permission_denied") || strings.Contains(e, "403"):
		return http.StatusBadGateway, "Google rejected the AI key. Check GEMINI_API_KEY on the server."
	case strings.Contains(e, "429") || strings.Contains(e, "resource_exhausted") || strings.Contains(e, "quota"):
		return http.StatusTooManyRequests, "AI usage limit reached. Please wait a minute and try again."
	case strings.Contains(e, "503") || strings.Contains(e, "unavailable") || strings.Contains(e, "overloaded") ||
		strings.Contains(e, "deadline exceeded") || strings.Contains(e, "timeout"):
		return http.StatusServiceUnavailable, "Gemini is overloaded right now. Please try again in a moment."
	}
	return http.StatusBadGateway, "The AI request failed. Please try again."
}

// callGemini sends the prompt. Pass a config to force a JSON response.
// It tries each model, and retries once on temporary overload / rate limits.
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

	var lastErr error
	started := time.Now()

	for _, modelName := range geminiModels() {
		// Never keep the user waiting more than ~35 seconds in total
		if time.Since(started) > 35*time.Second {
			break
		}

		// Gemini 2.5 Flash "thinks" before answering, which adds seconds.
		// Our task is simple, so turn thinking off to answer much faster.
		callCfg := &genai.GenerateContentConfig{}
		if cfg != nil {
			copied := *cfg
			callCfg = &copied
		}
		if strings.Contains(modelName, "2.5-flash") {
			callCfg.ThinkingConfig = &genai.ThinkingConfig{ThinkingBudget: genai.Ptr[int32](0)}
		}

		for attempt := 1; attempt <= 2; attempt++ {
			fmt.Printf("🔄 Trying model: %s (attempt %d)\n", modelName, attempt)

			// Each single call may take at most 15 seconds
			callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			resp, err := client.Models.GenerateContent(callCtx, modelName,
				[]*genai.Content{
					{Parts: []*genai.Part{{Text: prompt}}},
				}, callCfg)
			cancel()

			if err == nil && resp != nil && len(resp.Candidates) > 0 &&
				resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
				text := resp.Candidates[0].Content.Parts[0].Text
				fmt.Printf("✅ Success with model: %s\n", modelName)
				return text, nil
			}

			if err == nil {
				err = fmt.Errorf("model %s returned an empty response", modelName)
			}
			lastErr = err
			fmt.Printf("⚠️ Model %s failed: %v\n", modelName, err)

			// Temporary overload / rate limit: wait a little and try the same model once more
			if attempt == 1 && isRetryableAIError(err) && time.Since(started) < 25*time.Second {
				time.Sleep(2 * time.Second)
				continue
			}
			break
		}

		// A bad key / bad request will fail on every model - stop early
		if lastErr != nil && (strings.Contains(strings.ToLower(lastErr.Error()), "api key not valid") ||
			strings.Contains(strings.ToLower(lastErr.Error()), "api_key_invalid")) {
			break
		}
	}

	return "", fmt.Errorf("all AI models failed. Last error: %v", lastErr)
}

// extractJSON removes markdown fences and any text around the JSON object
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
