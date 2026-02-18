package ai

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"code.vikunja.io/api/pkg/i18n"
	"code.vikunja.io/api/pkg/modules/chat_session"
)

// GenerateResponse generates a mock AI response based on user input and current page context
// This is legacy implementation, kept for backward compatibility
func GenerateResponse(userContent string, routeName string, routeParams map[string]interface{}) (string, map[string]interface{}, bool) {
	lowerContent := strings.ToLower(userContent)

	// Check for navigation commands first
	if navInfo := extractNavigationCommand(lowerContent, routeName); navInfo != nil {
		return fmt.Sprintf(i18n.T("en", "ai.navigation.navigating_to"), navInfo.Message), map[string]interface{}{
			"route_name": navInfo.RouteName,
			"params":     navInfo.Params,
		}, true
	}

	// Default help message
	return generateHelpMessage(), nil, false
}

// GenerateAgentResponse generates an AI response using the agent system
func GenerateAgentResponse(ctx context.Context, userID int64, userContent string, routeName string, routeParams map[string]interface{}) (*AgentResponse, error) {
	// GenerateAgentResponse is deprecated, use SendMessage endpoint from chat.go instead
	// This function is kept for backward compatibility with tests
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("[AI] GenerateAgentResponse - UserID: %d, Provider: %s, Model: %s", userID, config.LLMProvider, config.OpenAIModel)

	agent, err := GetAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	session, err := chat_session.GetDefault().GetOrCreateSession(userID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	log.Printf("[AI] Session loaded - Messages count: %d", len(session.Messages))

	agentCtx := &AgentContext{
		UserID:           userID,
		CurrentRoute:     routeName,
		RouteParams:      routeParams,
		SessionData:      make(map[string]interface{}),
		MessageHistory:   make([]Message, 0, len(session.Messages)),
		SubordinateStaff: session.SubordinateStaff,
	}

	for _, msg := range session.Messages {
		agentCtx.MessageHistory = append(agentCtx.MessageHistory, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	log.Printf("[AI] AgentContext created - History messages: %d", len(agentCtx.MessageHistory))

	// Note: userContent is deprecated and ignored, all messages should be in session
	multiResponse, err := agent.ProcessMessage(ctx, agentCtx)
	if err != nil {
		return nil, err
	}

	// Return first response for backward compatibility
	if len(multiResponse.Responses) > 0 {
		return multiResponse.Responses[0], nil
	}

	return nil, fmt.Errorf("no response from agent")
}

// NavigationInfo contains navigation command details
type NavigationInfo struct {
	Message   string
	RouteName string
	Params    map[string]interface{}
}

// extractNavigationCommand extracts navigation commands from user message
func extractNavigationCommand(userContent, currentRoute string) *NavigationInfo {
	// Extract numbers from message
	numberMatch := regexp.MustCompile(`\d+`).FindString(userContent)
	projectNumber := parseInt(numberMatch)
	taskNumber := parseInt(numberMatch)
	teamNumber := parseInt(numberMatch)

	// Project navigation
	if (strings.Contains(userContent, "project") || strings.Contains(userContent, "项目")) && projectNumber > 0 {
		return &NavigationInfo{
			Message:   fmt.Sprintf(i18n.T("en", "ai.navigation.to_project_number"), projectNumber),
			RouteName: "project.index",
			Params:    map[string]interface{}{"projectId": projectNumber},
		}
	}

	if strings.Contains(userContent, "project") || strings.Contains(userContent, "项目") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_project_list"),
			RouteName: "projects.index",
			Params:    nil,
		}
	}

	// Task navigation
	if (strings.Contains(userContent, "task") || strings.Contains(userContent, "任务")) && taskNumber > 0 {
		return &NavigationInfo{
			Message:   fmt.Sprintf(i18n.T("en", "ai.navigation.to_task_number"), taskNumber),
			RouteName: "task.detail",
			Params:    map[string]interface{}{"id": taskNumber},
		}
	}

	if strings.Contains(userContent, "task") || strings.Contains(userContent, "任务") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_task_list"),
			RouteName: "tasks.range",
			Params:    nil,
		}
	}

	// Team navigation
	if (strings.Contains(userContent, "team") || strings.Contains(userContent, "团队")) && teamNumber > 0 {
		return &NavigationInfo{
			Message:   fmt.Sprintf(i18n.T("en", "ai.navigation.to_team_number"), teamNumber),
			RouteName: "teams.edit",
			Params:    map[string]interface{}{"id": teamNumber},
		}
	}

	if strings.Contains(userContent, "team") || strings.Contains(userContent, "团队") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_team_list"),
			RouteName: "teams.index",
			Params:    nil,
		}
	}

	// Label navigation
	if strings.Contains(userContent, "label") || strings.Contains(userContent, "标签") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_labels_list"),
			RouteName: "labels.index",
			Params:    nil,
		}
	}

	// Favorites navigation
	if strings.Contains(userContent, "favorite") || strings.Contains(userContent, "收藏") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_favorites"),
			RouteName: "project.index",
			Params:    map[string]interface{}{"projectId": -1},
		}
	}

	// Home/Homepage navigation
	if strings.Contains(userContent, "home") || strings.Contains(userContent, "主页") ||
		strings.Contains(userContent, "首页") || strings.Contains(userContent, "概览") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_home"),
			RouteName: "home",
			Params:    nil,
		}
	}

	// Upcoming tasks navigation
	if strings.Contains(userContent, "upcoming") || strings.Contains(userContent, "即将到来") ||
		strings.Contains(userContent, "即将进行") {
		return &NavigationInfo{
			Message:   i18n.T("en", "ai.navigation.to_upcoming"),
			RouteName: "tasks.range",
			Params:    map[string]interface{}{"showNulls": true},
		}
	}

	return nil
}

// generateHelpMessage generates the default help message
func generateHelpMessage() string {
	var sb strings.Builder

	sb.WriteString(i18n.T("en", "ai.help.greeting"))
	sb.WriteString("\n\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.view_projects"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.view_tasks"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.view_teams"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.view_labels"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.view_favorites"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.go_home"))
	sb.WriteString("\n\n")

	sb.WriteString("You can also directly enter:\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.example_project"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.example_task"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.example_team"))
	sb.WriteString("\n")

	sb.WriteString("• ")
	sb.WriteString(i18n.T("en", "ai.help.example_upcoming"))
	sb.WriteString("\n\n")

	sb.WriteString(i18n.T("en", "ai.help.closing"))

	return sb.String()
}

// parseInt safely parses a string to int
func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
