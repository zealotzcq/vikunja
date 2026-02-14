package ai

import (
	"context"
	"testing"
)

func TestMockLLMProvider(t *testing.T) {
	provider := NewMockLLMProvider()

	tests := []struct {
		name        string
		prompt      string
		wantContain string
	}{
		{
			name:        "help request",
			prompt:      "What can you help me with?",
			wantContain: "AI 助手",
		},
		{
			name:        "project navigation",
			prompt:      "Show me projects",
			wantContain: "TOOL: navigate",
		},
		{
			name:        "task creation",
			prompt:      "Create a task",
			wantContain: "创建任务",
		},
		{
			name:        "home navigation",
			prompt:      "Go to home",
			wantContain: "TOOL: navigate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := provider.Generate(ctx, tt.prompt)
			if err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}

			if len(resp) == 0 {
				t.Error("Generate() returned empty response")
			}

			if tt.wantContain != "" && !contains(resp, tt.wantContain) {
				t.Errorf("Generate() response should contain %q, got %q", tt.wantContain, resp)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

func TestToolManager(t *testing.T) {
	tm := GetToolManager()

	_ = RegisterDefaultTools()

	t.Run("register tool", func(t *testing.T) {
		tool := &Tool{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
			},
			Execute: func(ctx *AgentContext, params map[string]interface{}) (*ToolExecutionResult, error) {
				return &ToolExecutionResult{
					Result: "test result",
				}, nil
			},
		}

		err := tm.RegisterTool(tool)
		if err != nil {
			t.Errorf("RegisterTool() error = %v", err)
		}

		got, exists := tm.GetTool("test_tool")
		if !exists {
			t.Error("GetTool() should find the registered tool")
		}

		if got.Name != tool.Name {
			t.Errorf("GetTool() name = %v, want %v", got.Name, tool.Name)
		}
	})

	t.Run("execute tool", func(t *testing.T) {
		ctx := &AgentContext{
			SessionData: make(map[string]interface{}),
		}

		result, err := tm.ExecuteTool("navigate", ctx, map[string]interface{}{
			"route_name": "test.route",
			"content":    "Navigating to test route",
		})

		if err != nil {
			t.Errorf("ExecuteTool() error = %v", err)
		}

		if result == nil {
			t.Error("ExecuteTool() should return a result")
			return
		}

		if result.StopCommand == nil {
			t.Error("ExecuteTool() should set StopCommand for navigate tool")
		}

		if !ctx.ShouldNavigate {
			t.Error("ExecuteTool() should set ShouldNavigate to true")
		}

		if ctx.NavigationInfo == nil {
			t.Error("ExecuteTool() should set NavigationInfo")
		}
	})

	t.Run("get tool definitions", func(t *testing.T) {
		defs := tm.GetToolDefinitions()

		if len(defs) == 0 {
			t.Error("GetToolDefinitions() should return at least one tool")
		}

		for _, def := range defs {
			if _, ok := def["name"]; !ok {
				t.Error("Tool definition should have 'name' key")
			}
		}
	})

	tm.UnregisterTool("test_tool")
}

func TestAgentContext(t *testing.T) {
	ctx := &AgentContext{
		UserID:       123,
		CurrentRoute: "test.route",
		RouteParams:  make(map[string]interface{}),
		SessionData:  make(map[string]interface{}),
	}

	t.Run("add message to history", func(t *testing.T) {
		ctx.MessageHistory = append(ctx.MessageHistory, Message{
			Role:    "user",
			Content: "test message",
		})

		if len(ctx.MessageHistory) != 1 {
			t.Errorf("MessageHistory length = %v, want 1", len(ctx.MessageHistory))
		}
	})

	t.Run("add execution step", func(t *testing.T) {
		step := ExecutionStep{
			StepNumber: 1,
			Thought:    "test thought",
			Action:     "test action",
			Input:      "test input",
			Output:     "test output",
		}

		ctx.ExecutionSteps = append(ctx.ExecutionSteps, step)

		if len(ctx.ExecutionSteps) != 1 {
			t.Errorf("ExecutionSteps length = %v, want 1", len(ctx.ExecutionSteps))
		}
	})
}

func TestConfig(t *testing.T) {
	t.Run("load config", func(t *testing.T) {
		config, err := LoadConfig()
		if err != nil {
			t.Errorf("LoadConfig() error = %v", err)
		}

		if config == nil {
			t.Error("LoadConfig() should return a non-nil config")
		}

		if config.LLMProvider == "" {
			t.Error("LoadConfig() should set a default LLM provider")
		}
	})

	t.Run("validate config", func(t *testing.T) {
		config := &Config{
			LLMProvider: "mock",
		}

		err := ValidateConfig(config)
		if err != nil {
			t.Errorf("ValidateConfig() error = %v", err)
		}
	})

	t.Run("validate config with openai", func(t *testing.T) {
		config := &Config{
			LLMProvider: "openai",
			OpenAIKey:   "",
		}

		err := ValidateConfig(config)
		if err == nil {
			t.Error("ValidateConfig() should require OpenAI key")
		}
	})
}

func TestAgent(t *testing.T) {
	t.Run("get agent with mock provider", func(t *testing.T) {
		testConfig := &Config{
			LLMProvider: "mock",
		}
		_ = ValidateConfig(testConfig)

		agent := &Agent{
			config:       testConfig,
			llmProvider:  NewMockLLMProvider(),
			skillManager: GetSkillManager(),
			toolManager:  GetToolManager(),
			initialized:  true,
		}

		if agent == nil {
			t.Error("Agent should not be nil")
		}

		stats := agent.GetStats()
		if stats == nil {
			t.Error("GetStats() should return non-nil stats")
		}

		if stats["initialized"] != true {
			t.Error("Agent should be initialized")
		}

		if stats["llm_provider"] != "mock" {
			t.Errorf("LLM provider should be mock, got %v", stats["llm_provider"])
		}
	})
}
