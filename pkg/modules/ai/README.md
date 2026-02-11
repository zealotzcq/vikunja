# AI Agent Configuration

## Configuration File

Configure the AI agent by editing `config.yml` in the project root directory.

## Configuration Options

### LLM Provider Selection

```yaml
ai:
  llm_provider: mock  # Options: mock, openai, ollama
```

### OpenAI Configuration

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-your-api-key-here
    model: gpt-4o-mini
    base_url: https://api.openai.com/v1  # Optional, default: https://api.openai.com/v1
```

### Ollama Configuration

```yaml
ai:
  llm_provider: ollama
  ollama:
    base_url: http://localhost:11434  # Optional, default: http://localhost:11434
    model: llama2  # Optional, default: llama2
```

### Agent Behavior

```yaml
ai:
  max_iterations: 10  # Max agent iterations (default: 10)
  temperature: 0.7  # LLM temperature 0.0-2.0 (default: 0.7)
  enabled_skills: []  # List of enabled skill names (empty = all enabled)
  enabled_tools: []  # List of enabled tool names (empty = all enabled)
  system_prompt: "You are a helpful assistant..."  # Optional custom prompt
```

## Example Configurations

### Using Mock Provider (for development/testing)

```yaml
ai:
  llm_provider: mock
```

### Using OpenAI

```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-your-api-key-here
    model: gpt-4o-mini
```

### Using Ollama (local LLM)

```yaml
ai:
  llm_provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: llama2
```

### Using OpenAI with Custom Endpoint

```yaml
ai:
  llm_provider: openai
  openai:
    key: your-key
    base_url: https://your-custom-endpoint.com/v1
    model: gpt-4o-mini
```

### Using Azure OpenAI

```yaml
ai:
  llm_provider: openai
  openai:
    key: your-azure-openai-key
    base_url: https://your-resource.openai.azure.com/v1
    model: gpt-4
```

## API Usage

### Using the Agent System

```bash
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Create a task called \"Review documentation\"",
    "page_info": {
      "route_name": "tasks.range",
      "params": {}
    },
    "use_agent": true
  }'
```

## Configuration Reference

| Config Path | Description | Default |
|------------|-------------|---------|
| `ai.llm_provider` | LLM provider (openai, ollama, mock) | `mock` |
| `ai.openai.key` | OpenAI API key | - |
| `ai.openai.model` | OpenAI model | `gpt-4o-mini` |
| `ai.openai.base_url` | OpenAI base URL | `https://api.openai.com/v1` |
| `ai.ollama.base_url` | Ollama base URL | `http://localhost:11434` |
| `ai.ollama.model` | Ollama model | `llama2` |
| `ai.max_iterations` | Max agent iterations | `10` |
| `ai.temperature` | LLM temperature (0.0-2.0) | `0.7` |
| `ai.enabled_skills` | List of enabled skill names | (all enabled) |
| `ai.enabled_tools` | List of enabled tool names | (all enabled) |
| `ai.system_prompt` | Custom system prompt | - |

## Available Tools

- **navigate**: Navigate to different pages in the application
- **create_task**: Create a new task
- **search**: Search for tasks, projects, or teams

## Adding Custom Skills

Custom skills can be registered by implementing the `Skill` interface:

```go
type MyCustomSkill struct{}

func (s *MyCustomSkill) Name() string {
    return "my_custom_skill"
}

func (s *MyCustomSkill) Description() string {
    return "Description of what this skill does"
}

func (s *MyCustomSkill) Execute(ctx *AgentContext, input string) (string, error) {
    // Implementation here
    return "Result", nil
}

func (s *MyCustomSkill) CanHandle(input string) bool {
    // Return true if this skill can handle the input
    return strings.Contains(input, "magic")
}

// Register the skill
skillManager := ai.GetSkillManager()
skillManager.RegisterSkill(&MyCustomSkill{})
```

## Adding Custom Tools

Custom tools can be registered using the tool manager:

```go
toolManager := ai.GetToolManager()
toolManager.RegisterTool(&ai.Tool{
    Name: "my_tool",
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
    Execute: func(ctx *ai.AgentContext, params map[string]interface{}) (string, error) {
        // Implementation here
        return "Tool result", nil
    },
})
```

## Response Format

### Agent Response

```json
{
  "id": "msg_1234567890",
  "role": "assistant",
  "content": "I've created a task called 'Review documentation' for you.",
  "timestamp": 1707652800,
  "navigationCommand": {
    "routeName": "tasks.range",
    "params": {},
    "label": "View tasks"
  },
  "executionSteps": [
    {
      "stepNumber": 1,
      "thought": "User wants to create a task...",
      "action": "create_task",
      "input": "{\"title\": \"Review documentation\"}",
      "output": "Created task: Review documentation"
    }
  ],
  "tokensUsed": 150
}
```

## Development Notes

- The agent system uses a tool-calling approach, where the LLM can invoke tools to perform actions
- Each execution step is tracked for transparency and debugging
- The mock provider is useful for development and testing without requiring an LLM API key
- Skills are for high-level capabilities that can handle user input directly
- Tools are for specific functions that the LLM can invoke
- Chat history is loaded from `chat_session` to maintain conversation context

## Troubleshooting

### Agent not responding

Check agent configuration:
```go
agent, err := ai.GetAgent()
stats := agent.GetStats()
fmt.Printf("Agent stats: %+v\n", stats)
```

Expected output:
```
Agent stats: map[initialized:true llm_provider:openai max_iterations:10 num_skills:0 num_tools:3 temperature:0.7]
```

### Tool execution failing

Ensure the tool is properly registered and enabled:
```go
tm := ai.GetToolManager()
tools := tm.GetEnabledTools()
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}
```

### Configuration not loading

Check backend logs for configuration loading errors. The logs will show:
- Which config file was loaded
- What provider is being used
- Any configuration errors

### Debug Logs

The system includes debug logging to help troubleshoot issues:

```
[Chat] Using Agent system - UserID: 123, Message: Hello
[AI] GenerateAgentResponse - UserID: 123, Provider: openai, Model: gpt-4o-mini
[AI] Creating LLM provider: openai
[AI] Created OpenAI provider - Model: gpt-4o-mini, BaseURL:
[AI] Session loaded - Messages count: 0
[AI] AgentContext created - History messages: 0
[Chat] Agent response: I'm your AI assistant...
```

If you see:
- `[Chat] Using Mock system`: Your request doesn't have `use_agent: true`
- `[AI] Created Mock provider`: Configuration is not loading correctly
- `[AI] Unknown provider`: Config value is incorrect

### LLM API errors

- Verify your API key is correct in `config.yml`
- Check that the base URL is accessible
- Ensure you have sufficient API quota
- Review the error message for specific issues

## Quick Start

1. **Edit `config.yml`** to configure your LLM provider
2. **Restart the backend** to load the new configuration
3. **Send a request** with `use_agent: true`

Example `config.yml`:
```yaml
ai:
  llm_provider: openai
  openai:
    key: sk-proj-abc123...
    model: gpt-4o-mini
```

Restart the service and test with:
```bash
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello",
    "page_info": {"route_name": "home"},
    "use_agent": true
  }'
```

Check the logs to verify the configuration is loaded correctly.
