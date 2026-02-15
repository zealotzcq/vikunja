package ai

import (
	"testing"

	"code.vikunja.io/api/pkg/modules/chat_session"
)

func TestBuildSystemPromptWithSubordinateStaff(t *testing.T) {
	testConfig := &Config{
		LLMProvider: "mock",
	}
	_ = ValidateConfig(testConfig)

	provider := NewMockLLMProvider()

	agent := &Agent{}
	agent.mu.Lock()
	agent.config = testConfig
	agent.llmProvider = provider
	agent.skillManager = GetSkillManager()
	agent.toolManager = GetToolManager()
	agent.initialized = true
	agent.mu.Unlock()

	agentCtx := &AgentContext{
		UserID:       123,
		CompanyID:    456,
		Language:     "en",
		CurrentRoute: "tasks.index",
		SubordinateStaff: []chat_session.SubordinateStaffInfo{
			{
				UserID:    789,
				Username:  "john_doe",
				Name:      "John Doe",
				ProjectID: 101,
			},
			{
				UserID:    790,
				Username:  "jane_smith",
				Name:      "Jane Smith",
				ProjectID: 102,
			},
		},
	}

	prompt := agent.buildSystemPrompt(agentCtx)

	if prompt == "" {
		t.Fatal("System prompt should not be empty")
	}

	if !containsString(prompt, "Subordinate Staff:") {
		t.Error("System prompt should contain 'Subordinate Staff:'")
	}

	if !containsString(prompt, `"user_id":789`) {
		t.Error("System prompt should contain subordinate staff user_id 789")
	}

	if !containsString(prompt, `"username":"john_doe"`) {
		t.Error("System prompt should contain subordinate staff username john_doe")
	}

	if !containsString(prompt, `"name":"John Doe"`) {
		t.Error("System prompt should contain subordinate staff name John Doe")
	}

	if !containsString(prompt, `"project_id":101`) {
		t.Error("System prompt should contain subordinate staff project_id 101")
	}

	if !containsString(prompt, `"user_id":790`) {
		t.Error("System prompt should contain subordinate staff user_id 790")
	}

	t.Logf("Generated system prompt:\n%s", prompt)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
