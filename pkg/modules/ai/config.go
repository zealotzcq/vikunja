package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/viper"
)

// Config holds the AI agent configuration
type Config struct {
	// LLM Provider configuration
	LLMProvider string `mapstructure:"llm_provider"` // "openai", "anthropic", "ollama", etc.

	// OpenAI Configuration
	OpenAIKey     string `mapstructure:"key"`
	OpenAIModel   string `mapstructure:"model"`
	OpenAIBaseURL string `mapstructure:"base_url"`

	// Ollama Configuration
	OllamaBaseURL string `mapstructure:"base_url"`
	OllamaModel   string `mapstructure:"model"`

	// Agent Configuration
	MaxIterations   int      `mapstructure:"max_iterations"`
	Temperature     float64  `mapstructure:"temperature"`
	EnabledSkills   []string `mapstructure:"enabled_skills"`
	EnabledTools    []string `mapstructure:"enabled_tools"`
	LLMLog          bool     `mapstructure:"llm_log"`
	LLMLogPath      string   `mapstructure:"llm_log_path"`
	LLMLogBriefMode bool     `mapstructure:"llm_log_brief_mode"`

	// System Prompt File (path to file containing the system prompt)
	SystemPromptFile string `mapstructure:"system_prompt_file"`
	SystemPrompt     string

	// Prompts Directory (path to directory containing tool prompt files)
	PromptsDir string `mapstructure:"prompts_dir"`
}

var (
	config     *Config
	configOnce sync.Once
)

// LoadConfig loads the AI configuration from config file
func LoadConfig() (*Config, error) {
	var err error
	configOnce.Do(func() {
		config = &Config{
			LLMProvider: viper.GetString("ai.llm_provider"),
		}

		// Load OpenAI config
		if viper.IsSet("ai.openai.key") {
			config.OpenAIKey = viper.GetString("ai.openai.key")
		}
		if viper.IsSet("ai.openai.model") {
			config.OpenAIModel = viper.GetString("ai.openai.model")
		}
		if viper.IsSet("ai.openai.base_url") {
			config.OpenAIBaseURL = viper.GetString("ai.openai.base_url")
		}

		// Load Ollama config
		if viper.IsSet("ai.ollama.base_url") {
			config.OllamaBaseURL = viper.GetString("ai.ollama.base_url")
		}
		if viper.IsSet("ai.ollama.model") {
			config.OllamaModel = viper.GetString("ai.ollama.model")
		}

		// Load agent config
		if viper.IsSet("ai.max_iterations") {
			config.MaxIterations = viper.GetInt("ai.max_iterations")
		}
		if viper.IsSet("ai.temperature") {
			config.Temperature = viper.GetFloat64("ai.temperature")
		}
		if viper.IsSet("ai.enabled_skills") {
			config.EnabledSkills = viper.GetStringSlice("ai.enabled_skills")
		}
		if viper.IsSet("ai.enabled_tools") {
			config.EnabledTools = viper.GetStringSlice("ai.enabled_tools")
		}
		if viper.IsSet("ai.llm_log") {
			config.LLMLog = viper.GetBool("ai.llm_log")
		}
		if viper.IsSet("ai.llm_log_path") {
			config.LLMLogPath = viper.GetString("ai.llm_log_path")
		}
		if viper.IsSet("ai.llm_log_brief_mode") {
			config.LLMLogBriefMode = viper.GetBool("ai.llm_log_brief_mode")
		}

		systemPromptFile := "./soul.md"
		if viper.IsSet("ai.system_prompt_file") {
			systemPromptFile = viper.GetString("ai.system_prompt_file")
		}
		config.SystemPromptFile = systemPromptFile
		promptContent, err := os.ReadFile(systemPromptFile)
		if err == nil {
			config.SystemPrompt = string(promptContent)
		} else {
			config.SystemPrompt = getDefaultSystemPrompt()
		}

		// Set defaults
		if config.LLMProvider == "" {
			config.LLMProvider = "mock"
		}
		if config.OpenAIModel == "" {
			config.OpenAIModel = "gpt-4o-mini"
		}
		if config.OllamaBaseURL == "" {
			config.OllamaBaseURL = "http://localhost:11434"
		}
		if config.OllamaModel == "" {
			config.OllamaModel = "llama2"
		}
		if config.MaxIterations == 0 {
			config.MaxIterations = 10
		}
		if config.Temperature == 0 {
			config.Temperature = 0.7
		}
		if config.SystemPrompt == "" {
			config.SystemPrompt = getDefaultSystemPrompt()
		}
		if config.EnabledSkills == nil {
			config.EnabledSkills = []string{}
		}
		if config.EnabledTools == nil {
			config.EnabledTools = []string{}
		}
		if config.LLMLogPath == "" {
			config.LLMLogPath = "llmlog"
		}
		if viper.IsSet("ai.prompts_dir") {
			config.PromptsDir = viper.GetString("ai.prompts_dir")
		}
		if config.PromptsDir == "" {
			config.PromptsDir = "./prompts"
		}

		err = nil
	})
	return config, err
}

func getDefaultSystemPrompt() string {
	return `You are a helpful task management assistant for Vikunja. You help users manage their tasks, projects, teams, and labels efficiently.

Your capabilities include:
- Navigating to different pages (projects, tasks, teams, labels, favorites, home)
- Creating and managing tasks
- Organizing projects
- Managing team collaboration
- Working with labels and filters

CRITICAL: You MUST use tools to respond to user requests. Do NOT provide text responses directly. You MUST call a tool for every interaction:
- Use available tools to perform actions
- When you have completed your work, use the 'finish_task' tool to send your response to the user
- The 'finish_task' tool is the ONLY tool that ends the conversation
- The 'finish_task' tool can include optional navigation (route_name, params) or button navigation

When responding:
1. Always use tools to perform actions and provide responses
2. Be concise and helpful in your finish_task response
3. If a user wants to navigate, include the navigation in the finish_task call
4. If a user needs to perform a complex task, break it down and use tools step by step
5. Always ask for clarification using the 'question' tool if the request is ambiguous

Available tools and skills will be provided dynamically. Use them appropriately to fulfill user requests.`
}

// GetConfig returns the loaded configuration
func GetConfig() *Config {
	if config == nil {
		c, _ := LoadConfig()
		return c
	}
	return config
}

// UpdateConfig updates specific configuration values
func UpdateConfig(updates map[string]interface{}) error {
	c := GetConfig()
	data, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}
	return json.Unmarshal(data, c)
}

// ValidateConfig validates the configuration
func ValidateConfig(c *Config) error {
	if c.LLMProvider == "openai" && c.OpenAIKey == "" {
		return fmt.Errorf("openai API key is required when using openai provider")
	}
	if c.LLMProvider == "" {
		return fmt.Errorf("LLM provider must be specified")
	}
	if c.MaxIterations <= 0 {
		c.MaxIterations = 10
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		c.Temperature = 0.7
	}
	return nil
}
