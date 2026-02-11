# 工具（Tools）实现说明

## 📁 核心文件结构

```
pkg/modules/ai/
├── tools.go          # 工具定义和管理
├── agent.go          # Agent 核心逻辑（调用工具）
├── config.go         # 配置管理
├── skills.go         # 技能管理
└── ...
```

## 🔧 工具实现详解

### 1. 工具定义结构

**位置**: `pkg/modules/ai/tools.go`（第9-15行）

```go
type Tool struct {
    Name        string                 `json:"name"`        // 工具名称，唯一标识
    Description string                 `json:"description"` // 工具描述（给 LLM 看）
    Parameters  map[string]interface{} `json:"parameters"` // JSON Schema 格式的参数定义
    Execute     func(ctx *AgentContext, params map[string]interface{}) (string, error) `json:"-"`
}
```

**说明**:
- `Name`: 工具的唯一标识符，用于 LLM 选择和 Agent 调用
- `Description`: 描述工具的功能，帮助 LLM 理解何时使用
- `Parameters`: JSON Schema 格式的参数定义，符合 OpenAI Function Calling 规范
- `Execute`: 实际执行逻辑，接收 AgentContext 和参数，返回结果

### 2. 工具管理器

**位置**: `pkg/modules/ai/tools.go`（第17-133行）

```go
type ToolManager struct {
    tools map[string]*Tool    // 注册的工具字典
    mu    sync.RWMutex         // 并发安全锁
}

// 核心方法
- RegisterTool(tool)      // 注册新工具
- UnregisterTool(name)    // 移除工具
- GetTool(name)          // 获取单个工具
- GetAllTools()          // 获取所有工具
- GetEnabledTools()      // 获取启用的工具（根据配置）
- GetToolDefinitions() // 获取 LLM 可用的工具定义
- ExecuteTool(name, ctx, params) // 执行工具
```

### 3. Agent 如何调用工具

**位置**: `pkg/modules/ai/agent.go`（第64-200行）

```go
func (a *Agent) runAgentLoop(ctx context.Context, agentCtx *AgentContext, userMessage string) (*AgentResponse, error) {
    for i := 0; i < maxIterations; i++ {
        // 1. 构建提示词
        prompt := a.buildPrompt(agentCtx, userMessage, i)

        // 2. 获取启用的工具
        tools := a.toolManager.GetEnabledTools()

        // 3. 获取工具定义（给 LLM 用）
        toolDefinitions := a.toolManager.GetToolDefinitions()

        // 4. 让 LLM 选择并调用工具
        if len(tools) > 0 {
            llmResponse, err = a.llmProvider.GenerateWithTools(ctx, prompt, toolDefinitions)
        } else {
            llmResponse, err = a.llmProvider.Generate(ctx, prompt)
        }

        // 5. 解析 LLM 返回的工具调用
        toolCall, err := a.parseToolCall(llmResponse)

        // 6. 如果没有工具调用，返回响应
        if toolCall == nil {
            return &AgentResponse{Content: llmResponse, ...}
        }

        // 7. 执行工具
        result, err := a.toolManager.ExecuteTool(toolCall.Name, agentCtx, toolCall.Input)

        // 8. 记录执行步骤
        agentCtx.ExecutionSteps = append(agentCtx.ExecutionSteps, ExecutionStep{
            StepNumber: i + 1,
            Thought:    llmResponse,
            Action:     toolCall.Name,
            Input:      toolCall.Input,
            Output:     result,
        })
    }
}
```

**关键点**:
- Agent 使用 `GenerateWithTools` 方法将工具定义传递给 LLM
- LLM 返回工具调用请求（格式：`TOOL: tool_name\nINPUT: {json}`）
- Agent 解析工具调用，提取工具名和参数
- Agent 使用 `ExecuteTool` 执行工具
- 所有执行步骤被记录，用于调试和追踪

### 4. 默认工具实现

**位置**: `pkg/modules/ai/tools.go`（第136-263行）

