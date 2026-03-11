package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/company"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/i18n"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/user"
)

// Tool represents a function that the agent can call
type Tool struct {
	Name           string                                                                               `json:"name"`
	Description    string                                                                               `json:"description"`
	Parameters     map[string]interface{}                                                               `json:"parameters"`
	ShouldStopLoop bool                                                                                 `json:"should_stop_loop"`
	Execute        func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) `json:"-"`
}

// ToolManager manages available tools
type ToolManager struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

var (
	toolManager *ToolManager
	toolOnce    sync.Once
)

// loadToolPrompt loads tool description from a file in the prompts directory
func loadToolPrompt(toolName string) string {
	cfg := GetConfig()
	if cfg.PromptsDir == "" {
		return ""
	}

	promptFile := filepath.Join(cfg.PromptsDir, toolName+".md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		return ""
	}

	return string(content)
}

// GetToolManager returns the singleton tool manager
func GetToolManager() *ToolManager {
	toolOnce.Do(func() {
		toolManager = &ToolManager{
			tools: make(map[string]*Tool),
		}
	})
	return toolManager
}

// RegisterTool registers a new tool
func (tm *ToolManager) RegisterTool(tool *Tool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if tool.Execute == nil {
		return fmt.Errorf("tool execute function cannot be nil")
	}

	tm.tools[tool.Name] = tool
	return nil
}

// UnregisterTool removes a tool
func (tm *ToolManager) UnregisterTool(name string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tools, name)
}

// GetTool returns a tool by name
func (tm *ToolManager) GetTool(name string) (*Tool, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tool, exists := tm.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (tm *ToolManager) GetAllTools() []*Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tools := make([]*Tool, 0, len(tm.tools))
	for _, tool := range tm.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolDefinitions returns tool definitions for LLM
func (tm *ToolManager) GetToolDefinitions() []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	definitions := make([]map[string]interface{}, 0, len(tm.tools))
	for _, tool := range tm.tools {
		def := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		}
		definitions = append(definitions, def)
	}
	return definitions
}

