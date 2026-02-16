package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/modules/chat_session"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error)
	GenerateWithMessages(ctx context.Context, messages []Message, tools []map[string]interface{}) (*LLMProviderResponse, error)
}

// LLMResponse represents response from an LLM
type LLMResponse struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
}

// LLMProviderResponse represents a structured response from an LLM provider
type LLMProviderResponse struct {
	Content      string             `json:"content"`
	FinishReason string             `json:"finish_reason"`
	ToolCalls    []ProviderToolCall `json:"tool_calls,omitempty"`
}

// ProviderToolCall represents a structured tool call from the provider
type ProviderToolCall struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// AgentContext holds the context for an agent execution
type AgentContext struct {
	UserID           int64                               `json:"user_id"`
	CompanyID        int64                               `json:"company_id"`
	MessageHistory   []Message                           `json:"message_history"`
	CurrentRoute     string                              `json:"current_route"`
	RouteParams      map[string]interface{}              `json:"route_params"`
	SessionData      map[string]interface{}              `json:"session_data"`
	Language         string                              `json:"language"`
	SubordinateStaff []chat_session.SubordinateStaffInfo `json:"subordinate_staff"`

	NavigationInfo   *NavigationInfo                `json:"navigation_info,omitempty"`
	ShouldNavigate   bool                           `json:"should_navigate"`
	ExecutionSteps   []ExecutionStep                `json:"execution_steps"`
	TokensUsed       int                            `json:"tokens_used"`
	QuestionData     string                         `json:"question_data,omitempty"`
	WaitingForAnswer bool                           `json:"waiting_for_answer,omitempty"`
	ButtonNavigation *chat_session.ButtonNavigation `json:"button_navigation,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	Name       string         `json:"name,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"` // Used in tool result messages to reference the tool call
	ToolCalls  []ToolCallInfo `json:"tool_calls,omitempty"`   // Tool call information for assistant messages
}

// ToolCallInfo represents information about a tool call
type ToolCallInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ExecutionStep represents a step in the agent's execution
type ExecutionStep struct {
	StepNumber int    `json:"step_number"`
	Thought    string `json:"thought"`
	Action     string `json:"action"`
	Input      string `json:"input"`
	Output     string `json:"output"`
}

// Agent represents an AI agent
type Agent struct {
	config       *Config
	llmProvider  LLMProvider
	skillManager *SkillManager
	toolManager  *ToolManager
	mu           sync.RWMutex
	initialized  bool
}

var (
	globalAgent *Agent
	agentOnce   sync.Once
)

// GetAgent returns the singleton agent instance
func GetAgent() (*Agent, error) {
	var err error
	agentOnce.Do(func() {
		agent := &Agent{
			skillManager: GetSkillManager(),
			toolManager:  GetToolManager(),
			initialized:  false,
		}
		err = agent.Initialize()
		if err != nil {
			return
		}
		globalAgent = agent
	})

	return globalAgent, err
}

// Initialize initializes the agent with configuration
func (a *Agent) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	a.config = config

	provider, err := a.createLLMProvider()
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}
	a.llmProvider = provider

	if err := RegisterDefaultTools(); err != nil {
		return fmt.Errorf("failed to register default tools: %w", err)
	}

	a.initialized = true
	return nil
}

func (a *Agent) createLLMProvider() (LLMProvider, error) {

	switch a.config.LLMProvider {
	case "openai":

		return NewOpenAIProvider(a.config), nil
	case "ollama":

		return NewOllamaProvider(a.config), nil
	case "mock":

		return NewMockLLMProvider(), nil
	default:

		return NewMockLLMProvider(), nil
	}
}

// ProcessMessage processes a user message and returns the agent's response
func (a *Agent) ProcessMessage(ctx context.Context, agentCtx *AgentContext, message string) (*AgentResponse, error) {
	a.mu.RLock()
	initialized := a.initialized
	a.mu.RUnlock()

	if !initialized {
		if err := a.Initialize(); err != nil {
			return nil, fmt.Errorf("agent not initialized: %w", err)
		}
	}

	if agentCtx == nil {
		agentCtx = &AgentContext{
			MessageHistory: []Message{},
			SessionData:    make(map[string]interface{}),
		}
	}

	response, messages, err := a.runAgentLoop(ctx, agentCtx, message)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	agentCtx.MessageHistory = append(agentCtx.MessageHistory, messages...)

	return response, nil
}

func (a *Agent) runAgentLoop(ctx context.Context, agentCtx *AgentContext, userMessage string) (*AgentResponse, []Message, error) {
	maxIterations := a.config.MaxIterations

	messages := a.buildMessages(agentCtx)

	// Add user message only once at the beginning
	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	// Track which messages to save to history (excluding system prompt)
	messagesToSave := make([]Message, 0)

	for i := 0; i < maxIterations; i++ {
		toolDefinitions := a.toolManager.GetToolDefinitions()

		if len(toolDefinitions) == 0 {
			return nil, nil, fmt.Errorf("no tool definitions available")
		}

		providerResponse, err := a.llmProvider.GenerateWithMessages(ctx, messages, toolDefinitions)

		if err != nil {
			return nil, nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		// If LLM returns content but no tool calls, convert to message_reply tool call
		if providerResponse.Content != "" && len(providerResponse.ToolCalls) == 0 {
			providerResponse.ToolCalls = []ProviderToolCall{
				{
					ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
					Type: "function",
					Name: "message_reply",
					Arguments: map[string]interface{}{
						"content": providerResponse.Content,
					},
				},
			}
		}

		// Process tool calls
		for _, toolCall := range providerResponse.ToolCalls {
			if toolCall.Name == "" {
				GetLLMLogger().LogExchange("agent", "", fmt.Sprintf("Warning: Skipping tool call with empty name, ID: %s, arguments: %v", toolCall.ID, toolCall.Arguments))
				continue
			}

			step := ExecutionStep{
				StepNumber: i + 1,
				Thought:    providerResponse.Content,
				Action:     toolCall.Name,
			}

			inputJSON, _ := json.Marshal(toolCall.Arguments)
			step.Input = string(inputJSON)

			if len(toolCall.Arguments) == 0 {
				GetLLMLogger().LogExchange("agent", "", fmt.Sprintf("Warning: Tool call '%s' has empty arguments", toolCall.Name))
			}

			tool, toolExists := a.toolManager.GetTool(toolCall.Name)

			executionResult, execErr := a.toolManager.ExecuteTool(toolCall.Name, agentCtx, toolCall.Arguments)
			if execErr != nil {
				step.Output = fmt.Sprintf("Error: %s", execErr.Error())
			} else if executionResult != nil {
				if executionResult.Error != "" {
					step.Output = executionResult.Error
				} else {
					step.Output = executionResult.Result
				}

				// Check if tool should stop the loop
				if toolExists && tool.ShouldStopLoop && executionResult.StopCommand == nil {
					executionResult.StopCommand = &ToolStopCommand{
						Response: executionResult.Result,
					}
				}
			}

			agentCtx.ExecutionSteps = append(agentCtx.ExecutionSteps, step)

			toolCallID := toolCall.ID
			if toolCallID == "" {
				toolCallID = fmt.Sprintf("call_%d", time.Now().UnixNano())
			}

			assistantMsg := Message{
				Role:    "assistant",
				Content: providerResponse.Content,
				ToolCalls: []ToolCallInfo{
					{
						ID:   toolCallID,
						Type: "function",
						Name: toolCall.Name,
						Arguments: func() string {
							if argsJSON, err := json.Marshal(toolCall.Arguments); err == nil {
								return string(argsJSON)
							}
							return "{}"
						}(),
					},
				},
			}
			messages = append(messages, assistantMsg)

			toolResultMsg := Message{
				Role:       "tool",
				ToolCallID: toolCallID,
				Name:       toolCall.Name,
				Content:    step.Output,
			}
			messages = append(messages, toolResultMsg)

			messagesToSave = append(messagesToSave, assistantMsg, toolResultMsg)

			if executionResult != nil && executionResult.StopCommand != nil {
				metadata := executionResult.StopCommand.Metadata
				if metadata == nil {
					metadata = make(map[string]interface{})
				}
				metadata["execution_steps"] = agentCtx.ExecutionSteps
				metadata["tokens_used"] = agentCtx.TokensUsed

				return &AgentResponse{
					Content:          executionResult.StopCommand.Response,
					NavigationInfo:   agentCtx.NavigationInfo,
					ShouldNavigate:   agentCtx.ShouldNavigate,
					ExecutionSteps:   agentCtx.ExecutionSteps,
					TokensUsed:       agentCtx.TokensUsed,
					Metadata:         metadata,
					ButtonNavigation: agentCtx.ButtonNavigation,
				}, messagesToSave, nil
			}
		}
	}

	return &AgentResponse{
		Content:          "I apologize, but I couldn't complete your request. Please try again.",
		ExecutionSteps:   agentCtx.ExecutionSteps,
		TokensUsed:       agentCtx.TokensUsed,
		ButtonNavigation: agentCtx.ButtonNavigation,
	}, messagesToSave, nil
}

func (a *Agent) buildPrompt(agentCtx *AgentContext, userMessage string, iteration int) string {
	prompt := a.buildSystemPrompt(agentCtx) + "\n\n"

	if len(agentCtx.MessageHistory) > 0 {
		prompt += "Conversation history:\n"
		for _, msg := range agentCtx.MessageHistory {
			prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
		prompt += "\n"
	}

	if agentCtx.CurrentRoute != "" {
		prompt += fmt.Sprintf("Current page: %s\n", agentCtx.CurrentRoute)
		if len(agentCtx.RouteParams) > 0 {
			prompt += fmt.Sprintf("Page params: %v\n", agentCtx.RouteParams)
		}
		prompt += "\n"
	}

	if iteration > 0 {
		prompt += "Previous actions:\n"
		for _, step := range agentCtx.ExecutionSteps {
			prompt += fmt.Sprintf("- %s: %s -> %s\n", step.Action, step.Input, step.Output)
		}
		prompt += "\n"
	}

	prompt += fmt.Sprintf("User: %s\n", userMessage)
	prompt += "\nPlease respond helpfully. If you need to use a tool, format your response as:\n"
	prompt += "TOOL: tool_name\nINPUT: {json_parameters}\n"
	prompt += "Otherwise, just provide your response directly."

	return prompt
}

func (a *Agent) buildSystemPrompt(agentCtx *AgentContext) string {
	var sb strings.Builder

	sb.WriteString(a.config.SystemPrompt)
	sb.WriteString("\n\n")

	sb.WriteString("## Environment Information\n")
	sb.WriteString(fmt.Sprintf("- Date: %s\n", time.Now().Format("2006-01-02")))
	if agentCtx.UserID > 0 {
		sb.WriteString(fmt.Sprintf("- User ID: %d\n", agentCtx.UserID))
	}
	if agentCtx.CompanyID > 0 {
		sb.WriteString(fmt.Sprintf("- Company ID: %d\n", agentCtx.CompanyID))
	}
	if agentCtx.Language != "" {
		sb.WriteString(fmt.Sprintf("- Language: %s\n", agentCtx.Language))
	}
	if agentCtx.CurrentRoute != "" {
		sb.WriteString(fmt.Sprintf("- Current Route: %s\n", agentCtx.CurrentRoute))
		if len(agentCtx.RouteParams) > 0 {
			sb.WriteString(fmt.Sprintf("- Route Params: %v\n", agentCtx.RouteParams))
		}
	}

	if len(agentCtx.SubordinateStaff) > 0 {
		sb.WriteString("\n- Subordinate Staff:\n")
		staffJSON, _ := json.Marshal(agentCtx.SubordinateStaff)
		sb.WriteString(fmt.Sprintf("  %s\n", string(staffJSON)))
	}

	sb.WriteString("\n")

	return sb.String()
}

func (a *Agent) buildMessages(agentCtx *AgentContext) []Message {
	messages := []Message{
		{
			Role:    "system",
			Content: a.buildSystemPrompt(agentCtx),
		},
	}

	for _, msg := range agentCtx.MessageHistory {
		messages = append(messages, msg)
	}

	return messages
}

func messagesToPrompt(messages []Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	return sb.String()
}

// AgentResponse represents the agent's response
type AgentResponse struct {
	Content          string                         `json:"content"`
	NavigationInfo   *NavigationInfo                `json:"navigation_info,omitempty"`
	ShouldNavigate   bool                           `json:"should_navigate"`
	ExecutionSteps   []ExecutionStep                `json:"execution_steps,omitempty"`
	TokensUsed       int                            `json:"tokens_used"`
	Metadata         map[string]interface{}         `json:"metadata,omitempty"`
	ButtonNavigation *chat_session.ButtonNavigation `json:"button_navigation,omitempty"`
}

// Reset resets the agent state
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.skillManager != nil {
		a.skillManager = GetSkillManager()
	}
	if a.toolManager != nil {
		a.toolManager = GetToolManager()
	}
}

// GetStats returns statistics about the agent
func (a *Agent) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"initialized":    a.initialized,
		"llm_provider":   a.config.LLMProvider,
		"num_skills":     len(a.skillManager.GetAllSkills()),
		"num_tools":      len(a.toolManager.GetAllTools()),
		"max_iterations": a.config.MaxIterations,
		"temperature":    a.config.Temperature,
	}
}
