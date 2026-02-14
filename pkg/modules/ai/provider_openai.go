package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider implements LLMProvider for OpenAI
type OpenAIProvider struct {
	config *Config
}

func NewOpenAIProvider(config *Config) *OpenAIProvider {
	return &OpenAIProvider{config: config}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Name       string           `json:"name,omitempty"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // Used in tool result messages to reference the tool call
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []openAIMessage{
		{Role: "system", Content: p.config.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	response, err := p.makeRequest(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
	messages := []openAIMessage{
		{Role: "system", Content: p.config.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	openAITools := make([]openAITool, 0, len(tools))
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		desc, _ := toolDef["description"].(string)
		params, _ := toolDef["parameters"].(map[string]interface{})

		openAITools = append(openAITools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}

	response, err := p.makeRequest(ctx, messages, openAITools)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (p *OpenAIProvider) GenerateWithMessages(ctx context.Context, messages []Message, tools []map[string]interface{}) (*LLMProviderResponse, error) {
	openAIMessages := make([]openAIMessage, 0, len(messages))

	for _, msg := range messages {
		openAIMessage := openAIMessage{
			Role:       msg.Role,
			Name:       msg.Name,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		// Convert ToolCalls to OpenAI format
		if len(msg.ToolCalls) > 0 {
			openAIMessage.ToolCalls = make([]openAIToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				openAIMessage.ToolCalls[i] = openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openAIFunctionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}

		openAIMessages = append(openAIMessages, openAIMessage)
	}

	openAITools := make([]openAITool, 0, len(tools))
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		desc, _ := toolDef["description"].(string)
		params, _ := toolDef["parameters"].(map[string]interface{})

		openAITool := openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        name,
				Description: desc,
			},
		}

		if params != nil {
			openAITool.Function.Parameters = params
		} else {
			openAITool.Function.Parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		openAITools = append(openAITools, openAITool)
	}

	return p.makeRequest(ctx, openAIMessages, openAITools)
}

func (p *OpenAIProvider) makeRequest(ctx context.Context, messages []openAIMessage, tools []openAITool) (*LLMProviderResponse, error) {
	baseURL := p.config.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	url := fmt.Sprintf("%s/chat/completions", baseURL)

	reqBody := openAIRequest{
		Model:       p.config.OpenAIModel,
		Messages:    messages,
		Temperature: p.config.Temperature,
		MaxTokens:   1000,
	}

	if tools != nil && len(tools) > 0 {
		reqBody.Tools = tools
		reqBody.ToolChoice = "auto"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.OpenAIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		GetLLMLogger().LogExchange("openai", string(jsonBody), string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		GetLLMLogger().LogExchange("openai", string(jsonBody), string(body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	GetLLMLogger().LogExchange("openai", string(jsonBody), string(body))

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := openAIResp.Choices[0]

	response := &LLMProviderResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}

	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ProviderToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			if tc.Function.Name == "" {
				GetLLMLogger().LogExchange("openai", "", fmt.Sprintf("Warning: Skipping tool call with empty name, ID: %s, Type: %s, Arguments: %s", tc.ID, tc.Type, tc.Function.Arguments))
				continue
			}

			var arguments map[string]interface{}
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					GetLLMLogger().LogExchange("openai", "", fmt.Sprintf("Failed to unmarshal tool arguments: %v, arguments: %s", err, tc.Function.Arguments))
					arguments = make(map[string]interface{})
				}
			} else {
				arguments = make(map[string]interface{})
			}

			response.ToolCalls = append(response.ToolCalls, ProviderToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: arguments,
			})
		}
	}

	return response, nil
}
