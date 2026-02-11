# 配置本地命令行工具

## 📁 相关文件

实现本地命令行工具主要涉及以下文件：

1. **`pkg/modules/ai/tools.go`** - 工具定义和注册
2. **`pkg/modules/ai/config.go`** - 配置管理（读取 config.yml）
3. **`config.yml`** - 工具配置

## 🎯 实现步骤

### 步骤 1：创建本地命令工具

在 `pkg/modules/ai/tools.go` 中添加新工具：

```go
import (
	"fmt"
	"os/exec"
	"strings"
)

// RunCommandTool 执行本地命令
var RunCommandTool = &Tool{
	Name:        "run_command",
	Description: "Execute a local shell command",
	Parameters: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute (e.g., 'ls -la', 'git status')",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for the command (optional, defaults to project root)",
			},
		},
		"required": []string{"command"},
	},
	Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
		command, ok := params["command"].(string)
		if !ok {
			return "", fmt.Errorf("command is required")
		}

		// 获取工作目录
		workingDir := "."
		if wd, ok := params["working_dir"].(string); ok && wd != "" {
			workingDir = wd
		}

		// 执行命令
		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = workingDir

		// 执行并捕获输出
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error executing command: %v\nCommand: %s\nStderr: %s", err, command, string(output)), nil
		}

		return fmt.Sprintf("Command executed successfully\n%s\n%s", command, string(output)), nil
	},
}
```

### 步骤 2：注册本地命令工具

修改 `RegisterDefaultTools()` 函数：

```go
func RegisterDefaultTools() error {
	tm := GetToolManager()

	// 注册现有的工具
	if err := tm.RegisterTool(navigationTool); err != nil {
		return fmt.Errorf("failed to register navigation tool: %w", err)
	}
	if err := tm.RegisterTool(createTaskTool); err != nil {
		return fmt.Errorf("failed to register create_task tool: %w", err)
	}
	if err := tm.RegisterTool(searchTool); err != nil {
		return fmt.Errorf("failed to register search tool: %w", err)
	}

	// 注册新的本地命令工具
	if err := tm.RegisterTool(RunCommandTool); err != nil {
		return fmt.Errorf("failed to register run_command tool: %w", err)
	}

	return nil
}
```

### 步骤 3：更新 config.yml

在 `config.yml` 中添加配置：

```yaml
# AI 助手配置
ai:
  llm_provider: openai
  openai:
    key: sk-your-key
    model: gpt-4o-mini

  # 工具配置
  enabled_tools:
    - run_command    # 启用本地命令工具
    # 或留空以启用所有工具
```

## 📝 使用示例

### 示例 1：列出文件

**用户输入**: "列出当前目录的文件"

**LLM 工具调用**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "ls -la"
  }
}
```

**工具执行**: `ls -la` 命令在项目根目录执行

**Agent 响应**: "Command executed successfully\n..."

### 示例 2：查看 Git 状态

**用户输入**: "查看 git 状态"

**LLM 工具调用**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "git status"
  }
}
```

**工具执行**: `git status` 命令执行

**Agent 响应**: 显示 git 状态的输出

### 示例 3：运行测试

**用户输入**: "运行所有测试"

**LLM 工具调用**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "go test ./..."
  }
}
```

**工具执行**: 运行所有 Go 测试

**Agent 响应**: "Command executed successfully\n..."

### 示例 4：指定工作目录

**用户输入**: "在 frontend 目录运行构建"

**LLM 工具调用**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "pnpm build",
    "working_dir": "frontend"
  }
}
```

**工具执行**: 在 `frontend` 目录执行 `pnpm build`

**Agent 响应**: "Command executed successfully\n..."

## 🚨 安全注意事项

### 1. 命令注入防护

当前实现存在命令注入风险！用户可以执行任意命令。

**安全改进方案**:

#### 方案 A：白名单命令（推荐）

```go
var RunCommandTool = &Tool{
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
					"ls",
					"git diff",
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

		// 验证命令在白名单中
		allowedCommands := []string{
			"git status",
			"git log",
			"ls",
			"git diff",
			"go test ./...",
		}

		if !contains(allowedCommands, command) {
			return fmt.Sprintf("Error: Command '%s' is not allowed", command), nil
		}

		// 执行命令
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}

		return string(output), nil
	},
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

#### 方案 B：参数验证

限制命令只能使用特定的参数模式：

```go
Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	// 验证命令只包含安全的命令和参数
	if !isSafeCommand(command) {
		return fmt.Sprintf("Error: Command '%s' contains potentially dangerous operations", command), nil
	}

	// ... 执行命令
}

func isSafeCommand(command string) bool {
	// 只允许 git、ls、cat、grep 等安全命令
	// 不允许 rm、mv、cp 到敏感目录
	dangerous := []string{"rm", "mv", "cp", "dd", "format", "fdisk", "mkfs"}

	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return false
		}
	}

	return true
}
```

#### 方案 C：使用配置文件

在 `config.yml` 中定义允许的命令：

```yaml
ai:
  allowed_commands:
    - git status
    - git log
    - ls -la
    - go test ./...
```

在工具中读取配置并验证：

```go
Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	cfg := GetConfig()
	if !isCommandAllowed(cfg, command) {
		return fmt.Sprintf("Error: Command '%s' is not allowed", command), nil
	}

	// 执行命令
	// ...
}

