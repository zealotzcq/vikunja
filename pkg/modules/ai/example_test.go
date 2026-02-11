package ai_test

import (
	"fmt"

	"code.vikunja.io/api/pkg/modules/ai"
)

// This file contains example code showing how to use the AI agent system.

func ExampleGetToolManager() {
	// Get the tool manager
	tm := ai.GetToolManager()

	// Register a custom tool
	tool := &ai.Tool{
		Name:        "example_tool",
		Description: "An example tool",
		Execute: func(ctx *ai.AgentContext, params map[string]interface{}) (string, error) {
			return "Example executed", nil
		},
	}

	_ = tm.RegisterTool(tool)
	fmt.Println("Tool registered")

	// Output: Tool registered
}

func ExampleLoadConfig() {
	// Load configuration
	config, _ := ai.LoadConfig()

	fmt.Printf("LLM Provider: %s\n", config.LLMProvider)
	fmt.Printf("Max Iterations: %d\n", config.MaxIterations)

	// Output:
	// LLM Provider: openai
	// Max Iterations: 10
}
