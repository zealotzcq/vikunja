# AI Agent 配置调试指南

## 问题诊断

如果你设置了 OpenAI 但聊天结果还是 mock 实现，可能的原因如下：

## 1. 检查请求是否使用 Agent

**问题**: 前端没有设置 `use_agent: true`

**解决方案**: 确保 API 请求包含 `use_agent: true`

```bash
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Create a task",
    "page_info": {"route_name": "tasks.range"},
    "use_agent": true  ⬅️ 必须设置为 true
  }'
```

**前端请求示例**:
```javascript
fetch('/api/v1/chat/send', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    message: 'Create a task',
    page_info: { route_name: 'tasks.range' },
    use_agent: true  ⬅️ 重要：这里必须为 true
  })
})
```

## 2. 检查 config.yml 配置

**问题**: config.yml 文件配置不正确

**检查方法**:

打开项目根目录的 `config.yml` 文件，确认 AI 配置部分：

```yaml
ai:
  llm_provider: openai  # 确认这里设置为 openai
  openai:
    key: sk-your-api-key-here  # 确认这里设置了 API key
    model: gpt-4o-mini
```

**正确配置**:

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-proj-abc123xyz789  # 你的 OpenAI API Key
    model: gpt-4o-mini
    base_url: https://api.openai.com/v1  # 可选
```

## 3. 重启后端服务

**问题**: 修改了 config.yml 但没有重启服务

**解决方法**: 重启后端服务以加载新配置

```bash
# 停止当前运行的服务
Ctrl+C

# 重新启动
go run main.go
```

## 4. 检查配置加载日志

**问题**: 配置没有正确加载

**调试方法**: 查看后端启动日志和请求日志

**预期的启动日志**:

```
# 如果配置正确，应该看到 OpenAI 相关日志
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
```

**预期的请求日志**:

```
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: I'm your AI assistant...
```

**如果看到 Mock 相关日志**:

```
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: mock, Model: gpt-4o-mini
[AI] Creating LLM provider: mock
[AI] Created Mock provider
```

说明配置还是 `mock`，请检查 config.yml。

## 5. 验证配置文件格式

**问题**: config.yml 语法错误或格式不正确

**检查方法**:

```bash
# 验证 YAML 语法是否正确
# (如果有 yamllint 工具）
yamllint config.yml

# 或者简单地查看文件
cat config.yml
```

**正确的 YAML 格式**:

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-proj-abc123...
    model: gpt-4o-mini
    base_url: ""
  ollama:
    base_url: http://localhost:11434
    model: llama2
  max_iterations: 10
  temperature: 0.7
  enabled_skills: []
  enabled_tools: []
  system_prompt: ""
```

**注意**:
- 缩进必须是空格，不能是 Tab
- `key` 前面不能有引号
- 布尔值和列表要符合 YAML 规范

## 6. 检查 API Key 有效性

**问题**: OpenAI API Key 无效

**测试方法**:

```bash
# 直接测试 OpenAI API 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer sk-proj-abc123..."

# 如果返回模型列表，说明 API Key 有效
# 如果返回 401，说明 API Key 无效
```

## 7. 常见配置错误

### 错误 1: 配置路径错误

**问题**: config.yml 文件不在项目根目录

**检查**:
```bash
ls config.yml
```

应该看到 config.yml 文件。

### 错误 2: 配置缩进错误

**问题**: 使用了 Tab 而不是空格

**解决**: 使用 2 个空格作为缩进

```yaml
# ❌ 错误：使用 Tab
ai:
	llm_provider: openai

# ✅ 正确：使用空格
ai:
  llm_provider: openai
```

### 错误 3: 字段名拼写错误

**问题**: 字段名拼写不正确

**检查**: 确保字段名正确

```yaml
ai:
  llm_provider: openai  # ✅ 正确
  # llmprovider: openai   # ❌ 错误：缺少下划线
```

### 错误 4: 引号使用错误

