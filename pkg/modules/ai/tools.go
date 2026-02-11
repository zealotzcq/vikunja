package ai

import (
	"encoding/json"
	"fmt"
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

	return tool.Execute(ctx, params)
}

// RegisterDefaultTools registers the default set of tools
func RegisterDefaultTools() error {
	tm := GetToolManager()

	navigationTool := &Tool{
		Name:        "navigate",
		Description: "Navigate to a specific page in the application",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"route_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the route to navigate to (e.g., 'projects.index', 'tasks.range', 'teams.index')",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Optional route parameters",
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

			ctx.NavigationInfo = &NavigationInfo{
				RouteName: routeName,
				Params:    routeParams,
			}
			ctx.ShouldNavigate = true

			return fmt.Sprintf("Navigating to %s", routeName), nil
		},
	}

	if err := tm.RegisterTool(navigationTool); err != nil {
		return fmt.Errorf("failed to register navigation tool: %w", err)
	}

	createTaskTool := &Tool{
		Name:        "create_task",
		Description: "Create a new task",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the task",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "The description of the task (optional)",
				},
				"project_id": map[string]interface{}{
					"type":        "integer",
					"description": "The project ID to add the task to (optional)",
				},
			},
			"required": []string{"title"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
			title, ok := params["title"].(string)
			if !ok {
				return "", fmt.Errorf("title is required")
			}

			var description string
			if d, ok := params["description"].(string); ok {
				description = d
			}

			result := fmt.Sprintf("Created task: %s", title)
			if description != "" {
				result += fmt.Sprintf("\nDescription: %s", description)
			}

			return result, nil
		},
	}

	if err := tm.RegisterTool(createTaskTool); err != nil {
		return fmt.Errorf("failed to register create_task tool: %w", err)
	}

	searchTool := &Tool{
		Name:        "search",
		Description: "Search for tasks, projects, or teams",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "The type of search (tasks, projects, teams)",
					"enum":        []string{"tasks", "projects", "teams"},
				},
			},
			"required": []string{"query", "type"},
		},
		Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
			query, ok := params["query"].(string)
			if !ok {
				return "", fmt.Errorf("query is required")
			}

			searchType, ok := params["type"].(string)
			if !ok {
				searchType = "tasks"
			}

			return fmt.Sprintf("Searching for %s: %s", searchType, query), nil
		},
	}

	if err := tm.RegisterTool(searchTool); err != nil {
		return fmt.Errorf("failed to register search tool: %w", err)
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
