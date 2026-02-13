package v1

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/ai"
	"code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/labstack/echo/v5"
)

// Allowed usernames for chat assistant
var allowedChatUsernames = map[string]bool{
	"leader": true,
	"王大牛":    true,
}

// isUserAllowedForChat checks if user is allowed to use the chat assistant
func isUserAllowedForChat(a web.Auth) bool {
	userObj, isUser := a.(*user.User)
	if !isUser {
		return false
	}
	return allowedChatUsernames[userObj.Username]
}

// SendMessageRequest represents a request to send a chat message
type SendMessageRequest struct {
	MessageID string    `json:"message_id"`
	Message   string    `json:"message" validate:"required"`
	PageInfo  *PageInfo `json:"page_info"`
	UseAgent  bool      `json:"use_agent"` // Use the new agent system instead of mock
}

// PageInfo represents current page context
type PageInfo struct {
	RouteName string                 `json:"route_name" validate:"required"`
	Params    map[string]interface{} `json:"params"`
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	ID                string             `json:"id"`
	Role              string             `json:"role"` // "user" | "assistant"
	Content           string             `json:"content"`
	Timestamp         int64              `json:"timestamp"`
	NavigationCommand *NavigationCommand `json:"navigationCommand,omitempty"`
}

// NavigationCommand represents a navigation action
type NavigationCommand struct {
	RouteName string                 `json:"routeName"`
	Params    map[string]interface{} `json:"params"`
	Label     string                 `json:"label"`
}

// SendMessage handles sending a message to the chat assistant
func SendMessage(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	if !isUserAllowedForChat(a) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	userID := a.GetID()

	// Bind request body
	req := new(SendMessageRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	userMsgID := req.MessageID
	if userMsgID == "" {
		userMsgID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}

	userMessage := chat_session.Message{
		ID:        userMsgID,
		Type:      "user_input",
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now().Unix(),
	}

	log.Printf("[Chat] Adding user message for user %d: id=%s, type=%s", userID, userMsgID, userMessage.Type)

	if err := chat_session.GetDefault().AddMessage(userID, userMessage); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to save message: %v", err))
	}

	go processUserMessageAsync(context.Background(), userID, userMsgID, req)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        userMsgID,
		"status":    "processing",
		"timestamp": time.Now().Unix(),
	})
}

// GetSession retrieves the current user's chat session
func GetSession(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	if !isUserAllowedForChat(a) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	userID := a.GetID()

	// Get or create session
	sessionData, err := chat_session.GetDefault().GetOrCreateSession(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get session: %v", err))
	}

	return c.JSON(http.StatusOK, sessionData)
}

// ClearSession clears the current user's chat session
func ClearSession(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	if !isUserAllowedForChat(a) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	userID := a.GetID()

	// Clear the session
	if err := chat_session.GetDefault().ClearSession(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to clear session: %v", err))
	}

	return c.JSON(http.StatusOK, models.Message{Message: "Session cleared successfully"})
}

// GetChatHistory retrieves the current user's chat history (filtered for frontend)
func GetChatHistory(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	if !isUserAllowedForChat(a) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	userID := a.GetID()

	sessionData, err := chat_session.GetDefault().GetOrCreateSession(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get session: %v", err))
	}

	frontendMessages := []ChatMessage{}
	for _, msg := range sessionData.Messages {
		if msg.Type == "user_input" || msg.Type == "assistant_response" {
			var navigationCommand *NavigationCommand
			if msg.NavigationCommand != nil {
				navigationCommand = &NavigationCommand{
					RouteName: msg.NavigationCommand.RouteName,
					Params:    msg.NavigationCommand.Params,
					Label:     msg.NavigationCommand.Label,
				}
			}
			frontendMessages = append(frontendMessages, ChatMessage{
				ID:                msg.ID,
				Role:              msg.Role,
				Content:           msg.Content,
				Timestamp:         msg.Timestamp,
				NavigationCommand: navigationCommand,
			})
		}
	}

	response := map[string]interface{}{
		"id":         sessionData.ID,
		"user_id":    sessionData.UserID,
		"created_at": sessionData.CreatedAt.Unix(),
		"messages":   frontendMessages,
		"expires_at": sessionData.ExpiresAt.Unix(),
	}

	return c.JSON(http.StatusOK, response)
}