#### 工具 1: navigate - 页面导航

```go
navigationTool := &Tool{
    Name:        "navigate",
    Description: "Navigate to a specific page in the application",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "route_name": map[string]interface{}{
                "type":        "string",
                "description": "The name of the route to navigate to (e.g., 'projects.index', 'tasks.range')",
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

        // 设置导航信息
        ctx.NavigationInfo = &NavigationInfo{
            RouteName: routeName,
            Params:    routeParams,
        }
        ctx.ShouldNavigate = true

        return fmt.Sprintf("Navigating to %s", routeName), nil
    },
}
```

#### 工具 2: create_task - 创建任务

```go
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

        var projectID int64
        if pid, ok := params["project_id"].(float64); ok {
            projectID = int64(pid)
        }

        // 这里应该调用实际的 task 创建逻辑
        // 目前只是返回模拟结果
        result := fmt.Sprintf("Created task: %s", title)
        if description != "" {
            result += fmt.Sprintf("\nDescription: %s", description)
        }
        if projectID > 0 {
            result += fmt.Sprintf("\nProject ID: %d", projectID)
        }

        return result, nil
    },
}
```

#### 工具 3: search - 搜索功能

```go
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

        // 这里应该调用实际的搜索逻辑
        // 目前只是返回模拟结果
        return fmt.Sprintf("Searching for %s: %s", searchType, query), nil
    },
}
```

### 5. 工具注册

**位置**: `pkg/modules/ai/tools.go`（第231行）

```go
func RegisterDefaultTools() error {
    tm := GetToolManager()

    // 注册三个默认工具
    if err := tm.RegisterTool(navigationTool); err != nil {
        return fmt.Errorf("failed to register navigation tool: %w", err)
    }
    if err := tm.RegisterTool(createTaskTool); err != nil {
        return fmt.Errorf("failed to register create_task tool: %w", err)
    }
    if err := tm.RegisterTool(searchTool); err != nil {
        return fmt.Errorf("failed to register search tool: %w", err)
    }

    return nil
}
```

### 6. 配置过滤

**位置**: `pkg/modules/ai/tools.go`（第103-119行）

```go
func (tm *ToolManager) GetEnabledTools() []*Tool {
    cfg := GetConfig()

    // 如果 enabled_tools 为空，返回所有工具
    if len(cfg.EnabledTools) == 0 {
        return tm.GetAllTools()
    }

    // 只返回在 enabled_tools 列表中的工具
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
```

**配置方式**（在 `config.yml` 中）:
```yaml
ai:
  enabled_tools:           # 空 = 所有工具都启用
    # 或
    enabled_tools:
      - navigate      # 只启用导航工具
      - create_task    # 只启用创建任务工具
```

## 🎯 工具调用流程图

```
用户消息
    ↓
1. Agent 接收消息
    ↓
2. 构建 Prompt（包含工具定义）
    ↓
3. 调用 LLM.GenerateWithTools(prompt, toolDefinitions)
    ↓
4. LLM 分析并决定是否使用工具
    ↓
    ┌─────────────────┴
    │               │
    ↓               ↓
5. 解析工具调用    6. 执行工具
    ↓               ↓
    ↓               ↓
7. 记录执行步骤    8. 返回结果
    ↓               ↓
```

## 📋 当前工具总结

| 工具名 | 功能 | 参数 | 当前实现状态 |
|--------|------|------|------------|
| navigate | 页面导航 | route_name (必填), params (可选) | ✅ 完整实现 |
| create_task | 创建任务 | title (必填), description (可选), project_id (可选) | ⚠️  模拟实现 |
| search | 搜索 | query (必填), type (可选: tasks/projects/teams) | ⚠️  模拟实现 |

**注意**：
- `create_task` 和 `search` 目前是模拟实现，返回字符串而不是真正执行操作
- 需要集成实际的业务逻辑（调用相应的 API 或数据库操作）
