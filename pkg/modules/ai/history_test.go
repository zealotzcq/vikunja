package ai

import (
	"context"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/modules/keyvalue"
)

func init() {
	keyvalue.InitStorage()
}

func TestGenerateAgentResponseWithHistory(t *testing.T) {
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

	userID := int64(123)
	companyID := int64(0)

	chat_session.GetDefault().ClearSession(userID, companyID)

	historyMessages := []chat_session.Message{
		{
			ID:        "msg_1",
			Role:      "user",
			Content:   "Hello",
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		},
		{
			ID:        "msg_2",
			Role:      "assistant",
			Content:   "Hi there! How can I help you?",
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		},
		{
			ID:        "msg_3",
			Role:      "user",
			Content:   "Show me projects",
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		},
	}

	for _, msg := range historyMessages {
		if err := chat_session.GetDefault().AddMessage(userID, companyID, msg); err != nil {
			t.Fatalf("Failed to add message to session: %v", err)
		}
	}

	session, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	agentCtx := &AgentContext{
		UserID:         userID,
		CurrentRoute:   "projects.index",
		RouteParams:    nil,
		SessionData:    make(map[string]interface{}),
		MessageHistory: make([]Message, 0, len(session.Messages)),
	}

	for _, msg := range session.Messages {
		agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	response, err := agent.ProcessMessage(
		context.Background(),
		agentCtx,
		"What tasks do I have?",
	)

	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Content == "" {
		t.Error("Response content should not be empty")
	}

	chat_session.GetDefault().ClearSession(userID, companyID)
}

func TestAgentContextHistory(t *testing.T) {
	userID := int64(456)
	companyID := int64(0)

	chat_session.GetDefault().ClearSession(userID, companyID)

	_, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	historyMessages := []chat_session.Message{
		{
			ID:        "msg_1",
			Role:      "user",
			Content:   "First message",
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		},
		{
			ID:        "msg_2",
			Role:      "assistant",
			Content:   "First response",
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		},
	}

	for _, msg := range historyMessages {
		if err := chat_session.GetDefault().AddMessage(userID, companyID, msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	updatedSession, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if len(updatedSession.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(updatedSession.Messages))
	}

	if updatedSession.Messages[0].Role != "user" {
		t.Errorf("First message role should be 'user', got '%s'", updatedSession.Messages[0].Role)
	}

	if updatedSession.Messages[1].Role != "assistant" {
		t.Errorf("Second message role should be 'assistant', got '%s'", updatedSession.Messages[1].Role)
	}

	agentCtx := &AgentContext{
		UserID:         userID,
		CurrentRoute:   "",
		RouteParams:    nil,
		SessionData:    make(map[string]interface{}),
		MessageHistory: make([]Message, 0, len(updatedSession.Messages)),
	}

	for _, msg := range updatedSession.Messages {
		agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if len(agentCtx.MessageHistory) != 2 {
		t.Errorf("Expected 2 history messages, got %d", len(agentCtx.MessageHistory))
	}

	if agentCtx.MessageHistory[0].Role != "user" {
		t.Errorf("First history message role should be 'user', got '%s'", agentCtx.MessageHistory[0].Role)
	}

	if agentCtx.MessageHistory[1].Role != "assistant" {
		t.Errorf("Second history message role should be 'assistant', got '%s'", agentCtx.MessageHistory[1].Role)
	}

	chat_session.GetDefault().ClearSession(userID, companyID)
}

func TestAgentContextWithEmptyHistory(t *testing.T) {
	userID := int64(789)
	companyID := int64(0)

	chat_session.GetDefault().ClearSession(userID, companyID)

	_, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	agentCtx := &AgentContext{
		UserID:         userID,
		CurrentRoute:   "",
		RouteParams:    nil,
		SessionData:    make(map[string]interface{}),
		MessageHistory: make([]Message, 0),
	}

	if len(agentCtx.MessageHistory) != 0 {
		t.Errorf("Expected 0 history messages, got %d", len(agentCtx.MessageHistory))
	}

	chat_session.GetDefault().ClearSession(userID, companyID)
}
