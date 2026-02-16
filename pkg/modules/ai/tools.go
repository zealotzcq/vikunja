package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
		Name:           "show_navigation",
		ShouldStopLoop: true,
		Description: `Display a navigation button for user to navigate to various locations.

Use this tool when you want to provide a button that allows users to navigate to:
- Tasks (route: task.detail with param id)
- Projects (route: project.index with param projectId)
- Project list (route: projects.index)
- Home (route: home) - all task list, higher priority than upcoming page
- Favourite task list (route: project.index with param projectId = -1)
- Upcomming task list (route: tasks.range with param showNulls=true)

The button will display a label and optionally a title showing the target (e.g., task title, project title).

Parameters:
- route_name (required): The name of route to navigate to
- params (optional): Route parameters (e.g., id for task, projectId for project)
- label (required): The button label text (e.g., taskid,projectid)
- title (optional): The title of the target entity to display after the label (e.g., taskid: task title, projectid: project title)

Example usage:
- Show task button: route_name="task.detail", params={"id": 123}, label="查看任务123", title="任务123:写报告"
- Navigate to project: route_name="project.index", params={"projectId": 456}, label="查看项目456", title="项目456:盘古计划"
- Navigate to home: route_name="home", label="回到主页"`,
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

			ctx.ButtonNavigation = &chat_session.ButtonNavigation{
				RouteName: routeName,
				Params:    routeParams,
				Label:     label,
				Title:     title,
			}

			autoNavigate := routeName != "task.detail"

			if autoNavigate {
				ctx.NavigationInfo = &NavigationInfo{
					RouteName: routeName,
					Params:    routeParams,
				}
				ctx.ShouldNavigate = true
			}

			return &ToolExecutionResult{
				Result: "好的",
			}, nil
		},
	}

	if err := tm.RegisterTool(showNavigationTool); err != nil {
		return fmt.Errorf("failed to register show_navigation tool: %w", err)
	}

	finishTaskTool := &Tool{
		Name:           "finish_job",
		ShouldStopLoop: true,
		Description: `Call this tool when you have completed your work and want to respond to the user. This is the ONLY tool that ends the conversation.

This tool sends your response to the user.

IMPORTANT: You MUST use this tool to end the conversation. Do not provide text responses without calling this tool.`,
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

	if err := tm.RegisterTool(finishTaskTool); err != nil {
		return fmt.Errorf("failed to register finish_job tool: %w", err)
	}

	if len(sm.GetAllSkills()) > 0 {
		skillTool := &Tool{
			Name:           "skill",
			ShouldStopLoop: false,
			Description:    `Load a skill to get detailed instructions for a specific task. Skills provide specialized knowledge and step-by-step guidance. Use this when a task matches an available skill's description. Only the skills listed here are available: ` + sm.FormatSkillsForTool(),
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
	}

	assignTaskTool := &Tool{
		Name:           "assign_task",
		ShouldStopLoop: true,
		Description: `Assign a task to a staff member or the user himself. Use this when user wants to assign work or a task to someone.

The system context contains information of subordinate staffs and the user, including:
- User ID, Username, Name, and Project ID for each subordinate

Priority determination (based on user's tone/phrasing):
- HIGH priority: When user says "马上", "立即", "尽快", "urgent", "immediately", etc.
- MEDIUM priority (default): Normal tone without urgency indicators
- LOW priority: When user says "有空", "有时间", "不急", "when convenient", "no rush", etc.

Due date calculation:
- If user specifies a time expression (e.g., "today", "tomorrow", "下周五", "2024-12-25"), use that expression
- If NO time expression is specified, use priority-based calculation:
  * HIGH priority: 1 day from now
  * MEDIUM priority: 3 days from now
  * LOW priority: 7 days from now

Task properties:
- Start date: Now (current time)
- IsFavorite: true (favorited by default)
- Subscription: Subscribed to task notifications

Supported time expressions (fill in the time_expression parameter in English):
- Relative dates: "today", "tomorrow", "yesterday", "today", "tomorrow", "yesterday"
- Relative times: "in 2 hours", "30 minutes later", "after 3 days", "in 2 hours", "30 minutes later"
- Time periods: "next week", "this month", "last year", "next week", "this month", "next year"
- Weekdays: "next Monday", "last Friday", "next Friday", "last Monday", "Friday"
- Absolute dates: "2024-12-25", "December 25, 2024", "December 25"
- Combinations: "tomorrow at 3pm", "next Monday at 9am", "next Friday at 5pm"
注意:填充time_expression时,指定语言为英语,"大后天" 应该翻译成 "after 3 days"


Example usage:
- "让小王马上写报告" -> HIGH priority, no time_expr, due in 1 day
- "叫李四有空的时候整理文档" -> LOW priority, no time_expr, due in 7 days
- "给张三安排个任务，明天截止" -> MEDIUM priority, time_expr="tomorrow"
- "让小王下周五提交报告" -> MEDIUM priority, time_expr="Next Friday"
- "给李四分配任务，2小时后完成" -> HIGH priority, time_expr="two hours later"`,
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
			if timeExpressionUsed {
				response += fmt.Sprintf("- 截止日期：%s (根据时间表达式设定)\n", dueDate.Format("2006-01-02 15:04"))
			} else {
				response += fmt.Sprintf("- 截止日期：%s (根据优先级设定)\n", dueDate.Format("2006-01-02"))
			}

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
					"task_id":         task.ID,
					"assigned_to":     targetStaff.UserID,
					"project_id":      targetStaff.ProjectID,
					"priority":        priority,
					"due_date":        dueDate.Format(time.RFC3339),
					"time_expression": timeExpressionUsed,
				},
			}, nil
		},
	}

	if err := tm.RegisterTool(assignTaskTool); err != nil {
		return fmt.Errorf("failed to register assign_task tool: %w", err)
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
func parseTimeExpression(expr string, now time.Time) (time.Time, error) {
	expr = strings.ToLower(strings.TrimSpace(expr))

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
