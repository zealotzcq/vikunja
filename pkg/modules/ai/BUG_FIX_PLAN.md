# Bug 修复计划

## 🐛 Bug 1: 导航失败导致工具执行失败

### 问题描述
- 用户请求：导航到收藏页
- Agent 响应为空字符串
- 导致前端显示错误并卡住输入框

### 根本原因

1. **OpenAI Provider 格式不正确**：`provider_openai.go` 文件被压缩或格式错误
2. **工具调用解析失败**：`parseToolCall` 函数期望格式 `TOOL: xxx\nINPUT: xxx`
3. **LLM 未返回工具调用**：OpenAI 可能没有理解需要使用工具
4. **工具执行错误没有适当处理**：没有备用响应返回给前端

## 🔧 修复方案

### 步骤 1：修复 OpenAI Provider

1. **重写 `provider_openai.go`**：确保正确的文件格式
2. **添加 JSON Schema 支持**：确保 `openAITool.Parameters` 正确解析
3. **添加工具调用支持**：正确处理 OpenAI Function Calling 格式

### 步骤 2：修复工具执行错误处理

1. **在工具执行失败时返回有意义的消息**
2. **确保 Agent 响应不为空**
3. **记录错误日志**

### 步骤 3：修复界面卡住问题

1. **确保总是返回响应内容**
2. **错误时提供友好的错误消息**
3. **不阻塞用户继续输入**

## 📋 修复内容

### 1. provider_openai.go

**当前问题**：文件格式错误，所有代码在一行上

**修复后**：
```go
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAIProvider struct {
    config *Config
}

func NewOpenAIProvider(config *Config) *OpenAIProvider {
    return &OpenAIProvider{config: config}
}

type openAIRequest struct {
    Model       string          `json:"model"`
    Messages    []openAIMessage `json:"messages"`
    Temperature float64         `json:"temperature"`
    MaxTokens   int             `json:"max_tokens,omitempty"`
    Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type openAITool struct {
    Type     string                 `json:"type"`
    Function openAIFunction `json:"function"`
}

type openAIFunction struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}

type openAIResponse struct {
    Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
    Index   int          `json:"index"`
    Message openAIMessage `json:"message"`
    FinishReason string `json:"finish_reason,omitempty"`
}

func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
    baseURL := p.config.OpenAIBaseURL
    if baseURL == "" {
        baseURL = "https://api.openai.com/v1"
    }

    url := fmt.Sprintf("%s/chat/completions", baseURL)

    messages := []openAIMessage{
        {Role: "system", Content: p.config.SystemPrompt},
        {Role: "user", Content: prompt},
    }

    openAITools := make([]openAITool, 0, len(tools))
    for _, toolDef := range tools {
        name, _ := toolDef["name"].(string)
        desc, _ := toolDef["description"].(string)
        params, _ := toolDef["parameters"].(map[string]interface{})

        openAITools = append(openAITools, openAITool{
            Type:     "function",
            Function: openAIFunction{
                Name:        name,
                Description: desc,
                Parameters:  params,
            },
        })
    }

    reqBody := openAIRequest{
        Model:       p.config.OpenAIModel,
        Messages:    messages,
        Temperature: p.config.Temperature,
        MaxTokens:   1000,
        Tools:       openAITools,
    }

    // ... 发送请求并解析响应
}
```

### 2. Agent 工具失败处理

**当前问题**：工具执行失败时返回空响应

