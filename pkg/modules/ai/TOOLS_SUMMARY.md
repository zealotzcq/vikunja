# 工具实现总结

## 📁 核心文件

### 1. 工具定义和管理
- **`pkg/modules/ai/tools.go`** (283 行)
  - `Tool` 结构体（工具定义）
  - `ToolManager` 结构体（工具管理器）
  - `RegisterDefaultTools()` 函数（注册默认工具）

### 2. Agent 核心逻辑
- **`pkg/modules/ai/agent.go`**
  - `runAgentLoop()` 函数（执行循环）
  - `parseToolCall()` 函数（解析工具调用）
  - 工具执行和步骤记录

### 3. 配置管理
- **`pkg/modules/ai/config.go`** (127 行)
  - 从 `config.yml` 读取配置
  - `GetConfig()` 函数
  - `ValidateConfig()` 函数

## 🎯 当前工具

| 工具名 | 功能 | 参数 | 状态 |
|--------|------|------|------|
| `navigate` | 页面导航 | `route_name` (必填), `params` (可选) | ✅ 完整 |
| `create_task` | 创建任务 | `title` (必填), `description` (可选), `project_id` (可选) | ⚠️ 模拟 |
| `search` | 搜索 | `query` (必填), `type` (可选) | ⚠️ 模拟 |

## 🔧 工具配置方式

### 1. 启用所有工具（默认）

```yaml
ai:
  enabled_tools:    # 空 = 启用所有工具
```

### 2. 只启用特定工具

```yaml
ai:
  enabled_tools:
    - navigate
    - create_task
```

### 3. 启用本地命令工具

```yaml
ai:
  enabled_tools:
    - run_command
```

## 📝 如何添加新工具

### 步骤 1：定义工具

```go
var MyNewTool = &Tool{
    Name:        "my_new_tool",
    Description: "Description of what this tool does",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type": "string",
                "description": "Description of param1",
            },
        },
        "required": []string{"param1"},
    },
    Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
        // 实现工具逻辑
        param1, _ := params["param1"].(string)
        result := fmt.Sprintf("Processed: %s", param1)
        return result, nil
    },
}
```

### 步骤 2：注册工具

在 `RegisterDefaultTools()` 函数中添加：

```go
func RegisterDefaultTools() error {
    tm := GetToolManager()

    // 注册默认工具
    if err := tm.RegisterTool(navigationTool); err != nil {
        return fmt.Errorf("failed to register navigation tool: %w", err)
    }
    if err := tm.RegisterTool(createTaskTool); err != nil {
        return fmt.Errorf("failed to register create_task tool: %w", err)
    }
    if err := tm.RegisterTool(searchTool); err != nil {
        return fmt.Errorf("failed to register search tool: %w", err)
    }

    // 注册你的新工具
    if err := tm.RegisterTool(MyNewTool); err != nil {
        return fmt.Errorf("failed to register my_new_tool: %w", err)
    }

    return nil
}
```

### 步骤 3：在 config.yml 中启用（可选）

```yaml
ai:
  enabled_tools:
    - my_new_tool
```

## 🚨 本地命令行工具

### 当前实现状态

❌ **当前不支持**：现有代码中还没有实现本地命令行工具

### 添加方式

1. **查看 `LOCAL_COMMAND_TOOLS.md`** - 详细的实现指南和安全建议
2. **实现安全的命令执行工具** - 使用白名单或配置验证
3. **在 config.yml 中配置允许的命令** - 限制可执行的命令
4. **在 config.yml 中启用工具** - `enabled_tools: ["run_command"]`

### 安全建议

⚠️ **重要**：在生产环境中谨慎使用本地命令工具

1. **使用白名单**：只允许执行预定义的安全命令
2. **验证命令**：在执行前验证命令格式和内容
3. **限制工作目录**：防止访问敏感目录
4. **超时控制**：为命令执行设置超时（如 30 秒）
5. **审计日志**：记录所有命令执行

### 示例实现

```go
var SafeRunCommandTool = &Tool{
    Name:        "run_command",
    Description: "Execute approved local commands",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "command": map[string]interface{}{
                "type":        "string",
                "description": "The command to execute",
                "enum": []string{
                    "git status",
                    "git log",
                    "ls -la",
                    "go test ./...",
                },
            },
        },
        "required": []string{"command"},
    },
    Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
        command, ok := params["command"].(string)
        if !ok {
            return "", fmt.Errorf("command is required")
        }

        // 验证命令是否在白名单中
        if !isCommandAllowed(command) {
            return fmt.Sprintf("Error: Command '%s' is not allowed", command), nil
        }

        // 执行命令（带超时）
        cmd := exec.Command("sh", "-c", command)
        cmd.Timeout = 30 * time.Second  // 30 秒超时

        output, err := cmd.CombinedOutput()
        if err != nil {
            return fmt.Sprintf("Error: %v", err), nil
        }

        return string(output), nil
    },
}

func isCommandAllowed(command string) bool {
    allowed := []string{
        "git status",
        "git log",
        "ls -la",
        "go test ./...",
    }
    for _, a := range allowed {
        if command == a {
            return true
        }
    }
    return false
}
```

## 🎯 快速配置指南

### 启用所有工具

```yaml
ai:
  enabled_tools:    # 空 = 全部启用
```

### 只启用导航和创建任务

```yaml
ai:
  enabled_tools:
    - navigate
    - create_task
```

### 启用本地命令（需谨慎）

```yaml
ai:
  enabled_tools:
    - run_command

  # 可选：配置允许的命令白名单
  allowed_commands:
    - "git status"
    - "ls -la"
```

## 📋 完整实现流程

```
1. 定义工具（在 tools.go 中）
   ↓
2. 注册工具（在 RegisterDefaultTools 中）
   ↓
3. 在 config.yml 中启用（enabled_tools）
   ↓
4. 重启后端服务
   ↓
5. 前端发送消息（use_agent: true）
   ↓
6. Agent 选择工具并执行
   ↓
7. 返回结果给前端
```

## 🚀 总结

- ✅ 工具系统已完全实现（定义、管理、执行）
- ✅ Agent 已集成工具调用
- ✅ 支持从 config.yml 配置启用的工具
- ✅ 本地命令工具需要手动实现（参考 LOCAL_COMMAND_TOOLS.md）
- ⚠️ 生产环境需谨慎使用本地命令工具

## 相关文档

- `TOOLS_IMPLEMENTATION.md` - 详细工具实现说明
- `LOCAL_COMMAND_TOOLS.md` - 本地命令工具配置和安全指南
- `README.md` - AI Agent 使用文档
- `config.yml` - 配置文件（项目根目录）
