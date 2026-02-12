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
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type ollamaResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: p.config.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	return p.makeRequest(ctx, messages)
}

func (p *OllamaProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
	systemPrompt := p.config.SystemPrompt

	toolDescriptions := "\n\nAvailable tools:\n"
	for _, toolDef := range tools {
		name, _ := toolDef["name"].(string)
		desc, _ := toolDef["description"].(string)
		toolDescriptions += fmt.Sprintf("- %s: %s\n", name, desc)
	}

	systemPrompt += toolDescriptions
	systemPrompt += "\nTo use a tool, format your response as:\nTOOL: tool_name\nINPUT: {json_parameters}"

	messages := []ollamaMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	return p.makeRequest(ctx, messages)
}

func (p *OllamaProvider) makeRequest(ctx context.Context, messages []ollamaMessage) (string, error) {
	url := fmt.Sprintf("%s/api/chat", p.config.OllamaBaseURL)

	reqBody := ollamaRequest{
		Model:    p.config.OllamaModel,
		Messages: messages,
		Stream:   false,
		Options: ollamaOptions{
			Temperature: p.config.Temperature,
			NumPredict:  1000,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		GetLLMLogger().LogExchange("ollama", string(jsonBody), string(body))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	GetLLMLogger().LogExchange("ollama", string(jsonBody), string(body))

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}