func isCommandAllowed(cfg *Config, command string) bool {
	// 从配置读取允许的命令
	// 验证命令是否在允许列表中
	return true
}
```

## 🔧 配置方式

### 方式 1：启用所有工具（默认）

```yaml
ai:
  enabled_tools:    # 空 = 启用所有工具
    # 或
  enabled_tools:
    - run_command
```

### 方式 2：只启用特定工具

```yaml
ai:
  enabled_tools:
    - navigate       # 只允许页面导航
    - create_task    # 只允许创建任务
```

### 方式 3：启用本地命令工具

```yaml
ai:
  # 启用本地命令工具
  enabled_tools:
    - run_command
    - create_task
    - search
    - navigate

  # 可选：配置安全限制
  allowed_commands:
    - "git status"
    - "git log"
    - "ls -la"
    - "go test ./..."
```

## 🎯 完整实现示例

### 1. 创建安全的命令执行工具

```go
package ai

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

var (
	allowedCommands = []string{
		"git status",
		"git log",
		"git diff",
		"ls",
		"ls -la",
		"cat",
		"grep",
		"go test ./...",
		"go build ./...",
		"pnpm test",
		"pnpm build",
	}

	allowedMutex sync.RWMutex
)

// SafeRunCommandTool 安全的命令执行工具
var SafeRunCommandTool = &Tool{
	Name:        "run_command",
	Description: "Execute approved local commands",
	Parameters: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute (from allowed list)",
				"enum": allowedCommands,
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory (optional)",
			},
		},
		"required": []string{"command"},
	},
	Execute: func(ctx *AgentContext, params map[string]interface{}) (string, error) {
		command, ok := params["command"].(string)
		if !ok {
			return "", fmt.Errorf("command is required")
		}

		// 验证命令
		allowedMutex.RLock()
		allowed := isCommandAllowed(command)
		allowedMutex.RUnlock()

		if !allowed {
			return fmt.Sprintf("Error: Command '%s' is not allowed for security reasons", command), nil
		}

		// 获取工作目录
		workingDir := "."
		if wd, ok := params["working_dir"].(string); ok && wd != "" {
			workingDir = wd
		}

		// 执行命令
		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = workingDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error executing command: %v\nStderr: %s", err, string(output)), nil
		}

		return fmt.Sprintf("Executed: %s\nOutput:\n%s", command, string(output)), nil
	},
}

// isCommandAllowed 检查命令是否在允许列表中
func isCommandAllowed(command string) bool {
	for _, allowed := range allowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// UpdateAllowedCommands 更新允许的命令列表
func UpdateAllowedCommands(newCommands []string) {
	allowedMutex.Lock()
	defer allowedMutex.Unlock()
	allowedCommands = newCommands
}
```

### 2. 在 config.yml 中配置

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-your-api-key
    model: gpt-4o-mini

  # 启用工具
  enabled_tools:
    - run_command
    - navigate

  # 允许的命令（可选）
  allowed_commands:
    - "git status"
    - "git log -n 10"
    - "ls -la"
    - "go test ./pkg/modules/ai"
```

### 3. 注册工具

在 `RegisterDefaultTools()` 函数中添加：

```go
func RegisterDefaultTools() error {
	tm := GetToolManager()

	// 注册所有默认工具
	if err := tm.RegisterTool(navigationTool); err != nil {
		return fmt.Errorf("failed to register navigation tool: %w", err)
	}
	if err := tm.RegisterTool(createTaskTool); err != nil {
		return fmt.Errorf("failed to register create_task tool: %w", err)
	}
	if err := tm.RegisterTool(searchTool); err != nil {
		return fmt.Errorf("failed to register search tool: %w", err)
	}

	// 注册安全命令工具
	if err := tm.RegisterTool(SafeRunCommandTool); err != nil {
		return fmt.Errorf("failed to register run_command tool: %w", err)
	}

	return nil
}
```

## 🎯 使用场景

### 场景 1：查看项目文件

**用户**: "帮我查看项目根目录的文件"

**LLM 处理**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "ls -la"
  }
}
```

**输出**: 返回文件列表

### 场景 2：查看 Git 日志

**用户**: "显示最近的 git 提交记录"

**LLM 处理**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "git log -n 10"
  }
}
```

**输出**: 返回最近的 10 条提交记录

### 场景 3：运行 AI 模块测试

**用户**: "运行 AI 模块的所有测试"

**LLM 处理**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "go test ./pkg/modules/ai"
  }
}
```

**输出**: 运行测试并显示结果

### 场景 4：构建前端

**用户**: "帮我构建前端项目"

**LLM 处理**:
```json
{
  "tool": "run_command",
  "input": {
    "command": "pnpm build",
    "working_dir": "frontend"
  }
}
```

**输出**: 在 frontend 目录执行构建

## 📋 配置总结

| 配置项 | 说明 | 默认值 |
|--------|------|---------|
| `ai.enabled_tools` | 启用的工具列表 | `[]`（全部） |
| `ai.allowed_commands` | 允许的命令白名单 | 可选 |

## ⚠️ 生产环境建议

1. **禁用本地命令工具**：在生产环境，建议不启用 `run_command` 工具
2. **严格白名单**：如果必须启用，使用严格的白名单
3. **审计日志**：记录所有命令执行，用于安全审计
4. **隔离环境**：考虑在隔离的环境中执行命令

## 🔐 最佳实践

1. **验证命令**：在执行前验证命令格式和内容
2. **限制工作目录**：防止访问敏感目录
3. **超时控制**：为命令执行设置超时
4. **权限控制**：结合用户权限控制工具可用性
5. **监控资源**：监控 CPU、内存使用