**问题**: 值使用了多余的引号

**解决**: 字符串值不需要额外的引号

```yaml
ai:
  llm_provider: openai  # ✅ 正确
  # llm_provider: "openai"  # ❌ 错误：不需要引号
  openai:
    key: sk-proj-abc123...  # ✅ 正确
```

## 8. 完整的配置和测试流程

### 步骤 1: 配置 config.yml

```yaml
# config.yml
ai:
  llm_provider: openai
  openai:
    key: sk-proj-abc123xyz789
    model: gpt-4o-mini
```

### 步骤 2: 重启后端

```bash
# 停止当前运行的服务
Ctrl+C

# 重新启动
go run main.go
```

### 步骤 3: 查看启动日志

应该看到：
```
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
```

如果看到 `mock`，说明配置没有正确加载。

### 步骤 4: 发送测试请求

```bash
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, can you help me?",
    "page_info": {"route_name": "home"},
    "use_agent": true
  }'
```

### 步骤 5: 查看请求日志

应该看到：
```
[Chat] Using Agent system - UserID: 123, Message: Hello, can you help me?
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: Hello! I'm your AI assistant...
```

## 9. 调试步骤总结

1. ✅ 确认 `use_agent: true` 在请求中
2. ✅ 检查 `config.yml` 文件配置
3. ✅ 重启后端服务
4. ✅ 查看启动日志，确认配置加载
5. ✅ 查看请求日志，确认 provider 类型
6. ✅ 测试 API Key 有效性
7. ✅ 检查响应内容

## 10. 快速检查命令

```bash
# 检查 config.yml 文件
cat config.yml

# 测试 API 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer <your-key>"

# 发送测试请求
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello","page_info":{"route_name":"home"},"use_agent":true}'
```

## 11. 常见问题解决

### 问题：看到 Mock 响应但配置是 OpenAI

**检查清单**:
- [ ] config.yml 中 `ai.llm_provider` 设置为 `openai`
- [ ] config.yml 中 `ai.openai.key` 已设置
- [ ] 后端服务已重启
- [ ] 请求包含 `use_agent: true`
- [ ] YAML 格式正确（无 Tab 缩进）

**解决方法**:
1. 检查 config.yml 文件位置
2. 验证 YAML 语法
3. 确认字段名拼写正确
4. 重启后端服务
5. 查看详细日志

### 问题：启动时没有看到 AI 相关日志

**可能原因**:
- config.yml 没有被读取
- AI 配置部分不存在或格式错误

**解决方法**:
1. 确认 config.yml 文件在项目根目录
2. 验证 YAML 语法
3. 检查 viper 配置初始化

### 问题：配置加载了但仍然使用 Mock

**可能原因**:
- OpenAI API 调用失败
- 网络无法访问 OpenAI
- API Key 无效

**解决方法**:
查看日志中的详细错误信息，并检查：
- API Key 是否正确
- 网络连接是否正常
- API 服务是否可用

## 12. 预期的正确日志流

```
[启动日志]
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:

[请求日志]
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: Hello! I'm your AI assistant...
```

如果看到以上日志，说明配置和使用都正确！

## 13. 示例完整配置文件

```yaml
# Vikunja 配置文件

service:
  interface: 0.0.0.0:3456
  publicurl: http://localhost:3456
  rootpath: ./

cors:
  enable: false

database:
  type: sqlite
  path: ./vikunja.db

# AI 助手配置
ai:
  llm_provider: openai  # mock, openai, ollama
  openai:
    key: sk-proj-abc123xyz789...  # 你的 OpenAI API Key
    model: gpt-4o-mini
    base_url: https://api.openai.com/v1
  ollama:
    base_url: http://localhost:11434
    model: llama2
  max_iterations: 10
  temperature: 0.7
  enabled_skills: []
  enabled_tools: []
  system_prompt: ""
```

复制以上配置到 `config.yml`，然后重启服务即可。
