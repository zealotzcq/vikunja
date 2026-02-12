package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Tool represents a function that the agent can call
type Tool struct {
	Name        string                                                                 `json:"name"`
	Description string                                                                 `json:"description"`
	Parameters  map[string]interface{}                                                 `json:"parameters"`
	Execute     func(ctx *AgentContext, params map[string]interface{}) (string, error) `json:"-"`
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
func (tm *ToolManager) ExecuteTool(name string, ctx *AgentContext, params map[string]interface{}) (string, error) {
	tm.mu.RLock()
	tool, exists := tm.tools[name]
	tm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}



	result, err := tool.Execute(ctx, params)



	return result, err
}

// RegisterDefaultTools registers the default set of tools
func RegisterDefaultTools() error {
	tm := GetToolManager()
	sm := GetSkillManager()

	navigationTool := &Tool{
		Name: "navigate",
		Description: `Navigate to a specific page in the application.

Available routes:
- home: Homepage/Dashboard
- projects.index: Project list page
- project.index: Specific project detail (requires projectId in params). Use projectId: -1 for favorites
- tasks.range: Task list page
- task.detail: Specific task detail (requires id in params)
- teams.index: Team list page
- teams.edit: Specific team detail/edit (requires id in params)
- labels.index: Label list page

Usage examples:
- Navigate to homepage: {"route_name": "home"}
- Navigate to projects: {"route_name": "projects.index"}
- Navigate to project 123: {"route_name": "project.index", "params": {"projectId": 123}}
- Navigate to favorites: {"route_name": "project.index", "params": {"projectId": -1}}
- Navigate to tasks: {"route_name": "tasks.range"}
- Navigate to task 456: {"route_name": "task.detail", "params": {"id": 456}}
- Navigate to teams: {"route_name": "teams.index"}
- Navigate to team 789: {"route_name": "teams.edit", "params": {"id": 789}}
- Navigate to labels: {"route_name": "labels.index"}`,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"route_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the route to navigate to",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Optional route parameters (e.g., projectId, id)",
				},
			},
			"required": []string{"route_name"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
			routeName, ok := params["route_name"].(string)
			if !ok {
				return "", fmt.Errorf("route_name is required")
			}

			var routeParams map[string]interface{}
			if p, ok := params["params"].(map[string]interface{}); ok {
				routeParams = p
			}

			var message string
			if routeName == "home" {
				message = "正在为您返回首页"
			} else if routeName == "projects.index" {
				message = "正在为您打开项目列表"
			} else if routeName == "project.index" {
				if pid, ok := routeParams["projectId"].(float64); ok {
					if int64(pid) == -1 {
						message = "正在为您打开收藏"
					} else {
						message = fmt.Sprintf("正在为您打开项目 %d", int64(pid))
					}
				} else {
					message = "正在为您打开项目详情"
				}
			} else if routeName == "tasks.range" {
				message = "正在为您打开任务列表"
			} else if routeName == "task.detail" {
				if tid, ok := routeParams["id"].(float64); ok {
					message = fmt.Sprintf("正在为您打开任务 %d", int64(tid))
				} else {
					message = "正在为您打开任务详情"
				}
			} else if routeName == "teams.index" {
				message = "正在为您打开团队列表"
			} else if routeName == "teams.edit" {
				if tid, ok := routeParams["id"].(float64); ok {
					message = fmt.Sprintf("正在为您打开团队 %d", int64(tid))
				} else {
					message = "正在为您打开团队详情"
				}
			} else if routeName == "labels.index" {
				message = "正在为您打开标签列表"
			} else {
				message = fmt.Sprintf("正在导航到 %s", routeName)
			}

			ctx.NavigationInfo = &NavigationInfo{
				Message:   message,
				RouteName: routeName,
				Params:    routeParams,
			}
			ctx.ShouldNavigate = true



			return message, nil
		},
	}

	if err := tm.RegisterTool(navigationTool); err != nil {
		return fmt.Errorf("failed to register navigation tool: %w", err)
	}

	skillTool := &Tool{
		Name: "skill",
		Description: sm.FormatSkillsForTool() + `

Use this tool to load a skill's full instructions when you need them. Call with:

{
  "name": "skill-name"
}

The skill's complete content will be returned, including all instructions, workflows, and additional details.`,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the skill to load",
				},
			},
			"required": []string{"name"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
			skillName, ok := params["name"].(string)
			if !ok {
				return "", fmt.Errorf("name is required")
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
				return "", fmt.Errorf("skill '%s' not found. Available skills: %s", skillName, available)
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

			return sb.String(), nil
		},
	}

	if err := tm.RegisterTool(skillTool); err != nil {
		return fmt.Errorf("failed to register skill tool: %w", err)
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
