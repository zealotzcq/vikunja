# AI Agent System

## Overview

The AI Agent System provides a flexible framework for building intelligent task management assistants in Vikunja. It supports multiple LLM providers, dynamic skill/tool registration, and extensible agent behaviors.

## Architecture

```
┌─────────────────┐
│   chat.go       │  HTTP API endpoints
│   (Routes)      │
└────────┬────────┘
         │
┌────────▼────────┐
│   ai.go         │  Legacy mock implementation
│   (AI Module)   │
└────────┬────────┘
         │
┌────────▼────────┐
│   agent.go      │  Core agent logic
│   (Agent)       │
└───────┬────┬────┘
        │    │
        │    ├──► skills.go    (Skill management)
        │    └──► tools.go     (Tool management)
        │
        ├──► config.go       (Configuration)
        │
        ├──► provider_openai.go    (OpenAI LLM)
        ├──► provider_ollama.go    (Ollama LLM)
        └──► provider_mock.go      (Mock LLM for testing)
```

## Quick Start

### 1. Using the Mock Provider (No API Key Required)

```bash
export AI_LLM_PROVIDER=mock
```

### 2. Using OpenAI

```bash
export AI_LLM_PROVIDER=openai
export AI_OPENAI_KEY=sk-your-api-key-here
export AI_OPENAI_MODEL=gpt-4o-mini
```

### 3. Using Ollama (Local LLM)

```bash
# Install Ollama from https://ollama.ai
# Download a model: ollama pull llama2

export AI_LLM_PROVIDER=ollama
export AI_OLLAMA_BASE_URL=http://localhost:11434
export AI_OLLAMA_MODEL=llama2
```

## API Usage

### Legacy API (Mock Implementation)

```bash
curl -X POST http://localhost:3456/api/v1/chat/send \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Show me project 123",
    "page_info": {
      "route_name": "projects.index",
      "params": {}
    }
  }'
```

### New Agent API

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

## Response Format

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

## Extending the System

### Adding a Custom Tool

```go
package customtools

import (
    "code.vikunja.io/api/pkg/modules/ai"
    "fmt"
)

func RegisterCustomTool() error {
    tm := ai.GetToolManager()

    tool := &ai.Tool{
        Name: "my_custom_tool",
        Description: "Performs a custom operation",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type": "string",
                    "description": "First parameter",
                },
            },
            "required": []string{"param1"},
        },
        Execute: func(ctx *ai.AgentContext, params map[string]interface{}) (string, error) {
            param1, _ := params["param1"].(string)

            // Your custom logic here
            result := fmt.Sprintf("Processed: %s", param1)

            return result, nil
        },
    }

    return tm.RegisterTool(tool)
}
```

### Adding a Custom Skill

```go
package customskills

import (
    "code.vikunja.io/api/pkg/modules/ai"
    "strings"
)

type MySkill struct{}

func (s *MySkill) Name() string {
    return "my_custom_skill"
}

func (s *MySkill) Description() string {
    return "Handles specific user requests"
}

func (s *MySkill) Execute(ctx *ai.AgentContext, input string) (string, error) {
    return "Skill executed successfully", nil
}

func (s *MySkill) CanHandle(input string) bool {
    return strings.Contains(input, "magic")
}

func RegisterCustomSkill() error {
    sm := ai.GetSkillManager()
    return sm.RegisterSkill(&MySkill{})
}
```

### Adding a Custom LLM Provider

```go
package customprovider

import (
    "code.vikunja.io/api/pkg/modules/ai"
    "context"
)

type CustomProvider struct {
    config *ai.Config
}

func (p *CustomProvider) Generate(ctx context.Context, prompt string) (string, error) {
    // Your LLM implementation here
    return "Response", nil
}

func (p *CustomProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
    // Your LLM implementation with tools here
    return "Response with tools", nil
}

func NewCustomProvider(config *ai.Config) ai.LLMProvider {
    return &CustomProvider{config: config}
}
```

## Configuration Reference

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `AI_LLM_PROVIDER` | LLM provider (openai, ollama, mock) | `openai` |
| `AI_OPENAI_KEY` | OpenAI API key | - |
| `AI_OPENAI_MODEL` | OpenAI model | `gpt-4o-mini` |
| `AI_OPENAI_BASE_URL` | OpenAI base URL | `https://api.openai.com/v1` |
| `AI_OLLAMA_BASE_URL` | Ollama base URL | `http://localhost:11434` |
| `AI_OLLAMA_MODEL` | Ollama model | `llama2` |
| `AI_MAX_ITERATIONS` | Max agent iterations | `10` |
| `AI_TEMPERATURE` | LLM temperature (0.0-2.0) | `0.7` |
| `AI_ENABLED_SKILLS` | Comma-separated enabled skills | (all enabled) |
| `AI_ENABLED_TOOLS` | Comma-separated enabled tools | (all enabled) |
| `AI_SYSTEM_PROMPT` | Custom system prompt | - |

## Available Tools

### navigate
Navigate to a specific page in the application.

Parameters:
- `route_name` (required): The route name
- `params` (optional): Route parameters

### create_task
Create a new task.

Parameters:
- `title` (required): Task title
- `description` (optional): Task description
- `project_id` (optional): Project ID

### search
Search for tasks, projects, or teams.

Parameters:
- `query` (required): Search query
- `type` (required): Search type (tasks, projects, teams)

## Testing

Run tests for the AI module:

```bash
cd pkg/modules/ai
go test -v
```

## Troubleshooting

### Agent Not Responding

Check the agent configuration and status:

```go
agent, err := ai.GetAgent()
if err != nil {
    log.Printf("Failed to get agent: %v", err)
    return
}

stats := agent.GetStats()
log.Printf("Agent stats: %+v", stats)
```

### Tool Execution Failing

Verify the tool is registered and enabled:

```go
tm := ai.GetToolManager()
tools := tm.GetEnabledTools()
for _, tool := range tools {
    log.Printf("Tool: %s - %s", tool.Name, tool.Description)
}
```

### LLM API Errors

- Verify your API key is correct
- Check that the base URL is accessible
- Ensure you have sufficient API quota
- Review the error message for specific issues

## Best Practices

1. **Start with Mock Provider**: Use the mock provider for development and testing before integrating with real LLMs.

2. **Enable Selective Tools**: Only enable the tools you need to reduce complexity and improve performance:
   ```bash
   export AI_ENABLED_TOOLS=navigate,create_task
   ```

3. **Customize System Prompt**: Tailor the system prompt to your specific use case for better results.

4. **Monitor Execution Steps**: Review execution steps to understand how the agent is processing requests.

5. **Handle Errors Gracefully**: Always check for errors when executing tools or processing agent responses.

## Future Enhancements

Potential areas for improvement:
- Add more pre-built tools for common Vikunja operations
- Implement skill-based tool selection for better efficiency
- Add support for streaming responses
- Implement conversation memory management
- Add multi-agent collaboration support
- Integrate with Vikunja's permission system for tool authorization

## Support

For issues or questions:
- Check the code in `pkg/modules/ai/`
- Review test files for usage examples
- See `pkg/modules/ai/README.md` for detailed configuration
