package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error)
}

// AgentContext holds the context for an agent execution
type AgentContext struct {
	UserID         int64                  `json:"user_id"`
	MessageHistory []Message              `json:"message_history"`
	CurrentRoute   string                 `json:"current_route"`
	RouteParams    map[string]interface{} `json:"route_params"`
	SessionData    map[string]interface{} `json:"session_data"`

	NavigationInfo *NavigationInfo `json:"navigation_info,omitempty"`
	ShouldNavigate bool            `json:"should_navigate"`
	ExecutionSteps []ExecutionStep `json:"execution_steps"`
	TokensUsed     int             `json:"tokens_used"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	log.Printf("[AI] Creating LLM provider: %s", a.config.LLMProvider)

	switch a.config.LLMProvider {
	case "openai":
		log.Printf("[AI] Created OpenAI provider - Model: %s, BaseURL: %s", a.config.OpenAIModel, a.config.OpenAIBaseURL)
		return NewOpenAIProvider(a.config), nil
	case "ollama":
		log.Printf("[AI] Created Ollama provider - Model: %s, BaseURL: %s", a.config.OllamaModel, a.config.OllamaBaseURL)
		return NewOllamaProvider(a.config), nil
	case "mock":
		log.Printf("[AI] Created Mock provider")
		return NewMockLLMProvider(), nil
	default:
		log.Printf("[AI] Unknown provider '%s', falling back to Mock", a.config.LLMProvider)
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

	agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
		Role:    "user",
		Content: message,
	})

	response, err := a.runAgentLoop(ctx, agentCtx, message)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
		Role:    "assistant",
		Content: response.Content,
	})

	return response, nil
}

func (a *Agent) runAgentLoop(ctx context.Context, agentCtx *AgentContext, userMessage string) (*AgentResponse, error) {
	maxIterations := a.config.MaxIterations

	for i := 0; i < maxIterations; i++ {
		prompt := a.buildPrompt(agentCtx, userMessage, i)

		tools := a.toolManager.GetEnabledTools()
		toolDefinitions := a.toolManager.GetToolDefinitions()

		var llmResponse string
		var err error

		if len(tools) > 0 {
			llmResponse, err = a.llmProvider.GenerateWithTools(ctx, prompt, toolDefinitions)
		} else {
			llmResponse, err = a.llmProvider.Generate(ctx, prompt)
		}

		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		toolCall, err := a.parseToolCall(llmResponse)
		if err != nil {
			continue
		}

		if toolCall == nil {
			return &AgentResponse{
				Content:        llmResponse,
				NavigationInfo: agentCtx.NavigationInfo,
				ShouldNavigate: agentCtx.ShouldNavigate,
				ExecutionSteps: agentCtx.ExecutionSteps,
				TokensUsed:     agentCtx.TokensUsed,
			}, nil
		}

		step := ExecutionStep{
			StepNumber: i + 1,
			Thought:    llmResponse,
			Action:     toolCall.Name,
			Input:      toolCall.InputJSON,
		}

		toolResult, err := a.toolManager.ExecuteTool(toolCall.Name, agentCtx, toolCall.Input)
		if err != nil {
			step.Output = fmt.Sprintf("Error: %s", err.Error())
		} else {
			step.Output = toolResult
		}

		agentCtx.ExecutionSteps = append(agentCtx.ExecutionSteps, step)
	}

	return &AgentResponse{
		Content:        "I apologize, but I couldn't complete your request. Please try again.",
		ExecutionSteps: agentCtx.ExecutionSteps,
		TokensUsed:     agentCtx.TokensUsed,
	}, nil
}

func (a *Agent) buildPrompt(agentCtx *AgentContext, userMessage string, iteration int) string {
	prompt := a.config.SystemPrompt + "\n\n"

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

type ToolCall struct {
	Name      string
	Input     map[string]interface{}
	InputJSON string
}

func (a *Agent) parseToolCall(response string) (*ToolCall, error) {
	lines := strings.Split(response, "\n")

	var toolName string
	var inputLine string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TOOL:") {
			toolName = strings.TrimSpace(strings.TrimPrefix(line, "TOOL:"))
		} else if strings.HasPrefix(line, "INPUT:") {
			inputLine = strings.TrimSpace(strings.TrimPrefix(line, "INPUT:"))
		}
	}

	if toolName == "" {
		return nil, nil
	}

	if inputLine == "" {
		return &ToolCall{Name: toolName, Input: make(map[string]interface{})}, nil
	}

	var input map[string]interface{}
	if err := json.Unmarshal([]byte(inputLine), &input); err != nil {
		return nil, fmt.Errorf("failed to parse tool input: %w", err)
	}

	return &ToolCall{
		Name:      toolName,
		Input:     input,
		InputJSON: inputLine,
	}, nil
}

// AgentResponse represents the agent's response
type AgentResponse struct {
	Content        string                 `json:"content"`
	NavigationInfo *NavigationInfo        `json:"navigation_info,omitempty"`
	ShouldNavigate bool                   `json:"should_navigate"`
	ExecutionSteps []ExecutionStep        `json:"execution_steps,omitempty"`
	TokensUsed     int                    `json:"tokens_used"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
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
