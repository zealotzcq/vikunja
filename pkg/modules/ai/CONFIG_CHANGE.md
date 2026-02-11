# AI Agent 配置方式变更 - 完成

## ✅ 已完成

已将 AI Agent 配置方式从**环境变量**改为**config.yml文件**。

## 📝 修改的文件

### 1. `config.yml` - 添加AI配置

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

# AI 助手配置 (新增)
ai:
  llm_provider: mock  # Options: mock, openai, ollama
  openai:
    key: ""  # OpenAI API Key
    model: gpt-4o-mini
    base_url: ""  # Optional, default: https://api.openai.com/v1
  ollama:
    base_url: http://localhost:11434
    model: llama2
  max_iterations: 10
  temperature: 0.7
  enabled_skills: []
  enabled_tools: []
  system_prompt: ""
```

### 2. `pkg/modules/ai/config.go` - 从viper读取配置

**变更**:
- ❌ 删除环境变量读取逻辑
- ✅ 使用viper从config.yml读取
- ✅ 修改字段映射：`openai_key` → `openai.key`

**新的配置结构**:
```go
type Config struct {
    LLMProvider    string  `mapstructure:"llm_provider"`
    OpenAIKey      string  `mapstructure:"key"`         // ← 改为 key
    OpenAIModel    string  `mapstructure:"model"`        // ← 改为 model
    OpenAIBaseURL  string  `mapstructure:"base_url"`    // ← 改为 base_url
    OllamaBaseURL string  `mapstructure:"base_url"`
    OllamaModel    string  `mapstructure:"model"`
    MaxIterations   int     `mapstructure:"max_iterations"`
    Temperature    float64 `mapstructure:"temperature"`
    EnabledSkills  []string `mapstructure:"enabled_skills"`
    EnabledTools   []string `mapstructure:"enabled_tools"`
    SystemPrompt   string  `mapstructure:"system_prompt"`
}
```

**新的LoadConfig函数**:
```go
func LoadConfig() (*Config, error) {
    configOnce.Do(func() {
        config = &Config{
            LLMProvider: viper.GetString("ai.llm_provider"),
        }

        // 从 config.yml 读取配置
        if viper.IsSet("ai.openai.key") {
            config.OpenAIKey = viper.GetString("ai.openai.key")
        }
        if viper.IsSet("ai.openai.model") {
            config.OpenAIModel = viper.GetString("ai.openai.model")
        }
        // ... 其他配置

        // 设置默认值
        if config.LLMProvider == "" {
            config.LLMProvider = "mock"
        }
        // ... 其他默认值
    })

    return config, nil
}
```

### 3. `pkg/modules/ai/README.md` - 更新文档

**变更**:
- ❌ 删除所有环境变量相关内容
- ✅ 改为config.yml配置说明
- ✅ 添加完整配置示例
- ✅ 添加配置参考表

### 4. `pkg/modules/ai/TROUBLESHOOTING.md` - 更新调试指南

**变更**:
- ❌ 删除环境变量检查步骤
- ✅ 改为config.yml文件检查
- ✅ 添加YAML格式验证
- ✅ 添加重启服务步骤

### 5. 调试日志保持不变

调试日志已经添加到以下文件中：
- `pkg/modules/ai/ai.go` - `GenerateAgentResponse` 函数
- `pkg/modules/ai/agent.go` - `createLLMProvider` 函数
- `pkg/routes/api/v1/chat.go` - `SendMessage` 函数

日志格式：
```
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: I'm your AI assistant...
```

## 🔧 配置示例

### 使用 Mock (开发/测试）

```yaml
ai:
  llm_provider: mock
```

### 使用 OpenAI

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-proj-abc123xyz789...
    model: gpt-4o-mini
```

### 使用 Ollama (本地LLM)

```yaml
ai:
  llm_provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: llama2
```

### 使用 Azure OpenAI

```yaml
ai:
  llm_provider: openai
  openai:
    key: your-azure-key
    base_url: https://your-resource.openai.azure.com/v1
    model: gpt-4
```

## 📋 使用步骤

1. **编辑 config.yml** - 添加AI配置
2. **重启后端服务** - 让配置生效
3. **查看日志** - 确认配置正确加载
4. **测试请求** - 验证功能正常

## 🔍 调试步骤

### 1. 检查config.yml配置

```bash
cat config.yml
```

确认 `ai.llm_provider` 设置为 `openai` 而不是 `mock`。

### 2. 查看启动日志

后端启动时应该看到：
```
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
```

如果看到 `mock`，说明配置没有正确加载。

### 3. 查看请求日志

发送请求时应该看到：
```
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
```

如果 Provider 显示为 `mock`，说明配置加载有问题。

### 4. 检查响应内容

- ✅ OpenAI 响应：英文，多样化，智能
- ❌ Mock 响应：中文，固定模式，简单

## ⚠️ 常见问题

### 问题：配置是openai但响应是mock

**检查清单**:
1. ✅ config.yml 中 `ai.llm_provider` 是 `openai`
2. ✅ config.yml 中 `ai.openai.key` 已设置
3. ✅ YAML 格式正确（使用空格缩进，不是Tab）
4. ✅ 后端服务已重启
5. ✅ 请求包含 `use_agent: true`

**解决方法**:
1. 确认 config.yml 文件位置（项目根目录）
2. 验证 YAML 语法（无Tab，正确的缩进）
3. 重启后端服务
4. 查看日志中的错误信息

### 问题：启动时没有看到 AI 日志

**可能原因**:
- config.yml 文件不存在或位置错误
- YAML 格式错误导致无法解析

**解决方法**:
1. 确认 config.yml 在项目根目录
2. 使用 YAML 验证工具检查语法
3. 查看启动日志中的配置加载错误

### 问题：配置加载了但API调用失败

**可能原因**:
- API Key 无效或过期
- 网络无法访问 OpenAI
- API 配额用完

**解决方法**:
1. 测试 API Key 有效性
2. 检查网络连接
3. 查看详细的错误日志

## 📚 相关文档

- `README.md` - 用户配置指南
- `TROUBLESHOOTING.md` - 调试和故障排除
- `INTERACTION.md` - Chat Session 交互说明
- `DEBUG_GUIDE.md` - 调试日志说明

## ✅ 验证步骤

运行以下命令验证修改：

```bash
# 1. 检查config.yml
cat config.yml

# 2. 编译相关包
go build ./pkg/modules/ai/... ./pkg/routes/api/v1/...

# 3. 运行测试
cd pkg/modules/ai && go test -v
```

预期结果：
- ✅ 所有测试通过
- ✅ 无编译错误
- ✅ config.yml 格式正确

## 🎯 总结

**主要变更**:
1. ✅ 配置方式：环境变量 → config.yml
2. ✅ 配置文件：新增 AI 配置项
3. ✅ 配置加载：使用 viper 读取
4. ✅ 文档更新：所有相关文档已更新
5. ✅ 向后兼容：保留 Mock provider 选项

**使用方式**:
1. 编辑项目根目录的 `config.yml`
2. 添加 `ai` 配置部分
3. 重启后端服务
4. 发送带有 `use_agent: true` 的请求

**优势**:
- ✅ 配置集中管理
- ✅ 更容易调试和维护
- ✅ 无需设置环境变量
- ✅ 支持版本控制
- ✅ 与其他配置统一管理
