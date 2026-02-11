# 调试功能已添加

## 已添加的调试日志

为了帮助你排查 OpenAI 配置问题，我已经在以下位置添加了调试日志：

### 1. `pkg/modules/ai/ai.go`

在 `GenerateAgentResponse` 函数中添加了：
```go
log.Printf("[AI] GenerateAgentResponse - UserID: %d, Provider: %s, Model: %s",
    userID, config.LLMProvider, config.OpenAIModel)
log.Printf("[AI] Session loaded - Messages count: %d", len(session.Messages))
log.Printf("[AI] AgentContext created - History messages: %d", len(agentCtx.MessageHistory))
```

### 2. `pkg/modules/ai/agent.go`

在 `createLLMProvider` 函数中添加了：
```go
log.Printf("[AI] Creating LLM provider: %s", a.config.LLMProvider)

switch a.config.LLMProvider {
case "openai":
    log.Printf("[AI] Created OpenAI provider - Model: %s, BaseURL: %s",
        a.config.OpenAIModel, a.config.OpenAIBaseURL)
case "ollama":
    log.Printf("[AI] Created Ollama provider - Model: %s, BaseURL: %s",
        a.config.OllamaModel, a.config.OllamaBaseURL)
case "mock":
    log.Printf("[AI] Created Mock provider")
default:
    log.Printf("[AI] Unknown provider '%s', falling back to Mock", a.config.LLMProvider)
}
```

### 3. `pkg/routes/api/v1/chat.go`

在 `SendMessage` 函数中添加了：
```go
if req.UseAgent {
    log.Printf("[Chat] Using Agent system - UserID: %d, Message: %s, Route: %s",
        userID, req.Message, routeName)

    // ... 处理 agent 响应

    log.Printf("[Chat] Agent response: %s", aiResponse)
} else {
    log.Printf("[Chat] Using Mock system - UserID: %d, Message: %s",
        userID, req.Message)

    // ... 处理 mock 响应

    log.Printf("[Chat] Mock response: %s", aiResponse)
}
```

## 如何使用日志调试

### 1. 启动后端并查看日志

```bash
# 启动后端
go run main.go
```

你应该在控制台看到类似的日志输出。

### 2. 发送请求并观察日志

```bash
# 使用 Agent 系统（use_agent: true）
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello",
    "page_info": {"route_name": "home"},
    "use_agent": true
  }'
```

### 3. 检查日志输出

#### ✅ 正确的 OpenAI 配置日志：

```
[Chat] Using Agent system - UserID: 123, Message: Hello, Route: home
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: I'm your AI assistant powered by OpenAI...
```

#### ❌ 使用 Mock 的日志：

```
[Chat] Using Agent system - UserID: 123, Message: Hello, Route: home
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: 我是您的 AI 助手...
```

如果 Agent 响应是中文的默认帮助信息，说明实际使用的是 Mock provider。

#### ❌ 请求没有使用 Agent 的日志：

```
[Chat] Using Mock system - UserID: 123, Message: Hello
[Chat] Mock response: 我是您的 AI 助手...
```

说明前端请求没有包含 `use_agent: true`。

## 排查步骤

### 步骤 1: 检查请求日志

看日志开头是 `[Chat] Using Agent system` 还是 `[Chat] Using Mock system`

- **Mock system**: 请求中缺少 `use_agent: true`
- **Agent system**: 请求正确，继续下一步

### 步骤 2: 检查 Provider 日志

看 `[AI] Creating LLM provider: ?` 显示的是什么

- **openai**: 配置正确
- **mock**: 环境变量未设置或加载失败
- **unknown**: 配置值拼写错误

### 步骤 3: 检查响应内容

如果日志显示使用了 Agent 但响应还是 mock 风格的中文：

- 说明 LLM 调用失败
- 检查 OpenAI API Key 是否有效
- 检查网络连接
- 检查是否有错误日志

## 常见问题和解决方案

### 问题 1: 请求中使用 Mock 系统

**原因**: 前端请求没有 `use_agent: true`

**解决**: 在前端请求中添加该字段

```javascript
{
  "message": "Hello",
  "page_info": {"route_name": "home"},
  "use_agent": true  // ⬅️ 必须添加
}
```

### 问题 2: Provider 显示为 "mock"

**原因**: 环境变量未设置

**解决**:
```bash
export AI_LLM_PROVIDER=openai
export AI_OPENAI_KEY=sk-your-key
```

然后**重启后端服务**。

### 问题 3: Provider 显示为 "unknown"

**原因**: 环境变量值拼写错误

**解决**: 检查拼写，应该是小写 `openai`，不是 `OpenAI` 或其他变体。

```bash
# 错误
export AI_LLM_PROVIDER=OpenAI

# 正确
export AI_LLM_PROVIDER=openai
```

### 问题 4: 显示 OpenAI provider 但响应是 mock

**原因**: OpenAI API 调用失败，但没有明确的错误日志

**解决**: 在 `provider_openai.go` 中添加更详细的错误日志

```go
if resp.StatusCode != http.StatusOK {
    log.Printf("[OpenAI] API request failed - Status: %d, Body: %s",
        resp.StatusCode, string(body))
    return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
}
```

### 问题 5: 环境变量设置不生效

**原因**: 使用了配置文件，覆盖了环境变量

**解决**: 检查 `config.yml` 文件中的配置

```yaml
# config.yml
services:
  ai:
    llm_provider: "openai"  # 这里会覆盖环境变量
```

删除配置文件中的相关设置或使用环境变量覆盖。

## 测试 OpenAI 连接

```bash
# 直接测试 OpenAI API 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $AI_OPENAI_KEY"

# 如果返回模型列表，说明 API Key 有效
# 如果返回 401，说明 API Key 无效
```

## 完整的调试流程

1. ✅ 检查后端日志，确认是否使用 Agent
2. ✅ 检查 Provider 类型，确认配置加载
3. ✅ 检查响应内容，确认 LLM 调用
4. ✅ 如果有问题，查看详细错误日志
5. ✅ 测试 API Key 有效性
6. ✅ 检查环境变量设置
7. ✅ 重启后端服务

## 预期的正确日志流

```
[Chat] Using Agent system - UserID: 123, Message: Create a task called "Test", Route: tasks.range
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: I'll help you create a task called "Test"...
```

如果看到这样的日志，说明配置和使用都正确！