**修复后**：
```go
func (a *Agent) runAgentLoop(ctx context.Context, agentCtx *AgentContext, userMessage string) (*AgentResponse, error) {
    maxIterations := a.config.MaxIterations

    for i := 0; i < maxIterations; i++ {
        prompt := a.buildPrompt(agentCtx, userMessage, i)
        tools := a.toolManager.GetEnabledTools()
        toolDefinitions := a.toolManager.GetToolDefinitions()

        var llmResponse string
        var err error

        if len(tools) > 0 {
            llmResponse, err = a.llmProvider.GenerateWithTools(ctx, prompt, toolDefinitions)
        } else {
            llmResponse, err = a.llmProvider.Generate(ctx, prompt)
        }

        if err != nil {
            log.Printf("[AI] LLM generation failed: %v", err)

            // 返回错误消息而不是空
            return &AgentResponse{
                Content:     fmt.Sprintf("抱歉，我遇到了一些问题。%v", err.Error()),
                ShouldNavigate: false,
                ExecutionSteps: agentCtx.ExecutionSteps,
            }, nil
        }

        // 解析工具调用
        toolCall, err := a.parseToolCall(llmResponse)

        // 如果没有工具调用，返回直接响应
        if toolCall == nil {
            return &AgentResponse{
                Content:     llmResponse,
                NavigationInfo: agentCtx.NavigationInfo,
                ShouldNavigate: agentCtx.ShouldNavigate,
                ExecutionSteps: agentCtx.ExecutionSteps,
                TokensUsed:     agentCtx.TokensUsed,
            }, nil
        }

        // 执行工具
        result, err := a.toolManager.ExecuteTool(toolCall.Name, agentCtx, toolCall.Input)

        if err != nil {
            log.Printf("[AI] Tool execution failed: %v", err)

            // 工具执行失败时返回错误消息
            return &AgentResponse{
                Content:     fmt.Sprintf("工具执行失败：%v。请稍后重试。", err.Error()),
                ShouldNavigate: false,
                ExecutionSteps: append(agentCtx.ExecutionSteps, ExecutionStep{
                    StepNumber: i + 1,
                    Thought:    llmResponse,
                    Action:     toolCall.Name,
                    Input:      toolCall.Input,
                    Output:     err.Error(),
                }),
            }, nil
        }

        // 记录成功的工具执行
        agentCtx.ExecutionSteps = append(agentCtx.ExecutionSteps, ExecutionStep{
            StepNumber: i + 1,
            Thought:    llmResponse,
            Action:     toolCall.Name,
            Input:      toolCall.Input,
            Output:     result,
        })

        // 继续下一轮
        // ...
    }
}
```

### 3. 修复 parseToolCall 以支持 OpenAI 格式

```go
func (a *Agent) parseToolCall(response string) (*ToolCall, error) {
    // OpenAI Function Calling 返回格式是 JSON，不是文本
    // 需要解析 JSON 响应中的 tool_calls 字段

    // 更新 buildPrompt 以支持 OpenAI 格式
}
```

## ✅ 修复后的预期行为

### 场景 1：成功导航

**用户请求**： "我要导航到收藏页"

**预期日志**：
```
[Chat] Using Agent system - UserID: 1, Message: 我要导航到收藏页
[AI] GenerateAgentResponse - UserID: 1, Provider: openai
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[AI] Building prompt with navigation tools
[AI] LLM response (tool call detected)
[AI] Tool call: navigate -> {"route_name": "project.index", "params": {"projectId": -1}}
[AI] Tool executed successfully
[Chat] Agent response: 正在导航到收藏页...
```

**预期响应**：
```json
{
  "id": "msg_xxx",
  "role": "assistant",
  "content": "正在导航到收藏页...",
  "timestamp": 1234567890,
  "navigationCommand": {
    "routeName": "project.index",
    "params": {"projectId": -1},
    "label": "正在导航到收藏页"
  }
}
```

### 场景 2：工具失败

**用户请求**：工具调用失败的场景

**预期日志**：
```
[AI] Tool execution failed: invalid parameters
[AI] Returning error message
```

**预期响应**：
```json
{
  "id": "msg_xxx",
  "role": "assistant",
  "content": "工具执行失败：参数格式错误。请稍后重试。",
  "timestamp": 1234567890,
  "navigationCommand": null
}
```

## 🔍 实施计划

### 阶段 1：重写 provider_openai.go

1. 检查当前文件格式问题
2. 重新创建正确格式的文件
3. 添加 OpenAI Function Calling 支持
4. 测试基本的工具调用

### 阶段 2：修复 Agent 错误处理

1. 在工具执行失败时返回友好的错误消息
2. 确保总是返回有效的响应内容
3. 添加详细的错误日志

### 阶段 3：测试验证

1. 测试导航功能
2. 测试错误处理
3. 验证前端不再卡住

## 📋 注意事项

1. **文件格式**：确保 `.go` 文件使用正确的格式（LF 换行）
2. **错误处理**：所有错误都应该有有意义的用户友好消息
3. **日志完整性**：添加足够的日志用于调试
4. **响应不为空**：确保所有情况下都返回内容
