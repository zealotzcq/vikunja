# AI Agent 与 Chat Session 交互说明

## 当前架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                     用户请求流程                                │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  1. API Layer (chat.go)                                       │
│     - 接收用户消息和页面上下文                                    │
│     - 保存用户消息到 chat_session                                  │
└──────────────────────────────┬────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│  2. AI Agent (ai.go + agent.go)                                │
│     - 使用 agent.ProcessMessage() 处理消息                           │
│     - Agent 维护自己的 AgentContext.MessageHistory                     │
│     - MessageHistory 用于构建 LLM prompt                              │
└──────────────────────────────┬────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│  3. LLM Provider (provider_*.go)                               │
│     - 使用 MessageHistory 构建对话上下文                               │
│     - 调用 LLM API 生成响应                                     │
│     - 返回生成的文本                                             │
└──────────────────────────────┬────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│  4. API Layer (chat.go)                                       │
│     - 获取 AI 响应                                             │
│     - 保存助手消息到 chat_session                                  │
│     - 返回响应给前端                                              │
└─────────────────────────────────────────────────────────────────────┘
```

## 数据流向

### 当前实现（两条独立的消息记录）

```
Chat Session (chat_session)           AI Agent (ai/agent.go)
├── messages[]                     ├── MessageHistory[]
│   ├── msg_001 (user)               │   ├── Message{role:"user", content:"..."}
│   ├── msg_002 (assistant)          │   └── Message{role:"assistant", content:"..."}
│   ├── msg_003 (user)               └── 每次调用都会重新构建，不从 chat_session 读取
│   ├── msg_004 (assistant)
│   └── ...
│
│ 存储位置: keyvalue 存储            存储位置: 内存 (每次请求中)
│ 生命周期: 1小时 TTL                生命周期: 单次请求
│ 用途: 前端显示历史记录              用途: LLM prompt 构建
```

## 当前代码流程分析

### 1. 接收消息 (chat.go:88-98)

```go
// 保存用户消息到 chat_session
userMessage := chat_session.Message{
    ID:        userMsgID,
    Role:      "user",
    Content:   req.Message,
    Timestamp: time.Now().Unix(),
}

chat_session.GetDefault().AddMessage(userID, userMessage)
```

### 2. 调用 Agent (chat.go:112-126)

```go
// 使用 agent 处理（注意：Agent 内部维护自己的 MessageHistory）
agentResponse, err := ai.GenerateAgentResponse(
    c.Request().Context(),
    userID,           // 只传递 userID，但不从 chat_session 读取历史
    req.Message,      // 只传递当前消息
    routeName,
    routeParams,
)
```

### 3. Agent 内部处理 (agent.go:ProcessMessage)

```go
func (a *Agent) ProcessMessage(ctx context.Context, agentCtx *AgentContext, message string) (*AgentResponse, error) {
    // 将当前消息添加到 AgentContext.MessageHistory
    agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
        Role:    "user",
        Content: message,
    })

    // 处理消息
    response, err := a.runAgentLoop(ctx, agentCtx, message)

    // 将响应添加到 MessageHistory
    agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
        Role:    "assistant",
        Content: response.Content,
    })

    return response, nil
}
```

### 4. 保存助手消息 (chat.go:147-157)

```go
// 保存助手消息到 chat_session
assistantMessage := chat_session.Message{
    ID:        assistantMsgID,
    Role:      "assistant",
    Content:   aiResponse,
    Timestamp: time.Now().Unix(),
}

chat_session.GetDefault().AddMessage(userID, assistantMessage)
```

## 当前设计的优缺点

### ✅ 优点

1. **职责分离**：
   - `chat_session`：持久化存储，用于前端显示历史
   - `ai.AgentContext.MessageHistory`：临时上下文，用于 LLM prompt 构建

2. **灵活性强**：
   - Agent 可以独立管理自己的对话上下文
   - chat_session 只负责持久化和展示

3. **简单清晰**：
   - 两个系统互不干扰
   - 易于理解和维护

### ❌ 缺点

1. **历史记录不共享**：
   - Agent 的 MessageHistory 是临时的，不包含历史消息
   - 每次新请求无法利用 chat_session 中的历史记录
   - LLM 无法记住之前的对话

2. **重复存储**：
   - 同样的消息存储在两个地方
   - chat_session 和 AgentContext.MessageHistory 各自维护

3. **上下文断裂**：
   - 用户切换页面或重新加载后，Agent 上下文丢失
   - 无法保持多轮对话的连贯性

## 改进方案

### 方案1：从 chat_session 加载历史（推荐）

**优点**：
- ✅ 保持对话连贯性
- ✅ 利用已有的历史记录
- ✅ LLM 可以理解之前的对话

**实现**：

```go
// ai.GenerateAgentResponse 修改
func GenerateAgentResponse(
    ctx context.Context,
    userID int64,
    userContent string,
    routeName string,
    routeParams map[string]interface{},
) (*AgentResponse, error) {
    agent, err := GetAgent()
    if err != nil {
        return nil, fmt.Errorf("failed to get agent: %w", err)
    }

    // 1. 从 chat_session 加载历史记录
    session, err := chat_session.GetDefault().GetOrCreateSession(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    // 2. 构建 AgentContext，包含历史消息
    agentCtx := &AgentContext{
        UserID:       userID,
        CurrentRoute: routeName,
        RouteParams:  routeParams,
        SessionData:  make(map[string]interface{}),
        MessageHistory: make([]Message, 0, len(session.Messages)),
    }

    // 3. 将 chat_session 消息转换为 Agent Message 格式
    for _, msg := range session.Messages {
        agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
            Role:    msg.Role,
            Content: msg.Content,
        })
    }

    // 4. 处理当前消息
    return agent.ProcessMessage(ctx, agentCtx, userContent)
}
```

### 方案2：Agent 管理长期上下文

**优点**：
- ✅ Agent 独立管理上下文
- ✅ 更灵活的上下文管理策略

**实现**：

```go
// Agent 添加持久化存储
type Agent struct {
    config       *Config
    llmProvider  LLMProvider
    skillManager *SkillManager
    toolManager  *ToolManager
    mu           sync.RWMutex
    initialized   bool

    // 添加用户上下文存储
    userContexts map[int64]*AgentContext
    contextsMu   sync.RWMutex
}

