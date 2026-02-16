package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements LLMProvider for Ollama
type OllamaProvider struct {
	config *Config
}

func NewOllamaProvider(config *Config) *OllamaProvider {
	return &OllamaProvider{config: config}
}

type ollamaRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"num_predict,omitempty"`
	Tools       []ollamaTool    `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
}

type ollamaMessage struct {
	Role       string           `json:"role"`
	Name       string           `json:"name,omitempty"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type ollamaToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFunction `json:"function"`
}

type ollamaFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ollamaResponse struct {
	Message      ollamaMessage `json:"message"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: p.config.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	response, err := p.makeRequest(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (p *OllamaProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: p.config.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	ollamaTools := make([]ollamaTool, 0, len(tools))
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		desc, _ := toolDef["description"].(string)
		params, _ := toolDef["parameters"].(map[string]interface{})

		ollamaTools = append(ollamaTools, ollamaTool{
			Type: "function",
			Function: ollamaFunction{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}

	response, err := p.makeRequest(ctx, messages, ollamaTools)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (p *OllamaProvider) GenerateWithMessages(ctx context.Context, messages []Message, tools []map[string]interface{}) (*LLMProviderResponse, error) {
	ollamaMessages := make([]ollamaMessage, 0, len(messages))

	for _, msg := range messages {
		ollamaMessage := ollamaMessage{
			Role:       msg.Role,
			Name:       msg.Name,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.ToolCalls) > 0 {
			ollamaMessage.ToolCalls = make([]ollamaToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				var arguments map[string]interface{}
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &arguments); err != nil {
						GetLLMLogger().LogExchange("ollama", "", fmt.Sprintf("Failed to unmarshal tool arguments in message: %v, arguments: %s", err, tc.Arguments))
						arguments = make(map[string]interface{})
					}
				} else {
					arguments = make(map[string]interface{})
				}

				ollamaMessage.ToolCalls[i] = ollamaToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: ollamaFunctionCall{
						Name:      tc.Name,
						Arguments: arguments,
					},
				}
			}
		}

		ollamaMessages = append(ollamaMessages, ollamaMessage)
	}

	ollamaTools := make([]ollamaTool, 0, len(tools))
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		desc, _ := toolDef["description"].(string)
		params, _ := toolDef["parameters"].(map[string]interface{})

		ollamaTool := ollamaTool{
			Type: "function",
			Function: ollamaFunction{
				Name:        name,
				Description: desc,
			},
		}

		if params != nil {
			ollamaTool.Function.Parameters = params
		} else {
			ollamaTool.Function.Parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		ollamaTools = append(ollamaTools, ollamaTool)
	}

	return p.makeRequest(ctx, ollamaMessages, ollamaTools)
}

func (p *OllamaProvider) makeRequest(ctx context.Context, messages []ollamaMessage, tools []ollamaTool) (*LLMProviderResponse, error) {
	url := fmt.Sprintf("%s/api/chat", p.config.OllamaBaseURL)

	reqBody := ollamaRequest{
		Model:       p.config.OllamaModel,
		Messages:    messages,
		Stream:      false,
		Temperature: p.config.Temperature,
		MaxTokens:   32000,
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
		GetLLMLogger().LogExchange("ollama", string(jsonBody), string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		GetLLMLogger().LogExchange("ollama", string(jsonBody), string(body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	GetLLMLogger().LogExchange("ollama", string(jsonBody), string(body))

	response := &LLMProviderResponse{
		Content:      ollamaResp.Message.Content,
		FinishReason: ollamaResp.FinishReason,
	}

	if response.FinishReason == "" {
		response.FinishReason = "stop"
	}

	if len(ollamaResp.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ProviderToolCall, 0, len(ollamaResp.Message.ToolCalls))
		for _, tc := range ollamaResp.Message.ToolCalls {
			if tc.Function.Name == "" {
				GetLLMLogger().LogExchange("ollama", "", fmt.Sprintf("Warning: Skipping tool call with empty name, ID: %s, Type: %s, Arguments: %v", tc.ID, tc.Type, tc.Function.Arguments))
				continue
			}

			response.ToolCalls = append(response.ToolCalls, ProviderToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	return response, nil
}
