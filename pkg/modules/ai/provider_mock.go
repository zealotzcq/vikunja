package ai

import (
	"context"
	"fmt"
	"strings"
)

// MockLLMProvider implements LLMProvider for testing and development
type MockLLMProvider struct{}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{}
}

func (p *MockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	lowerPrompt := strings.ToLower(prompt)

	if strings.Contains(lowerPrompt, "help") || strings.Contains(lowerPrompt, "帮助") {
		return `我是您的 AI 助手，可以帮助您：
• 导航到不同页面（项目、任务、团队、标签）
• 创建和管理任务
• 搜索内容
• 查看收藏

您想做什么呢？`, nil
	}

	if strings.Contains(lowerPrompt, "project") || strings.Contains(lowerPrompt, "项目") {
		if strings.Contains(lowerPrompt, "123") {
			return `TOOL: navigate
INPUT: {"route_name": "project.index", "params": {"projectId": 123}}`, nil
		}
		return `TOOL: navigate
INPUT: {"route_name": "projects.index", "params": {}}`, nil
	}

	if strings.Contains(lowerPrompt, "task") || strings.Contains(lowerPrompt, "任务") {
		if strings.Contains(lowerPrompt, "create") || strings.Contains(lowerPrompt, "创建") {
			return `我可以帮您创建任务。请告诉我：
1. 任务的标题是什么？
2. 您希望添加到哪个项目？
3. 有什么描述吗？`, nil
		}
		if strings.Contains(lowerPrompt, "456") {
			return `TOOL: navigate
INPUT: {"route_name": "task.detail", "params": {"id": 456}}`, nil
		}
		return `TOOL: navigate
INPUT: {"route_name": "tasks.range", "params": {}}`, nil
	}

	if strings.Contains(lowerPrompt, "team") || strings.Contains(lowerPrompt, "团队") {
		if strings.Contains(lowerPrompt, "789") {
			return `TOOL: navigate
INPUT: {"route_name": "teams.edit", "params": {"id": 789}}`, nil
		}
		return `TOOL: navigate
INPUT: {"route_name": "teams.index", "params": {}}`, nil
	}

	if strings.Contains(lowerPrompt, "label") || strings.Contains(lowerPrompt, "标签") {
		return `TOOL: navigate
INPUT: {"route_name": "labels.index", "params": {}}`, nil
	}

	if strings.Contains(lowerPrompt, "favorite") || strings.Contains(lowerPrompt, "收藏") {
		return `TOOL: navigate
INPUT: {"route_name": "project.index", "params": {"projectId": -1}}`, nil
	}

	if strings.Contains(lowerPrompt, "home") || strings.Contains(lowerPrompt, "主页") ||
		strings.Contains(lowerPrompt, "首页") || strings.Contains(lowerPrompt, "概览") {
		return `TOOL: navigate
INPUT: {"route_name": "home", "params": {}}`, nil
	}

	if strings.Contains(lowerPrompt, "upcoming") || strings.Contains(lowerPrompt, "即将到来") ||
		strings.Contains(lowerPrompt, "即将进行") {
		return `TOOL: navigate
INPUT: {"route_name": "tasks.range", "params": {"showNulls": true}}`, nil
	}

	if strings.Contains(lowerPrompt, "search") || strings.Contains(lowerPrompt, "搜索") {
		query := ""
		if idx := strings.Index(lowerPrompt, "search "); idx >= 0 {
			query = strings.TrimSpace(lowerPrompt[idx+7:])
		} else if idx := strings.Index(lowerPrompt, "搜索 "); idx >= 0 {
			query = strings.TrimSpace(lowerPrompt[idx+3:])
		}

		if query == "" {
			query = "example query"
		}

		return fmt.Sprintf(`TOOL: search
INPUT: {"query": "%s", "type": "tasks"}`, query), nil
	}

	return `我理解您的请求。让我帮您处理这个问题。请提供更多详细信息，这样我可以更好地帮助您。`, nil
}

func (p *MockLLMProvider) GenerateWithTools(ctx context.Context, prompt string, tools []map[string]interface{}) (string, error) {
	return p.Generate(ctx, prompt)
}
