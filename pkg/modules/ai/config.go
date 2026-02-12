package ai

import (
	"encoding/json"
	"fmt"
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
	MaxIterations int      `mapstructure:"max_iterations"`
	Temperature   float64  `mapstructure:"temperature"`
	EnabledSkills []string `mapstructure:"enabled_skills"`
	EnabledTools  []string `mapstructure:"enabled_tools"`
	LLMLog        bool     `mapstructure:"llm_log"`

	// System Prompt
	SystemPrompt string `mapstructure:"system_prompt"`
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
		if viper.IsSet("ai.system_prompt") {
			config.SystemPrompt = viper.GetString("ai.system_prompt")
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

When responding:
1. Be concise and helpful
2. If a user wants to navigate, respond with the action you're taking
3. If a user needs to perform a complex task, break it down
4. Always ask for clarification if the request is ambiguous

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
