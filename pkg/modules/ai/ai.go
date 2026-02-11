package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// GenerateResponse generates a mock AI response based on user input and current page context
func GenerateResponse(userContent string, routeName string, routeParams map[string]interface{}) (string, map[string]interface{}, bool) {
	lowerContent := strings.ToLower(userContent)

	// Check for navigation commands first
	if navInfo := extractNavigationCommand(lowerContent, routeName); navInfo != nil {
		return fmt.Sprintf("正在%s", navInfo.Message), map[string]interface{}{
			"route_name": navInfo.RouteName,
			"params":     navInfo.Params,
		}, true
	}

	// Default help message
	return generateHelpMessage(), nil, false
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
			Message:   fmt.Sprintf("跳转到项目 %d", projectNumber),
			RouteName: "project.index",
			Params:    map[string]interface{}{"projectId": projectNumber},
		}
	}

	if strings.Contains(userContent, "project") || strings.Contains(userContent, "项目") {
		return &NavigationInfo{
			Message:   "跳转到项目列表",
			RouteName: "projects.index",
			Params:    nil,
		}
	}

	// Task navigation
	if (strings.Contains(userContent, "task") || strings.Contains(userContent, "任务")) && taskNumber > 0 {
		return &NavigationInfo{
			Message:   fmt.Sprintf("跳转到任务 %d", taskNumber),
			RouteName: "task.detail",
			Params:    map[string]interface{}{"id": taskNumber},
		}
	}

	if strings.Contains(userContent, "task") || strings.Contains(userContent, "任务") {
		return &NavigationInfo{
			Message:   "跳转到任务列表",
			RouteName: "tasks.range",
			Params:    nil,
		}
	}

	// Team navigation
	if (strings.Contains(userContent, "team") || strings.Contains(userContent, "团队")) && teamNumber > 0 {
		return &NavigationInfo{
			Message:   fmt.Sprintf("跳转到团队 %d", teamNumber),
			RouteName: "teams.edit",
			Params:    map[string]interface{}{"id": teamNumber},
		}
	}

	if strings.Contains(userContent, "team") || strings.Contains(userContent, "团队") {
		return &NavigationInfo{
			Message:   "跳转到团队列表",
			RouteName: "teams.index",
			Params:    nil,
		}
	}

	// Label navigation
	if strings.Contains(userContent, "label") || strings.Contains(userContent, "标签") {
		return &NavigationInfo{
			Message:   "跳转到标签列表",
			RouteName: "labels.index",
			Params:    nil,
		}
	}

	// Favorites navigation
	if strings.Contains(userContent, "favorite") || strings.Contains(userContent, "收藏") {
		return &NavigationInfo{
			Message:   "查看收藏的任务",
			RouteName: "project.index",
			Params:    map[string]interface{}{"projectId": -1},
		}
	}

	// Home/Homepage navigation
	if strings.Contains(userContent, "home") || strings.Contains(userContent, "主页") ||
		strings.Contains(userContent, "首页") || strings.Contains(userContent, "概览") {
		return &NavigationInfo{
			Message:   "返回首页",
			RouteName: "home",
			Params:    nil,
		}
	}

	// Upcoming tasks navigation
	if strings.Contains(userContent, "upcoming") || strings.Contains(userContent, "即将到来") ||
		strings.Contains(userContent, "即将进行") {
		return &NavigationInfo{
			Message:   "查看即将到来的任务（包含无日期任务）",
			RouteName: "tasks.range",
			Params:    map[string]interface{}{"showNulls": true},
		}
	}

	return nil
}

// generateHelpMessage generates the default help message
func generateHelpMessage() string {
	return `您好！我是您的 AI 助手。我可以帮您：

• 查看项目
• 查看任务
• 查看团队
• 查看标签
• 查看收藏
• 返回首页

您也可以直接输入：
• "项目 123" 跳转到特定项目
• "任务 456" 跳转到特定任务
• "团队 789" 跳转到特定团队
• "即将到来" 查看包含无日期的任务

您可以告诉我您想做什么，我会尽力帮助您。`
}

// parseInt safely parses a string to int
func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