// processUserMessageAsync processes a user message asynchronously
func processUserMessageAsync(ctx context.Context, userID int64, userMsgID string, req *SendMessageRequest) {
	log.Printf("[Chat] Processing message async for user %d: id=%s", userID, userMsgID)

	routeName := ""
	var routeParams map[string]interface{}
	if req.PageInfo != nil {
		routeName = req.PageInfo.RouteName
		routeParams = req.PageInfo.Params
	}

	agent, err := ai.GetAgent()
	if err != nil {
		log.Printf("[Chat] Failed to get agent: %v", err)
		return
	}

	u, err := user.GetUserByID(db.NewSession(), userID)
	if err != nil {
		log.Printf("[Chat] Failed to get user: %v", err)
		return
	}

	agentCtx := &ai.AgentContext{
		UserID:         userID,
		CurrentRoute:   routeName,
		RouteParams:    routeParams,
		SessionData:    make(map[string]interface{}),
		MessageHistory: []ai.Message{},
		Language:       u.Language,
	}

	session, err := chat_session.GetDefault().GetOrCreateSession(userID)
	if err != nil {
		log.Printf("[Chat] Failed to get session: %v", err)
		return
	}

	for _, msg := range session.Messages {
		// Skip the current user message we're processing (it will be added separately)
		if msg.ID == userMsgID {
			continue
		}

		switch msg.Type {
		case "user_input":
			// User input message
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})

		case "assistant_response":
			// Final assistant response to user
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})

		case "tool_call":
			// Tool call from assistant
			// Convert to proper OpenAI tool_calls format
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
				ToolCalls: []ai.ToolCallInfo{
					{
						ID:        msg.ID,
						Type:      "function",
						Name:      msg.ToolName,
						Arguments: msg.ToolInput,
					},
				},
			})

		case "tool_result":
			// Result from tool execution
			// Format as tool message with tool output
			// ToolCallID should reference the corresponding tool call
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:       msg.Role,
				Content:    msg.ToolOutput,
				ToolCallID: msg.ToolCallID,
			})
		}
	}

	agentResponse, err := agent.ProcessMessage(ctx, agentCtx, req.Message)
	if err != nil {
		log.Printf("[Chat] Agent error: %v", err)
		return
	}

	assistantMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	var navigationCommand *chat_session.NavigationCommand
	if agentResponse.ShouldNavigate && agentResponse.NavigationInfo != nil {
		navigationCommand = &chat_session.NavigationCommand{
			RouteName: agentResponse.NavigationInfo.RouteName,
			Params:    agentResponse.NavigationInfo.Params,
			Label:     agentResponse.Content,
		}
	}

	metadata := make(map[string]interface{})
	if len(agentResponse.ExecutionSteps) > 0 {
		metadata["execution_steps"] = agentResponse.ExecutionSteps
	}
	if agentResponse.TokensUsed > 0 {
		metadata["tokens_used"] = agentResponse.TokensUsed
	}

	// Save tool calls and tool results to session
	for _, step := range agentResponse.ExecutionSteps {
		// Parse tool call from the thought (thought contains the tool call info)
		// Step.Thought format: "{content}\nTOOL: tool_name\nINPUT: {json_params}"
		toolCallID := fmt.Sprintf("call_%d", time.Now().UnixNano())
		var toolName string
		var toolInput string
		var contentLines []string

		lines := strings.Split(step.Thought, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "TOOL:") {
				toolName = strings.TrimSpace(strings.TrimPrefix(line, "TOOL:"))
			} else if strings.HasPrefix(line, "INPUT:") {
				toolInput = strings.TrimSpace(strings.TrimPrefix(line, "INPUT:"))
			} else if line != "" {
				contentLines = append(contentLines, line)
			}
		}

		// Save tool call message
		toolCallMsg := chat_session.Message{
			ID:        toolCallID,
			Type:      "tool_call",
			Role:      "assistant",
			Content:   strings.Join(contentLines, "\n"),
			ToolName:  toolName,
			ToolInput: toolInput,
			Timestamp: time.Now().Unix(),
		}
		log.Printf("[Chat] Saving tool call for user %d: tool=%s, content=%s", userID, toolName, toolCallMsg.Content)
		if err := chat_session.GetDefault().AddMessage(userID, toolCallMsg); err != nil {
			log.Printf("[Chat] Failed to save tool call message: %v", err)
		}

		// Save tool result message with ToolCallID referencing the tool call
		toolResultMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
		toolResultMsg := chat_session.Message{
			ID:         toolResultMsgID,
			Type:       "tool_result",
			Role:       "tool",
			ToolName:   toolName,
			ToolOutput: step.Output,
			ToolCallID: toolCallID,
			Timestamp:  time.Now().Unix(),
		}
		log.Printf("[Chat] Saving tool result for user %d: tool=%s", userID, toolName)
		if err := chat_session.GetDefault().AddMessage(userID, toolResultMsg); err != nil {
			log.Printf("[Chat] Failed to save tool result message: %v", err)
		}
	}

	assistantMessage := chat_session.Message{
		ID:                assistantMsgID,
		Type:              "assistant_response",
		Role:              "assistant",
		Content:           agentResponse.Content,
		Timestamp:         time.Now().Unix(),
		NavigationCommand: navigationCommand,
		Metadata:          metadata,
	}

	log.Printf("[Chat] Adding assistant message for user %d: id=%s, type=%s", userID, assistantMsgID, assistantMessage.Type)
	if err := chat_session.GetDefault().AddMessage(userID, assistantMessage); err != nil {
		log.Printf("[Chat] Failed to save assistant message: %v", err)
	}
}
