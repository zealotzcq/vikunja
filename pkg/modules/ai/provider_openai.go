package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
}

type openAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
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

	return p.makeRequest(ctx, messages, nil)
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

		log.Printf("[OpenAI Tool Def] Name: %s", name)

		openAITools = append(openAITools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}

	log.Printf("[OpenAI] Sending request with %d tools", len(openAITools))

	return p.makeRequest(ctx, messages, openAITools)
}

func (p *OpenAIProvider) makeRequest(ctx context.Context, messages []openAIMessage, tools []openAITool) (string, error) {
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

	if tools != nil {
		reqBody.Tools = tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("[OpenAI Request] %s", string(jsonBody))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.OpenAIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[OpenAI Response] %s", string(body))

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	choice := openAIResp.Choices[0]
	log.Printf("[OpenAI Choice] FinishReason: %s, Content: %s, ToolCalls count: %d",
		choice.FinishReason, choice.Message.Content, len(choice.Message.ToolCalls))

	if len(choice.Message.ToolCalls) > 0 {
		var toolCallStrs []string
		for _, tc := range choice.Message.ToolCalls {
			toolCallStrs = append(toolCallStrs, fmt.Sprintf("TOOL: %s\nINPUT: %s", tc.Function.Name, tc.Function.Arguments))
		}
		return fmt.Sprintf("%s\n%s", choice.Message.Content, toolCallStrs[0]), nil
	}

	return choice.Message.Content, nil
}