// 获取或创建用户上下文
func (a *Agent) getUserContext(userID int64) *AgentContext {
    a.contextsMu.RLock()
    ctx, exists := a.userContexts[userID]
    a.contextsMu.RUnlock()

    if exists {
        return ctx
    }

    a.contextsMu.Lock()
    defer a.contextsMu.Unlock()

    if ctx, exists := a.userContexts[userID]; exists {
        return ctx
    }

    ctx = &AgentContext{
        UserID:       userID,
        MessageHistory: []Message{},
        SessionData:  make(map[string]interface{}),
    }
    a.userContexts[userID] = ctx
    return ctx
}
```

### 方案3：混合方案（最佳实践）

结合方案1和2的优点：

1. **持久化存储**：使用 `chat_session` 作为真实历史来源
2. **缓存优化**：Agent 内部缓存最近的消息
3. **智能截断**：根据 token 限制智能截断历史

**架构**：

```
┌─────────────────────────────────────────────────────┐
│         Chat Session (持久化)                   │
│         - 完整历史记录                         │
│         - 1小时 TTL                             │
└───────────────┬─────────────────────────────────┘
                │ 读取
                ▼
┌─────────────────────────────────────────────────────┐
│         AI Agent (智能缓存)                      │
│         - 最近 N 条消息                         │
│         - 根据模型限制自动调整                    │
│         - 构建 LLM prompt                       │
└─────────────────────────────────────────────────────┘
```

**实现关键点**：

```go
const (
    maxHistoryTokens = 4000  // 预留 1000 tokens 给响应
    avgTokensPerMessage = 50 // 估算值
)

func (a *Agent) buildPrompt(agentCtx *AgentContext, userMessage string, iteration int) string {
    // 1. 从 chat_session 加载完整历史
    session, _ := chat_session.GetDefault().GetOrCreateSession(agentCtx.UserID)

    // 2. 智能截断：保留最近的 N 条消息
    maxMessages := maxHistoryTokens / avgTokensPerMessage
    recentMessages := session.Messages
    if len(recentMessages) > maxMessages {
        recentMessages = recentMessages[len(recentMessages)-maxMessages:]
    }

    // 3. 构建 prompt
    prompt := a.config.SystemPrompt + "\n\nConversation history:\n"
    for _, msg := range recentMessages {
        prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
    }

    return prompt
}
```

## 推荐实施方案

### 短期：使用方案1（最小改动）

```go
// 修改 pkg/modules/ai/ai.go
// 在 GenerateAgentResponse 中从 chat_session 加载历史
```

**优点**：
- ✅ 改动最小
- ✅ 立即获得对话记忆能力
- ✅ 利用现有基础设施

### 长期：使用方案3（最佳实践）

```go
// 实现智能历史管理和 token 优化
// 添加上下文窗口管理
// 支持不同模型的 token 限制
```

**优点**：
- ✅ 最佳用户体验
- ✅ 性能优化
- ✅ 成本控制

## 配置选项建议

```go
// Config 添加
type Config struct {
    // ... 现有字段

    // 新增：历史记录配置
    HistoryEnabled      bool   `mapstructure:"history_enabled"`       // 是否启用历史记录
    MaxHistoryMessages  int    `mapstructure:"max_history_messages"`  // 最大历史消息数
    MaxHistoryTokens   int    `mapstructure:"max_history_tokens"`    // 最大历史 tokens
    HistoryTTL        int    `mapstructure:"history_ttl"`         // 历史记录 TTL (分钟)
}
```

## 总结

当前实现中，**chat_session 和 AI agent 是独立的两个系统**：

1. **chat_session**：负责持久化和前端展示
2. **AI agent.MessageHistory**：负责 LLM prompt 构建（临时）

这种设计简单清晰，但存在**历史记录不共享**的问题。

**推荐改进**：在 `GenerateAgentResponse` 中从 `chat_session` 加载历史记录，让 LLM 能够理解之前的对话，提供更好的用户体验。