// GetEnabledTools returns tools that are enabled in the configuration
func (tm *ToolManager) GetEnabledTools() []*Tool {
	cfg := GetConfig()
	if len(cfg.EnabledTools) == 0 {
		return tm.GetAllTools()
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	enabled := make([]*Tool, 0)
	for _, name := range cfg.EnabledTools {
		if tool, exists := tm.tools[name]; exists {
			enabled = append(enabled, tool)
		}
	}
	return enabled
}

// ExecuteTool executes a tool by name with given parameters
func (tm *ToolManager) ExecuteTool(name string, ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
	tm.mu.RLock()
	tool, exists := tm.tools[name]
	tm.mu.RUnlock()

	if !exists {
		return &ToolExecutionResult{
			Error: fmt.Sprintf("tool '%s' not found", name),
		}, fmt.Errorf("tool '%s' not found", name)
	}

	result, err := tool.Execute(ctx, params)

	return result, err
}

// RegisterDefaultTools registers the default set of tools
func RegisterDefaultTools() error {
	tm := GetToolManager()
	sm := GetSkillManager()

	questionTool := &Tool{
		Name:           "question",
		ShouldStopLoop: false,
		Description:    loadToolPrompt("question"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"questions": map[string]interface{}{
					"description": "Questions to ask",
					"type":        "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"question": map[string]interface{}{
								"description": "Complete question",
								"type":        "string",
							},
							"header": map[string]interface{}{
								"description": "Very short label (max 30 chars)",
								"type":        "string",
							},
							"options": map[string]interface{}{
								"description": "Available choices",
								"type":        "array",
								"items": map[string]interface{}{
									"ref":  "QuestionOption",
									"type": "object",
									"properties": map[string]interface{}{
										"label": map[string]interface{}{
											"description": "Display text (1-5 words, concise)",
											"type":        "string",
										},
										"description": map[string]interface{}{
											"description": "Explanation of choice",
											"type":        "string",
										},
									},
									"required": []string{"label", "description"},
								},
							},
							"multiple": map[string]interface{}{
								"description": "Allow selecting multiple choices",
								"type":        "boolean",
							},
						},
						"required": []string{"question", "header", "options"},
					},
				},
			},
			"required": []string{"questions"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			questionsData, ok := params["questions"]
			if !ok {
				return &ToolExecutionResult{
					Error: "questions is required. Please provide a 'questions' array parameter containing question objects.",
				}, fmt.Errorf("questions is required")
			}

			questionsArray, ok := questionsData.([]interface{})
			if !ok {
				return &ToolExecutionResult{
					Error: "questions must be an array of question objects. Format: {\"questions\": [{\"question\": \"...\", \"header\": \"...\", \"options\": [...]}]}",
				}, fmt.Errorf("questions must be an array")
			}

			if len(questionsArray) == 0 {
				return &ToolExecutionResult{
					Error: "questions array cannot be empty. Please provide at least one question.",
				}, fmt.Errorf("questions array cannot be empty")
			}

			for i, q := range questionsArray {
				questionObj, ok := q.(map[string]interface{})
				if !ok {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("question at index %d must be an object with 'question', 'header', and 'options' fields", i),
					}, fmt.Errorf("question at index %d must be an object", i)
				}

				questionText, ok := questionObj["question"].(string)
				if !ok || questionText == "" {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("question at index %d must have a non-empty 'question' field", i),
					}, fmt.Errorf("question at index %d missing 'question' field", i)
				}

				header, ok := questionObj["header"].(string)
				if !ok || header == "" {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("question at index %d must have a non-empty 'header' field (max 30 chars)", i),
					}, fmt.Errorf("question at index %d missing 'header' field", i)
				}

				if len(header) > 30 {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("question at index %d 'header' field exceeds 30 character limit (current: %d)", i, len(header)),
					}, fmt.Errorf("question at index %d header too long", i)
				}

				options, ok := questionObj["options"].([]interface{})
				if !ok || len(options) == 0 {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("question at index %d must have a non-empty 'options' array", i),
					}, fmt.Errorf("question at index %d missing or empty 'options' array", i)
				}

				for j, opt := range options {
					optObj, ok := opt.(map[string]interface{})
					if !ok {
						return &ToolExecutionResult{
							Error: fmt.Sprintf("option at index %d in question %d must be an object with 'label' and 'description' fields", j, i),
						}, fmt.Errorf("option at index %d must be an object", j)
					}

					label, ok := optObj["label"].(string)
					if !ok || label == "" {
						return &ToolExecutionResult{
							Error: fmt.Sprintf("option at index %d in question %d must have a non-empty 'label' field", j, i),
						}, fmt.Errorf("option at index %d missing 'label' field", j)
					}

					description, ok := optObj["description"].(string)
					if !ok || description == "" {
						return &ToolExecutionResult{
							Error: fmt.Sprintf("option at index %d in question %d must have a non-empty 'description' field", j, i),
						}, fmt.Errorf("option at index %d missing 'description' field", j)
					}
				}
			}

			questionsJSON, err := json.Marshal(questionsData)
			if err != nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf("failed to marshal questions: %v", err),
				}, fmt.Errorf("failed to marshal questions: %w", err)
			}

			ctx.QuestionData = string(questionsJSON)
			ctx.WaitingForAnswer = true

			return &ToolExecutionResult{
				Result: i18n.T(ctx.Language, "ai.tool.question_sent"),
				StopCommand: &ToolStopCommand{
					Response: i18n.T(ctx.Language, "ai.tool.need_info"),
					Metadata: map[string]interface{}{
						"question": true,
					},
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(questionTool); err != nil {
		return fmt.Errorf("failed to register question tool: %w", err)
	}

	showNavigationTool := &Tool{
		Name:           "show_status_or_progress",
		ShouldStopLoop: true,
		Description:    loadToolPrompt("show_status_or_progress"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"projectId": map[string]interface{}{
					"type":        "integer",
					"description": "Project ID. Use -1 for favorites (company-wide status), specific project ID for person's status, or user's own project ID for personal status.",
				},
				"label": map[string]interface{}{
					"type":        "string",
					"description": "The button label text (required)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the target entity to display after the label (optional, e.g., person's name, company name)",
				},
			},
			"required": []string{"projectId", "label"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			label, ok := params["label"].(string)
			if !ok || label == "" {
				return &ToolExecutionResult{
					Error: "label is required",
				}, fmt.Errorf("label is required")
			}

			var title string
			if t, ok := params["title"].(string); ok {
				title = t
			}

			projectIdFloat, ok := params["projectId"].(float64)
			if !ok {
				return &ToolExecutionResult{
					Error: "projectId is required and must be an integer",
				}, fmt.Errorf("projectId is required and must be an integer")
			}
			projectId := int64(projectIdFloat)

			routeParams := map[string]interface{}{
				"projectId": projectId,
			}

			if projectId > 0 {
				label += fmt.Sprintf(" %d", projectId)
			}

			ctx.ButtonNavigation = &chat_session.ButtonNavigation{
				RouteName: "project.index",
				Params:    routeParams,
				Label:     label,
				Title:     title,
			}

			ctx.NavigationInfo = &NavigationInfo{
				RouteName: "project.index",
				Params:    routeParams,
			}
			ctx.ShouldNavigate = true

			return &ToolExecutionResult{
				Result: i18n.T(ctx.Language, "ai.tool.ok"),
			}, nil
		},
	}

	if err := tm.RegisterTool(showNavigationTool); err != nil {
		return fmt.Errorf("failed to register show_status_or_progress tool: %w", err)
	}

	replyTaskTool := &Tool{
		Name:           "message_reply",
		ShouldStopLoop: true,
		Description:    loadToolPrompt("message_reply"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Your response message to the user. This is REQUIRED.",
				},
			},
			"required": []string{"content"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			content, ok := params["content"].(string)
			if !ok {
				return &ToolExecutionResult{
					Error: "content is required",
				}, fmt.Errorf("content is required")
			}

			return &ToolExecutionResult{
				Result: content,
				StopCommand: &ToolStopCommand{
					Response: content,
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(replyTaskTool); err != nil {
		return fmt.Errorf("failed to register message_reply tool: %w", err)
	}

	if len(sm.GetAllSkills()) > 0 {
		skillTool := &Tool{
			Name:           "skill",
			ShouldStopLoop: false,
			Description:    `Load a skill to get detailed instructions for a specific task. Skills provide specialized knowledge and step-by-step guidance. Use this when a task matches an available skill's description. Only the skills listed here are available: ` + sm.FormatSkillsForTool(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"description": "The skill identifier from available_skills (e.g., 'skill-creator', 'chinese-novelist', ...)",
						"type":        "string",
					},
				},
				"required":             []string{"name"},
				"additionalProperties": false,
			},
			Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
				skillName, ok := params["name"].(string)
				if !ok {
					return &ToolExecutionResult{
						Error: "name is required",
					}, fmt.Errorf("name is required")
				}

				skillContent, err := sm.LoadSkillContent(skillName)
				if err != nil {
					available := strings.Join(func() []string {
						skills := sm.GetAllSkills()
						names := make([]string, 0, len(skills))
						for name := range skills {
							names = append(names, name)
						}
						return names
					}(), ", ")
					return &ToolExecutionResult{
						Error: fmt.Sprintf("skill '%s' not found. Available skills: %s", skillName, available),
					}, fmt.Errorf("skill '%s' not found. Available skills: %s", skillName, available)
				}

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("## Skill: %s\n\n", skillContent.Metadata.Name))
				sb.WriteString(fmt.Sprintf("**Base directory**: %s\n\n", skillContent.Dir))
				if skillContent.Metadata.Description != "" {
					sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", skillContent.Metadata.Description))
				}
				if skillContent.Metadata.License != "" {
					sb.WriteString(fmt.Sprintf("**License**: %s\n\n", skillContent.Metadata.License))
				}
				if skillContent.Metadata.Compatibility != "" {
					sb.WriteString(fmt.Sprintf("**Compatibility**: %s\n\n", skillContent.Metadata.Compatibility))
				}
				sb.WriteString(strings.TrimSpace(skillContent.Content))

				return &ToolExecutionResult{
					Result: sb.String(),
				}, nil
			},
		}

		if err := tm.RegisterTool(skillTool); err != nil {
			return fmt.Errorf("failed to register skill tool: %w", err)
		}
	}

	assignTaskTool := &Tool{
		Name:           "assign_task",
		ShouldStopLoop: true,
		Description:    loadToolPrompt("assign_task"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_identifier": map[string]interface{}{
					"type":        "integer",
					"description": "The user ID (integer) of subordinate staff member to assign task to. Must match a subordinate in the context.",
				},
				"task_title": map[string]interface{}{
					"type":        "string",
					"description": "The title or description of task to assign",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority level: 'high', 'medium', or 'low'. Default is 'medium' if not specified.",
					"enum":        []string{"high", "medium", "low"},
				},
				"time_expression": map[string]interface{}{
					"type":        "string",
					"description": "Optional natural language time expression (e.g., 'tomorrow', 'Next Friday', '2 hours later', '2024-12-25'). If provided, this overrides the priority-based due date calculation. Current date/time is " + time.Now().Format("2006-01-02") + ".",
				},
			},
			"required": []string{"user_identifier", "task_title"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			userIdentifierFloat, ok := params["user_identifier"].(float64)
			if !ok {
				return &ToolExecutionResult{
					Error: "user_identifier must be an integer",
				}, fmt.Errorf("user_identifier must be an integer")
			}

			userIdentifier := int64(userIdentifierFloat)
			if userIdentifier <= 0 {
				return &ToolExecutionResult{
					Error: "user_identifier must be a positive integer",
				}, fmt.Errorf("user_identifier must be a positive integer")
			}

			taskTitle, ok := params["task_title"].(string)
			if !ok || taskTitle == "" {
				return &ToolExecutionResult{
					Error: "task_title is required",
				}, fmt.Errorf("task_title is required")
			}

			priority := "medium"
			if p, ok := params["priority"].(string); ok {
				priority = p
			}

			var priorityValue int64
			var daysToAdd int

			switch strings.ToLower(priority) {
			case "high":
				priorityValue = 3
				daysToAdd = 1
			case "low":
				priorityValue = 1
				daysToAdd = 7
			default:
				priorityValue = 2
				daysToAdd = 3
			}

			var targetStaff *chat_session.SubordinateStaffInfo

			for _, staff := range ctx.SubordinateStaff {
				if staff.UserID == userIdentifier {
					targetStaff = &staff
					break
				}
			}

			if targetStaff == nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf("No staff found with user ID %d", userIdentifier),
				}, fmt.Errorf("no staff found with user ID %d", userIdentifier)
			}

			s := db.NewSession()
			if s == nil {
				return &ToolExecutionResult{
					Error: "Failed to create database session",
				}, fmt.Errorf("failed to create database session")
			}
			defer s.Close()

			authUser := &user.User{
				ID: ctx.UserID,
			}

			var projectID int64
			if targetStaff.ProjectID > 0 {
				projectID = targetStaff.ProjectID
			} else {
				targetUser := &user.User{ID: targetStaff.UserID}
				projectsInterface, _, _, err := (&models.Project{}).ReadAll(s, targetUser, "", 1, 1)
				if err != nil {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("Failed to get projects for user %s: %v", targetStaff.Username, err),
					}, fmt.Errorf("failed to get projects: %w", err)
				}

				projects, ok := projectsInterface.([]*models.Project)
				if !ok || len(projects) == 0 {
					if targetStaff.UserID == ctx.UserID {
						defaultProject := &models.Project{
							Title:       i18n.T(ctx.Language, "ai.tool.default_project.title"),
							Description: i18n.T(ctx.Language, "ai.tool.default_project.description"),
							OwnerID:     ctx.UserID,
						}
						if err := defaultProject.Create(s, authUser); err != nil {
							return &ToolExecutionResult{
								Error: fmt.Sprintf("Failed to create default project: %v", err),
							}, fmt.Errorf("failed to create default project: %w", err)
						}
						projectID = defaultProject.ID
					} else {
						return &ToolExecutionResult{
							Error: fmt.Sprintf("Staff member %s does not have an associated project", targetStaff.Username),
						}, fmt.Errorf("staff member %s does not have an associated project", targetStaff.Username)
					}
				} else {
					projectID = projects[0].ID
				}
			}

			now := time.Now()

			var dueDate time.Time
			var timeExpressionUsed bool

			if timeExpr, ok := params["time_expression"].(string); ok && timeExpr != "" {
				parsedTime, err := parseTimeExpression(timeExpr, now)
				if err != nil {
					return &ToolExecutionResult{
						Error: fmt.Sprintf("Failed to parse time expression '%s': %v", timeExpr, err),
					}, fmt.Errorf("failed to parse time expression: %w", err)
				}
				dueDate = parsedTime
				timeExpressionUsed = true
			} else {
				dueDate = now.AddDate(0, 0, daysToAdd)
				timeExpressionUsed = false
			}

			if dueDate.Before(now.Add(time.Hour)) {
				dueDate = now.Add(time.Hour)
			}

			task := &models.Task{
				Title:      taskTitle,
				ProjectID:  projectID,
				StartDate:  now,
				DueDate:    dueDate,
				Priority:   priorityValue,
				IsFavorite: true,
				Assignees:  []*user.User{{ID: targetStaff.UserID}},
			}

			err := task.Create(s, authUser)
			if err != nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf("Failed to create task: %v", err),
				}, fmt.Errorf("failed to create task: %w", err)
			}

			subscription := &models.Subscription{
				EntityType: models.SubscriptionEntityTask,
				EntityID:   task.ID,
			}
			if err := subscription.Create(s, authUser); err != nil {
				task.IsFavorite = false
			}

			displayName := targetStaff.Name
			if displayName == "" {
				displayName = targetStaff.Username
			}

			priorityMap := map[string]string{
				"high":   i18n.T(ctx.Language, "ai.tool.assign_task.priority_high"),
				"medium": i18n.T(ctx.Language, "ai.tool.assign_task.priority_medium"),
				"low":    i18n.T(ctx.Language, "ai.tool.assign_task.priority_low"),
			}

			response := fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.assign_task.success"), displayName, taskTitle)
			response += "\n"
			response += fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.assign_task.priority_label"), priorityMap[priority])
			response += "\n"
			if timeExpressionUsed {
				response += fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.assign_task.due_date_time"), dueDate.Format("2006-01-02 15:04"))
				response += "\n"
			} else {
				response += fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.assign_task.due_date_priority"), dueDate.Format("2006-01-02"))
				response += "\n"
			}

			label := i18n.T(ctx.Language, "ai.tool.assign_task.view_task")
			if task.ID > 0 {
				label += fmt.Sprintf(" %d", task.ID)
			}

			ctx.ButtonNavigation = &chat_session.ButtonNavigation{
				RouteName: "task.detail",
				Params: map[string]interface{}{
					"id": task.ID,
				},
				Label: label,
				Title: taskTitle,
			}

			return &ToolExecutionResult{
				Result: response,
				StopCommand: &ToolStopCommand{
					Response: response,
					Metadata: map[string]interface{}{
						"task_id":         task.ID,
						"assigned_to":     targetStaff.UserID,
						"project_id":      projectID,
						"priority":        priority,
						"due_date":        dueDate.Format(time.RFC3339),
						"time_expression": timeExpressionUsed,
					},
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(assignTaskTool); err != nil {
		return fmt.Errorf("failed to register assign_task tool: %w", err)
	}

	addSubordinateTool := &Tool{
		Name:           "add_subordinate",
		ShouldStopLoop: true,
		Description:    loadToolPrompt("add_subordinate"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"username": map[string]interface{}{
					"type":        "string",
					"description": "The username of the user to add as subordinate (required)",
				},
				"nickname": map[string]interface{}{
					"type":        "string",
					"description": "The display name/nickname for the user (optional, will be used to update user settings)",
				},
			},
			"required": []string{"username"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			username, ok := params["username"].(string)
			if !ok || username == "" {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.add_subordinate.error_username_required"),
				}, fmt.Errorf("username is required")
			}

			nickname, _ := params["nickname"].(string)

			s := db.NewSession()
			if s == nil {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.add_subordinate.error_db_session"),
				}, fmt.Errorf("failed to create database session")
			}
			defer s.Close()

			if err := s.Begin(); err != nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_start_transaction"), err),
				}, fmt.Errorf("failed to start transaction: %w", err)
			}

			authUser := &user.User{ID: ctx.UserID}

			if ctx.CompanyID == 0 {
				s.Rollback()
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.add_subordinate.error_company_required"),
				}, fmt.Errorf("user is not in a company")
			}

			userRole := company.GetUserRole(s, ctx.UserID, ctx.CompanyID)
			if userRole != "creator" {
				s.Rollback()
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.add_subordinate.error_not_creator"),
				}, fmt.Errorf("user is not a company creator")
			}

			log.Debugf("Looking up user with username: '%s' (len=%d)", username, len(username))
			targetUser, err := user.GetUserByUsername(s, username)
			if err != nil {
				log.Debugf("User lookup failed: %v", err)
				s.Rollback()
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_user_not_exist"), username),
				}, fmt.Errorf("user does not exist: %w", err)
			}
			log.Debugf("Found user: ID=%d, Username='%s', Name='%s'", targetUser.ID, targetUser.Username, targetUser.Name)

			targetUserRole := company.GetUserRole(s, targetUser.ID, ctx.CompanyID)
			if targetUserRole == "" {
				if err := company.AddStaffToCompany(s, ctx.CompanyID, targetUser.ID, "staff"); err != nil {
					s.Rollback()
					return &ToolExecutionResult{
						Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_add_to_company"), err),
					}, fmt.Errorf("failed to add user to company: %w", err)
				}
			}

			relation := &company.CompanyRelation{}
			has, err := s.Where("company_id = ? AND superior_user_id = ? AND subordinate_user_id = ?", ctx.CompanyID, ctx.UserID, targetUser.ID).Get(relation)
			if err != nil {
				s.Rollback()
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_create_relation"), err),
				}, fmt.Errorf("failed to check existing relation: %w", err)
			}
			if has {
				s.Rollback()
				displayName := nickname
				if displayName == "" {
					displayName = targetUser.Username
				}
				return &ToolExecutionResult{
					Result: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.success_already_subordinate"), displayName, relation.ProjectID),
					StopCommand: &ToolStopCommand{
						Response: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.response_already_subordinate"), displayName),
					},
				}, nil
			}

			projectTitle := fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.project_title"), username)
			project := &models.Project{
				Title:   projectTitle,
				OwnerID: ctx.UserID,
			}
			if err := project.Create(s, authUser); err != nil {
				s.Rollback()
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_create_project"), err),
				}, fmt.Errorf("failed to create project: %w", err)
			}

			projectUser := &models.ProjectUser{
				ProjectID:  project.ID,
				Username:   username,
				Permission: models.PermissionWrite,
			}
			if err := projectUser.Create(s, authUser); err != nil {
				s.Rollback()
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_share_project"), err),
				}, fmt.Errorf("failed to share project: %w", err)
			}

			newRelation := &company.CompanyRelation{
				CompanyID:         ctx.CompanyID,
				SuperiorUserID:    ctx.UserID,
				SubordinateUserID: targetUser.ID,
				ProjectID:         project.ID,
			}
			_, err = s.Insert(newRelation)
			if err != nil {
				s.Rollback()
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_create_relation"), err),
				}, fmt.Errorf("failed to create relation: %w", err)
			}

			if nickname != "" {
				targetUser.Name = nickname
				targetUser.DiscoverableByName = true
				targetUser.DiscoverableByEmail = true
				if _, err := user.UpdateUser(s, targetUser, false); err != nil {
					s.Rollback()
					return &ToolExecutionResult{
						Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_update_nickname"), err),
					}, fmt.Errorf("failed to update user: %w", err)
				}
			}

			if err := s.Commit(); err != nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.error_commit"), err),
				}, fmt.Errorf("failed to commit: %w", err)
			}

			displayName := nickname
			if displayName == "" {
				displayName = targetUser.Username
			}

			response := fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.add_subordinate.success"), displayName, project.ID)

			return &ToolExecutionResult{
				Result: response,
				StopCommand: &ToolStopCommand{
					Response: response,
					Metadata: map[string]interface{}{
						"subordinate_user_id": targetUser.ID,
						"project_id":          project.ID,
						"company_id":          ctx.CompanyID,
					},
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(addSubordinateTool); err != nil {
		return fmt.Errorf("failed to register add_subordinate tool: %w", err)
	}

	if err := tm.RegisterTool(addSubordinateTool); err != nil {
		return fmt.Errorf("failed to register add_subordinate tool: %w", err)
	}

	updateNicknameTool := &Tool{
		Name:           "update_nickname",
		ShouldStopLoop: true,
		Description:    loadToolPrompt("update_nickname"),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"username": map[string]interface{}{
					"type":        "string",
					"description": "The username of the user whose nickname to update (required)",
				},
				"nickname": map[string]interface{}{
					"type":        "string",
					"description": "The new nickname/display name to set for the user (required)",
				},
			},
			"required": []string{"username", "nickname"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			username, ok := params["username"].(string)
			if !ok || username == "" {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.update_nickname.error_username_required"),
				}, fmt.Errorf("username is required")
			}

			nickname, ok := params["nickname"].(string)
			if !ok || nickname == "" {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.update_nickname.error_nickname_required"),
				}, fmt.Errorf("nickname is required")
			}

			s := db.NewSession()
			if s == nil {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.add_subordinate.error_db_session"),
				}, fmt.Errorf("failed to create database session")
			}
			defer s.Close()

			log.Debugf("Looking up user with username: '%s' (len=%d)", username, len(username))
			targetUser, err := user.GetUserByUsername(s, username)
			if err != nil {
				log.Debugf("User lookup failed: %v", err)
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.update_nickname.error_user_not_exist"), username),
				}, fmt.Errorf("user does not exist: %w", err)
			}
			log.Debugf("Found user: ID=%d, Username='%s', Name='%s'", targetUser.ID, targetUser.Username, targetUser.Name)

			if ctx.CompanyID == 0 {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.update_nickname.error_company_required"),
				}, fmt.Errorf("user is not in a company")
			}

			currentUserRole := company.GetUserRole(s, ctx.UserID, ctx.CompanyID)
			if currentUserRole != "creator" {
				return &ToolExecutionResult{
					Error: i18n.T(ctx.Language, "ai.tool.update_nickname.error_not_creator"),
				}, fmt.Errorf("only company creator can update member nicknames")
			}

			oldName := targetUser.Name
			if oldName == "" {
				oldName = targetUser.Username
			}

			targetUser.Name = nickname
			targetUser.DiscoverableByName = true
			targetUser.DiscoverableByEmail = true

			_, err = user.UpdateUser(s, targetUser, false)
			if err != nil {
				return &ToolExecutionResult{
					Error: fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.update_nickname.error_update"), err),
				}, fmt.Errorf("failed to update nickname: %w", err)
			}

			response := fmt.Sprintf(i18n.T(ctx.Language, "ai.tool.update_nickname.success"), username, oldName, nickname)

			return &ToolExecutionResult{
				Result: response,
				StopCommand: &ToolStopCommand{
					Response: response,
					Metadata: map[string]interface{}{
						"user_id":      targetUser.ID,
						"old_nickname": oldName,
						"new_nickname": nickname,
					},
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(updateNicknameTool); err != nil {
		return fmt.Errorf("failed to register update_nickname tool: %w", err)
	}

	listSubordinatesTool := &Tool{
		Name:           "list_subordinates",
		ShouldStopLoop: true,
		Description:    `List all subordinate staff members. Triggered when the user asks about their subordinates, staff, or team members.`,
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			tableContent := "员工ID\t用户名\t当前昵称\n"
			staffCount := len(ctx.SubordinateStaff)
			for i, staff := range ctx.SubordinateStaff {
				if i == staffCount-1 && staff.UserID == ctx.UserID {
					continue
				}
				tableContent += fmt.Sprintf("%d\t%s\t%s\n", staff.UserID, staff.Username, staff.Name)
			}

			return &ToolExecutionResult{
				Result: tableContent,
				StopCommand: &ToolStopCommand{
					Response: tableContent,
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(listSubordinatesTool); err != nil {
		return fmt.Errorf("failed to register list_subordinates tool: %w", err)
	}

	return nil
}

// parseTimeExpression parses natural language time expressions and returns the corresponding time
// Supports:
// - Absolute dates: 2024-12-25, 2024/12/25, 2024年12月25日
// - Relative dates: today, tomorrow, yesterday, 今天, 明天, 昨天
// - Relative times: in 2 hours, 30 minutes later, 2小时后, 30分钟后
// - Time periods: next week, this month, next year, 下周, 本月, 明年
// - Weekdays: next Monday, last Friday, 下周一, 上周五
// - Weekday in period: Tuesday in next week, 下周三, 周五在下周
// - Day in month: 25th day in next month, 下个月的25号
func parseTimeExpression(expr string, now time.Time) (time.Time, error) {
	expr = strings.ToLower(strings.TrimSpace(expr))

	// Weekday in period: Tuesday in next week, 下周三, 周五在下周
	if t, ok := parseWeekdayInPeriod(expr, now); ok {
		return t, nil
	}

	// Day in month: 25th day in next month, 下个月的25号
	if t, ok := parseDayInMonth(expr, now); ok {
		return t, nil
	}

	// Absolute dates in various formats
	if t, ok := parseAbsoluteDate(expr, now); ok {
		return t, nil
	}

	// Relative dates: today, tomorrow, yesterday
	if t, ok := parseRelativeDate(expr, now); ok {
		return t, nil
	}

	// Relative times: in 2 hours, 30 minutes later, etc.
	if t, ok := parseRelativeTime(expr, now); ok {
		return t, nil
	}

	// Time periods: next week, this month, etc.
	if t, ok := parseTimePeriod(expr, now); ok {
		return t, nil
	}

	// Weekdays: next Monday, last Friday, etc.
	if t, ok := parseWeekday(expr, now); ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time expression: %s", expr)
}

// parseAbsoluteDate parses absolute date formats
func parseAbsoluteDate(expr string, now time.Time) (time.Time, bool) {
	// YYYY-MM-DD format
	re := regexp.MustCompile(`^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})`)
	if matches := re.FindStringSubmatch(expr); matches != nil {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		t := time.Date(year, time.Month(month), day, now.Hour(), now.Minute(), 0, 0, now.Location())
		return t, true
	}

	// DD-MM-YYYY or DD/MM/YYYY format
	re = regexp.MustCompile(`^(\d{1,2})[-/.](\d{1,2})[-/.](\d{4})`)
	if matches := re.FindStringSubmatch(expr); matches != nil {
		day, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		year, _ := strconv.Atoi(matches[3])
		t := time.Date(year, time.Month(month), day, now.Hour(), now.Minute(), 0, 0, now.Location())
		return t, true
	}

	// Chinese format: 2024年12月25日
	re = regexp.MustCompile(`^(\d{4})年(\d{1,2})月(\d{1,2})日`)
	if matches := re.FindStringSubmatch(expr); matches != nil {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		t := time.Date(year, time.Month(month), day, now.Hour(), now.Minute(), 0, 0, now.Location())
		return t, true
	}

	// Chinese format without year: 12月25日
	re = regexp.MustCompile(`^(\d{1,2})月(\d{1,2})日`)
	if matches := re.FindStringSubmatch(expr); matches != nil {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		t := time.Date(now.Year(), time.Month(month), day, now.Hour(), now.Minute(), 0, 0, now.Location())
		return t, true
	}

	return time.Time{}, false
}

// parseRelativeDate parses relative dates like today, tomorrow, yesterday
func parseRelativeDate(expr string, now time.Time) (time.Time, bool) {
	switch expr {
	case "today", "今天":
		return now, true
	case "tomorrow", "明天":
		return now.AddDate(0, 0, 1), true
	case "yesterday", "昨天":
		return now.AddDate(0, 0, -1), true
	}
	return time.Time{}, false
}

// parseRelativeTime parses relative times like "in 2 hours", "30 minutes later"
func parseRelativeTime(expr string, now time.Time) (time.Time, bool) {
	totalSeconds := 0

	// English patterns
	patterns := []struct {
		re         string
		multiplier int
	}{
		{`in\s+(\d+)\s+(second|minute|hour|day|week|month|year)s?`, 1},
		{`(\d+)\s+(second|minute|hour|day|week|month|year)s?\s+later`, 1},
		{`after\s+(\d+)\s+(second|minute|hour|day|week|month|year)s?`, 1},
	}

	unitMultipliers := map[string]int{
		"second": 1,
		"minute": 60,
		"hour":   3600,
		"day":    86400,
		"week":   604800,
		"month":  2592000,
		"year":   31536000,
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p.re)
		if matches := re.FindStringSubmatch(expr); matches != nil {
			amount, _ := strconv.Atoi(matches[1])
			unit := matches[2]
			totalSeconds += amount * unitMultipliers[unit]
		}
	}

	// Chinese patterns
	cnPatterns := []struct {
		re         string
		multiplier int
	}{
		{`(\d+)\s*秒(?:后|之?后)`, 1},
		{`(\d+)\s*分(?:钟)?(?:后|之?后)`, 60},
		{`(\d+)\s*小(?:时)?(?:后|之?后)`, 3600},
		{`(\d+)\s*天(?:后|之?后)`, 86400},
		{`(\d+)\s*周(?:后|之?后)`, 604800},
		{`(\d+)\s*月(?:后|之?后)`, 2592000},
		{`(\d+)\s*年(?:后|之?后)`, 31536000},
	}

	for _, p := range cnPatterns {
		re := regexp.MustCompile(p.re)
		if matches := re.FindStringSubmatch(expr); matches != nil {
			amount, _ := strconv.Atoi(matches[1])
			totalSeconds += amount * p.multiplier
		}
	}

	if totalSeconds > 0 {
		return now.Add(time.Duration(totalSeconds) * time.Second), true
	}

	return time.Time{}, false
}

// parseTimePeriod parses time periods like "next week", "this month"
func parseTimePeriod(expr string, now time.Time) (time.Time, bool) {
	// English patterns
	if strings.Contains(expr, "next week") {
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		return now.AddDate(0, 0, daysUntilSunday), true
	}
	if strings.Contains(expr, "this week") {
		daysUntilSaturday := (6 - int(now.Weekday())%7)
		if daysUntilSaturday < 0 {
			daysUntilSaturday += 7
		}
		return now.AddDate(0, 0, daysUntilSaturday), true
	}
	if strings.Contains(expr, "last week") {
		daysBack := int(now.Weekday()) + 7
		return now.AddDate(0, 0, -daysBack), true
	}
	if strings.Contains(expr, "next month") {
		if now.Month() == 12 {
			return time.Date(now.Year()+1, 1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
		}
		return time.Date(now.Year(), now.Month()+1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if strings.Contains(expr, "this month") {
		lastDay := time.Date(now.Year(), now.Month()+1, 0, now.Hour(), now.Minute(), 0, 0, now.Location())
		return lastDay, true
	}
	if strings.Contains(expr, "last month") {
		if now.Month() == 1 {
			return time.Date(now.Year()-1, 12, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
		}
		return time.Date(now.Year(), now.Month()-1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if strings.Contains(expr, "next year") {
		return time.Date(now.Year()+1, 1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if strings.Contains(expr, "this year") {
		return time.Date(now.Year(), 12, 31, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}

	// Chinese patterns
	if regexp.MustCompile(`下(?:个)?周`).MatchString(expr) {
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		return now.AddDate(0, 0, daysUntilSunday), true
	}
	if regexp.MustCompile(`本(?:个)?周`).MatchString(expr) {
		daysUntilSaturday := (6 - int(now.Weekday())%7)
		if daysUntilSaturday < 0 {
			daysUntilSaturday += 7
		}
		return now.AddDate(0, 0, daysUntilSaturday), true
	}
	if regexp.MustCompile(`上(?:个)?周`).MatchString(expr) {
		daysBack := int(now.Weekday()) + 7
		return now.AddDate(0, 0, -daysBack), true
	}
	if regexp.MustCompile(`下(?:个)?月`).MatchString(expr) {
		if now.Month() == 12 {
			return time.Date(now.Year()+1, 1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
		}
		return time.Date(now.Year(), now.Month()+1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if regexp.MustCompile(`本月`).MatchString(expr) {
		lastDay := time.Date(now.Year(), now.Month()+1, 0, now.Hour(), now.Minute(), 0, 0, now.Location())
		return lastDay, true
	}
	if regexp.MustCompile(`上(?:个)?月`).MatchString(expr) {
		if now.Month() == 1 {
			return time.Date(now.Year()-1, 12, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
		}
		return time.Date(now.Year(), now.Month()-1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if regexp.MustCompile(`明年`).MatchString(expr) {
		return time.Date(now.Year()+1, 1, 1, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}
	if regexp.MustCompile(`今年`).MatchString(expr) {
		return time.Date(now.Year(), 12, 31, now.Hour(), now.Minute(), 0, 0, now.Location()), true
	}

	return time.Time{}, false
}

// parseWeekday parses weekday expressions like "next Monday", "last Friday"
func parseWeekday(expr string, now time.Time) (time.Time, bool) {
	weekdays := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	cnWeekdays := map[string]time.Weekday{
		"周日": time.Sunday, "星期日": time.Sunday,
		"周一": time.Monday, "星期一": time.Monday,
		"周二": time.Tuesday, "星期二": time.Tuesday,
		"周三": time.Wednesday, "星期三": time.Wednesday,
		"周四": time.Thursday, "星期四": time.Thursday,
		"周五": time.Friday, "星期五": time.Friday,
		"周六": time.Saturday, "星期六": time.Saturday,
	}

	var targetWeekday time.Weekday
	var found bool

	for day, wd := range weekdays {
		if strings.Contains(expr, day) {
			targetWeekday = wd
			found = true
			break
		}
	}

	if !found {
		for day, wd := range cnWeekdays {
			if strings.Contains(expr, day) {
				targetWeekday = wd
				found = true
				break
			}
		}
	}

	if !found {
		return time.Time{}, false
	}

	currentWeekday := now.Weekday()

	// Check for next/last modifiers
	if strings.Contains(expr, "next") || strings.Contains(expr, "下周") {
		daysAhead := int(targetWeekday-currentWeekday+7) % 7
		if daysAhead == 0 {
			daysAhead = 7
		}
		return now.AddDate(0, 0, daysAhead), true
	}
	if strings.Contains(expr, "last") || strings.Contains(expr, "上周") {
		daysBack := int(currentWeekday-targetWeekday+7) % 7
		if daysBack == 0 {
			daysBack = 7
		}
		return now.AddDate(0, 0, -daysBack), true
	}

	// Default to this week
	daysAhead := int(targetWeekday-currentWeekday+7) % 7
	return now.AddDate(0, 0, daysAhead), true
}

// parseWeekdayInPeriod parses expressions like "Tuesday in next week", "周五在下周", "下周三"
func parseWeekdayInPeriod(expr string, now time.Time) (time.Time, bool) {
	weekdays := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	cnWeekdays := map[string]time.Weekday{
		"周日": time.Sunday, "星期日": time.Sunday,
		"周一": time.Monday, "星期一": time.Monday,
		"周二": time.Tuesday, "星期二": time.Tuesday,
		"周三": time.Wednesday, "星期三": time.Wednesday,
		"周四": time.Thursday, "星期四": time.Thursday,
		"周五": time.Friday, "星期五": time.Friday,
		"周六": time.Saturday, "星期六": time.Saturday,
	}

	var targetWeekday time.Weekday
	var found bool

	for day, wd := range weekdays {
		if strings.Contains(expr, day) {
			targetWeekday = wd
			found = true
			break
		}
	}

	if !found {
		for day, wd := range cnWeekdays {
			if strings.Contains(expr, day) {
				targetWeekday = wd
				found = true
				break
			}
		}
	}

	if !found {
		return time.Time{}, false
	}

	var periodStart time.Time
	var periodFound bool

	if strings.Contains(expr, "next week") || strings.Contains(expr, "in next week") || regexp.MustCompile(`下(?:个)?周`).MatchString(expr) {
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		periodStart = now.AddDate(0, 0, daysUntilSunday)
		periodFound = true
	} else if strings.Contains(expr, "this week") || strings.Contains(expr, "in this week") || regexp.MustCompile(`本(?:个)?周`).MatchString(expr) {
		periodStart = now
		periodFound = true
	} else if strings.Contains(expr, "last week") || strings.Contains(expr, "in last week") || regexp.MustCompile(`上(?:个)?周`).MatchString(expr) {
		daysBack := int(now.Weekday()) + 7
		periodStart = now.AddDate(0, 0, -daysBack)
		periodFound = true
	}

	if !periodFound {
		return time.Time{}, false
	}

	daysToWeekday := int(targetWeekday-periodStart.Weekday()+7) % 7
	result := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day()+daysToWeekday, now.Hour(), now.Minute(), 0, 0, now.Location())
	return result, true
}

// parseDayInMonth parses expressions like "25th day in next month", "下个月的25号"
func parseDayInMonth(expr string, now time.Time) (time.Time, bool) {
	var dayOfMonth int
	var monthOffset int
	var found bool

	year := now.Year()
	month := now.Month()

	if regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*next\s*month`).MatchString(expr) {
		re := regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*next\s*month`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = 1
			found = true
		}
	} else if regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*this\s*month`).MatchString(expr) {
		re := regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*this\s*month`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = 0
			found = true
		}
	} else if regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*last\s*month`).MatchString(expr) {
		re := regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)?\s*day\s*in\s*last\s*month`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = -1
			found = true
		}
	} else if regexp.MustCompile(`下(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`).MatchString(expr) {
		re := regexp.MustCompile(`下(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = 1
			found = true
		}
	} else if regexp.MustCompile(`本(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`).MatchString(expr) {
		re := regexp.MustCompile(`本(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = 0
			found = true
		}
	} else if regexp.MustCompile(`上(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`).MatchString(expr) {
		re := regexp.MustCompile(`上(?:个)?月(?:的|在)?(\d{1,2})(?:号|日)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			dayOfMonth, _ = strconv.Atoi(matches[1])
			monthOffset = -1
			found = true
		}
	}

	if !found || dayOfMonth < 1 || dayOfMonth > 31 {
		return time.Time{}, false
	}

	adjustedMonth := int(month) + monthOffset
	adjustedYear := year

	if adjustedMonth > 12 {
		adjustedYear++
		adjustedMonth -= 12
	} else if adjustedMonth < 1 {
		adjustedYear--
		adjustedMonth += 12
	}

	lastDayOfMonth := time.Date(adjustedYear, time.Month(adjustedMonth+1), 0, 0, 0, 0, 0, now.Location()).Day()
	if dayOfMonth > lastDayOfMonth {
		dayOfMonth = lastDayOfMonth
	}

	result := time.Date(adjustedYear, time.Month(adjustedMonth), dayOfMonth, now.Hour(), now.Minute(), 0, 0, now.Location())
	return result, true
}

// ToolResponse represents the response from a tool execution
type ToolResponse struct {
	Tool   string                 `json:"tool"`
	Result string                 `json:"result"`
	Error  string                 `json:"error,omitempty"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

// ToolExecutionResult represents the result of a tool execution
type ToolExecutionResult struct {
	Result      string                 `json:"result"`
	Error       string                 `json:"error,omitempty"`
	StopCommand *ToolStopCommand       `json:"stop_command,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolStopCommand represents a command to stop the agent loop and respond to user
type ToolStopCommand struct {
	Response string                 `json:"response"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MarshalJSON implements json.Marshaler for Tool
func (t *Tool) MarshalJSON() ([]byte, error) {
	type Alias Tool
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	})
}

func formatMap(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}
