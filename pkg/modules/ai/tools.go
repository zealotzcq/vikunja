package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/user"
)

// Tool represents a function that the agent can call
type Tool struct {
	Name        string                                                                               `json:"name"`
	Description string                                                                               `json:"description"`
	Parameters  map[string]interface{}                                                               `json:"parameters"`
	Execute     func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) `json:"-"`
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
		Name: "question",
		Description: `Use this tool when you need to ask the user questions during execution. This allows you to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices as you work
4. Offer choices to the user about what direction to take.

Usage notes:
- When custom is enabled (default), a "Type your own answer" option is added automatically; don't include "Other" or catch-all options
- Answers are returned as arrays of labels; set multiple: true to allow selecting more than one
- If you recommend a specific option, make that the first option in the list and add "(Recommended)" at the end of the label`,
		Parameters: map[string]interface{}{
			"$schema": "https://json-schema.org/draft-2020-12/schema",
			"type":    "object",
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
					Error: "questions is required",
				}, fmt.Errorf("questions is required")
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
				Result: "Question sent to user",
				StopCommand: &ToolStopCommand{
					Response: "I need some information from you to proceed",
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
		Name: "show_navigation",
		Description: `Display a navigation button for user to navigate to various locations.

Use this tool when you want to provide a button that allows users to navigate to:
- Tasks (route: task.detail with param id)
- Projects (route: project.index with param projectId)
- Teams (route: teams.edit with param id)
- Labels (route: labels.index)
- Task list (route: tasks.range)
- Project list (route: projects.index)
- Team list (route: teams.index)
- Home (route: home)

The button will display a label and optionally a title showing the target (e.g., task title, project title).

When auto_navigate is true, the page will automatically navigate to the target location in addition to showing the button.
When auto_navigate is false (default), only the button is shown without automatic navigation.

Parameters:
- route_name (required): The name of route to navigate to
- params (optional): Route parameters (e.g., id for task, projectId for project)
- label (required): The button label text (e.g., taskid,projectid)
- title (optional): The title of the target entity to display after the label (e.g., taskid: task title, projectid: project title)
- auto_navigate (optional): If true, automatically navigate to the target location; if false (default), only show the button

Example usage:
- Show task button: route_name="task.detail", params={"id": 123}, label="查看任务123", title="任务123:写报告", auto_navigate=false
- Auto-navigate to project: route_name="project.index", params={"projectId": 456}, label="查看项目456", title="项目456:盘古计划", auto_navigate=true`,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"route_name": map[string]interface{}{
					"type":        "string",
					"description": "The route name to navigate to (e.g., task.detail, project.index, teams.edit, labels.index, tasks.range, projects.index, teams.index, home)",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Optional route parameters (e.g., id for task, projectId for project)",
				},
				"label": map[string]interface{}{
					"type":        "string",
					"description": "The button label text (required)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the target entity to display after the label (optional, e.g., task title, project title)",
				},
				"auto_navigate": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, automatically navigate to the target location; if false (default), only show the button",
				},
			},
			"required": []string{"route_name", "label"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
			routeName, ok := params["route_name"].(string)
			if !ok || routeName == "" {
				return &ToolExecutionResult{
					Error: "route_name is required",
				}, fmt.Errorf("route_name is required")
			}

			label, ok := params["label"].(string)
			if !ok || label == "" {
				return &ToolExecutionResult{
					Error: "label is required",
				}, fmt.Errorf("label is required")
			}

			var routeParams map[string]interface{}
			if p, ok := params["params"].(map[string]interface{}); ok {
				routeParams = p
			}

			var title string
			if t, ok := params["title"].(string); ok {
				title = t
			}

			autoNavigate := false
			if an, ok := params["auto_navigate"].(bool); ok {
				autoNavigate = an
			}

			ctx.ButtonNavigation = &chat_session.ButtonNavigation{
				RouteName: routeName,
				Params:    routeParams,
				Label:     label,
				Title:     title,
			}

			if autoNavigate {
				ctx.NavigationInfo = &NavigationInfo{
					RouteName: routeName,
					Params:    routeParams,
				}
				ctx.ShouldNavigate = true
			}

			return &ToolExecutionResult{
				Result: "Navigation button shown",
			}, nil
		},
	}

	if err := tm.RegisterTool(showNavigationTool); err != nil {
		return fmt.Errorf("failed to register show_navigation tool: %w", err)
	}

	finishTaskTool := &Tool{
		Name: "finish_job",
		Description: `Call this tool when you have completed your work and want to respond to the user. This is the ONLY tool that ends the conversation.

This tool sends your response to the user and optionally provides navigation information.

Available routes for navigation:
- home: Homepage/Dashboard
- projects.index: Project list page
- project.index: Specific project detail (requires projectId in params). Use projectId: -1 for favorites
- tasks.range: Task list page
- task.detail: Specific task detail (requires id in params)
- teams.index: Team list page
- teams.edit: Specific team detail/edit (requires id in params)
- labels.index: Label list page

IMPORTANT: Use set_button_navigation tool BEFORE calling finish_job if you want to provide a button for the user to click.

Parameters:
- content (required): Your response message to the user
- route_name (optional): Route name to navigate to
- params (optional): Route parameters for navigation

IMPORTANT: You MUST use this tool to end the conversation. Do not provide text responses without calling this tool.`,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Your response message to the user. This is REQUIRED.",
				},
				"route_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the route to navigate to (optional)",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Optional route parameters (e.g., projectId, id)",
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

			metadata := make(map[string]interface{})

			routeName, hasRoute := params["route_name"].(string)
			if hasRoute {
				var routeParams map[string]interface{}
				if p, ok := params["params"].(map[string]interface{}); ok {
					routeParams = p
				}
				ctx.NavigationInfo = &NavigationInfo{
					RouteName: routeName,
					Params:    routeParams,
				}
				ctx.ShouldNavigate = true
				metadata["navigation"] = true
				metadata["route"] = routeName
				metadata["params"] = routeParams
			}

			return &ToolExecutionResult{
				Result: content,
				StopCommand: &ToolStopCommand{
					Response: content,
					Metadata: metadata,
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(finishTaskTool); err != nil {
		return fmt.Errorf("failed to register finish_job tool: %w", err)
	}

	skillTool := &Tool{
		Name:        "skill",
		Description: `Load a skill to get detailed instructions for a specific task. Skills provide specialized knowledge and step-by-step guidance. Use this when a task matches an available skill's description. Only the skills listed here are available: ` + sm.FormatSkillsForTool(),
		Parameters: map[string]interface{}{
			"$schema": "https://json-schema.org/draft-2020-12/schema",
			"type":    "object",
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

	assignTaskTool := &Tool{
		Name: "assign_task",
		Description: `Assign a task to a subordinate staff member. Use this when the user wants to assign work or a task to someone.

The system context contains subordinate staff information including:
- User ID, Username, Name, and Project ID for each subordinate

Priority determination (based on user's tone/phrasing):
- HIGH priority: When user says "马上", "立即", "尽快", "urgent", "immediately", etc.
- MEDIUM priority (default): Normal tone without urgency indicators
- LOW priority: When user says "有空", "有时间", "不急", "when convenient", "no rush", etc.

Due date calculation:
- HIGH priority: 1 day from now
- MEDIUM priority: 3 days from now
- LOW priority: 7 days from now

Task properties:
- Start date: Now (current time)
- IsFavorite: true (favorited by default)
- Subscription: Subscribed to task notifications

Example usage:
- "让小王马上写报告" -> HIGH priority, due in 1 day
- "叫李四有空的时候整理文档" -> LOW priority, due in 7 days
- "给张三安排个任务" -> MEDIUM priority, due in 3 days`,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_identifier": map[string]interface{}{
					"type":        "integer",
					"description": "The user ID (integer) of the subordinate staff member to assign the task to. Must match a subordinate in the context.",
				},
				"task_title": map[string]interface{}{
					"type":        "string",
					"description": "The title or description of the task to assign",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority level: 'high', 'medium', or 'low'. Default is 'medium' if not specified.",
					"enum":        []string{"high", "medium", "low"},
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

			if targetStaff.ProjectID <= 0 {
				return &ToolExecutionResult{
					Error: fmt.Sprintf("Staff member %s does not have an associated project", targetStaff.Username),
				}, fmt.Errorf("staff member %s does not have an associated project", targetStaff.Username)
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

			now := time.Now()
			dueDate := now.AddDate(0, 0, daysToAdd)

			task := &models.Task{
				Title:      taskTitle,
				ProjectID:  targetStaff.ProjectID,
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

			response := fmt.Sprintf("已成功为 %s 分配任务：%s\n", displayName, taskTitle)
			response += fmt.Sprintf("- 优先级：%s\n", map[string]string{"high": "高", "medium": "中", "low": "低"}[priority])
			response += fmt.Sprintf("- 截止日期：%s\n", dueDate.Format("2006-01-02"))

			ctx.ButtonNavigation = &chat_session.ButtonNavigation{
				RouteName: "task.detail",
				Params: map[string]interface{}{
					"id": task.ID,
				},
				Label: "查看任务",
				Title: taskTitle,
			}

			return &ToolExecutionResult{
				Result: response,
				Metadata: map[string]interface{}{
					"task_id":     task.ID,
					"assigned_to": targetStaff.UserID,
					"project_id":  targetStaff.ProjectID,
					"priority":    priority,
					"due_date":    dueDate.Format(time.RFC3339),
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(assignTaskTool); err != nil {
		return fmt.Errorf("failed to register assign_task tool: %w", err)
	}

	return nil
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
