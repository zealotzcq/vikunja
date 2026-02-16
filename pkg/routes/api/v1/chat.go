package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/company"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/ai"
	"code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/labstack/echo/v5"
)

// isUserAllowedForChat checks if user is allowed to use the chat assistant
func isUserAllowedForChat(a web.Auth, companyID int64) bool {
	userObj, isUser := a.(*user.User)
	if !isUser {
		return false
	}

	s := db.NewSession()
	defer s.Close()

	role := company.GetUserRole(s, userObj.ID, companyID)
	if role == "" {
		return false
	}

	return role == "creator" || role == "admin"
}

// SendMessageRequest represents a request to send a chat message
type SendMessageRequest struct {
	MessageID string    `json:"message_id"`
	Message   string    `json:"message" validate:"required"`
	PageInfo  *PageInfo `json:"page_info"`
	UseAgent  bool      `json:"use_agent"` // Use the new agent system instead of mock
	CompanyID int64     `json:"company_id"`
}

// SubmitQuestionAnswerRequest represents a request to submit a question answer
type SubmitQuestionAnswerRequest struct {
	Answer    string `json:"answer" validate:"required"`
	CompanyID int64  `json:"company_id" validate:"required"`
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
	ButtonNavigation  *ButtonNavigation  `json:"buttonNavigation,omitempty"`
	QuestionData      string             `json:"questionData,omitempty"`
	ToolCallID        string             `json:"toolCallID,omitempty"`
}

// NavigationCommand represents a navigation action
type NavigationCommand struct {
	RouteName string                 `json:"routeName"`
	Params    map[string]interface{} `json:"params"`
	Label     string                 `json:"label"`
}

// ButtonNavigation represents a button-based navigation action
type ButtonNavigation struct {
	RouteName string                 `json:"routeName"`
	Params    map[string]interface{} `json:"params"`
	Label     string                 `json:"label"`
	Title     string                 `json:"title,omitempty"`
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

	userID := a.GetID()

	// Bind request body
	req := new(SendMessageRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	if !isUserAllowedForChat(a, req.CompanyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
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
		CompanyID: req.CompanyID,
	}

	if err := chat_session.GetDefault().AddMessage(userID, req.CompanyID, userMessage); err != nil {
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

	userID := a.GetID()

	companyIDStr := c.QueryParam("company_id")
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil || companyID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "company_id is required and must be a positive integer")
	}

	if !isUserAllowedForChat(a, companyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	// Get or create session
	sessionData, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
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

	userID := a.GetID()

	companyIDStr := c.QueryParam("company_id")
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil || companyID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "company_id is required and must be a positive integer")
	}

	if !isUserAllowedForChat(a, companyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	// Clear the session
	if err := chat_session.GetDefault().ClearSession(userID, companyID); err != nil {
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

	userID := a.GetID()

	companyIDStr := c.QueryParam("company_id")
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil || companyID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "company_id is required and must be a positive integer")
	}

	if !isUserAllowedForChat(a, companyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	sessionData, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get session: %v", err))
	}

	frontendMessages := []ChatMessage{}
	for _, msg := range sessionData.Messages {
		if msg.Type == "user_input" || msg.Type == "assistant_response" || msg.Type == "question" || msg.Type == "button_navigation" {
			var navigationCommand *NavigationCommand
			if msg.NavigationCommand != nil {
				navigationCommand = &NavigationCommand{
					RouteName: msg.NavigationCommand.RouteName,
					Params:    msg.NavigationCommand.Params,
					Label:     msg.NavigationCommand.Label,
				}
			}

			var buttonNavigation *ButtonNavigation
			if msg.ButtonNavigation != nil {
				buttonNavigation = &ButtonNavigation{
					RouteName: msg.ButtonNavigation.RouteName,
					Params:    msg.ButtonNavigation.Params,
					Label:     msg.ButtonNavigation.Label,
					Title:     msg.ButtonNavigation.Title,
				}
			}

			frontendMessages = append(frontendMessages, ChatMessage{
				ID:                msg.ID,
				Role:              msg.Role,
				Content:           msg.Content,
				Timestamp:         msg.Timestamp,
				NavigationCommand: navigationCommand,
				ButtonNavigation:  buttonNavigation,
				QuestionData:      msg.QuestionData,
				ToolCallID:        msg.ToolCallID,
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
	routeName := ""
	var routeParams map[string]interface{}
	if req.PageInfo != nil {
		routeName = req.PageInfo.RouteName
		routeParams = req.PageInfo.Params
	}

	agent, err := ai.GetAgent()
	if err != nil {
		return
	}

	u, err := user.GetUserByID(db.NewSession(), userID)
	if err != nil {
		return
	}

	agentCtx := &ai.AgentContext{
		UserID:         userID,
		CompanyID:      req.CompanyID,
		CurrentRoute:   routeName,
		RouteParams:    routeParams,
		SessionData:    make(map[string]interface{}),
		MessageHistory: []ai.Message{},
		Language:       u.Language,
	}

	session, err := chat_session.GetDefault().GetOrCreateSession(userID, req.CompanyID)
	if err != nil {
		return
	}

	agentCtx.SubordinateStaff = session.SubordinateStaff

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
			// Skip if tool name is empty (old/corrupted data)
			if msg.ToolName == "" {
				continue
			}

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

		case "question_answer":
			// Skip question_answer messages
			// These are saved to the session but not used in the agent's message history
			// They represent answers to questions and are already handled through the question/answer flow
			continue
		}
	}

	agentResponse, err := agent.ProcessMessage(ctx, agentCtx, req.Message)
	if err != nil {
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

	hasQuestion := false
	var questionData string

	for _, step := range agentResponse.ExecutionSteps {
		if step.Action == "question" {
			hasQuestion = true

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(step.Input), &result); err == nil {
				if questions, ok := result["questions"]; ok {
					questionsJSON, _ := json.Marshal(questions)
					questionData = string(questionsJSON)
				}
			}
		}
	}

	// Save tool calls and tool results to session (including question tool calls)
	questionToolCallID := ""
	for _, step := range agentResponse.ExecutionSteps {
		// Use step.Action and step.Input directly (new structured format)
		toolCallID := fmt.Sprintf("call_%d", time.Now().UnixNano())

		// If this is a question tool call, save the ID for later use
		if step.Action == "question" {
			questionToolCallID = toolCallID
		}

		// Save tool call message
		toolCallMsg := chat_session.Message{
			ID:        toolCallID,
			Type:      "tool_call",
			Role:      "assistant",
			Content:   step.Thought,
			ToolName:  step.Action,
			ToolInput: step.Input,
			Timestamp: time.Now().Unix(),
			CompanyID: req.CompanyID,
		}
		if err := chat_session.GetDefault().AddMessage(userID, req.CompanyID, toolCallMsg); err != nil {
		}

		// Save tool result message with ToolCallID referencing tool call
		toolResultMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
		toolResultMsg := chat_session.Message{
			ID:         toolResultMsgID,
			Type:       "tool_result",
			Role:       "tool",
			ToolName:   step.Action,
			ToolOutput: step.Output,
			ToolCallID: toolCallID,
			Timestamp:  time.Now().Unix(),
			CompanyID:  req.CompanyID,
		}
		if err := chat_session.GetDefault().AddMessage(userID, req.CompanyID, toolResultMsg); err != nil {
		}
	}

	// Save question message if there was a question tool call
	if hasQuestion && questionData != "" {
		questionMessage := chat_session.Message{
			ID:           fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Type:         "question",
			Role:         "assistant",
			Content:      "",
			QuestionData: questionData,
			Timestamp:    time.Now().Unix(),
			CompanyID:    req.CompanyID,
			ToolCallID:   questionToolCallID,
		}
		chat_session.GetDefault().AddMessage(userID, req.CompanyID, questionMessage)
		return
	}

	// Save assistant response message

	var buttonNavigation *chat_session.ButtonNavigation
	if agentResponse.ButtonNavigation != nil {
		buttonNavigation = &chat_session.ButtonNavigation{
			RouteName: agentResponse.ButtonNavigation.RouteName,
			Params:    agentResponse.ButtonNavigation.Params,
			Label:     agentResponse.ButtonNavigation.Label,
			Title:     agentResponse.ButtonNavigation.Title,
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
		CompanyID:         req.CompanyID,
	}

	chat_session.GetDefault().AddMessage(userID, req.CompanyID, assistantMessage)

	// Save button navigation as separate message if exists
	if buttonNavigation != nil {
		buttonNavMsg := chat_session.Message{
			ID:               fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Type:             "button_navigation",
			Role:             "assistant",
			Content:          "",
			Timestamp:        time.Now().Unix(),
			ButtonNavigation: buttonNavigation,
			CompanyID:        req.CompanyID,
		}
		chat_session.GetDefault().AddMessage(userID, req.CompanyID, buttonNavMsg)
	}
}

// SubmitQuestionAnswer handles submitting an answer to a question
func SubmitQuestionAnswer(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	userID := a.GetID()

	req := new(SubmitQuestionAnswerRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	if !isUserAllowedForChat(a, req.CompanyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	answerMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	answerContent := fmt.Sprintf("[回答问题]%s", req.Answer)
	answerMessage := chat_session.Message{
		ID:        answerMsgID,
		Type:      "question_answer",
		Role:      "user",
		Content:   answerContent,
		Timestamp: time.Now().Unix(),
		CompanyID: req.CompanyID,
	}

	if err := chat_session.GetDefault().AddMessage(userID, req.CompanyID, answerMessage); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to save answer: %v", err))
	}

	go processQuestionAnswerAsync(context.Background(), userID, req.CompanyID, answerContent)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        answerMsgID,
		"status":    "processing",
		"timestamp": time.Now().Unix(),
	})
}

// processQuestionAnswerAsync processes a question answer asynchronously
func processQuestionAnswerAsync(ctx context.Context, userID, companyID int64, answer string) {
	session, err := chat_session.GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		return
	}

	agent, err := ai.GetAgent()
	if err != nil {
		return
	}

	u, err := user.GetUserByID(db.NewSession(), userID)
	if err != nil {
		return
	}

	agentCtx := &ai.AgentContext{
		UserID:         userID,
		CompanyID:      companyID,
		SessionData:    make(map[string]interface{}),
		MessageHistory: []ai.Message{},
		Language:       u.Language,
	}

	agentCtx.SubordinateStaff = session.SubordinateStaff

	for _, msg := range session.Messages {
		switch msg.Type {
		case "user_input":
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		case "assistant_response":
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		case "tool_call":
			if msg.ToolName == "" {
				continue
			}
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
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:       msg.Role,
				Content:    msg.ToolOutput,
				ToolCallID: msg.ToolCallID,
			})
		case "question_answer":
			// Skip question_answer messages - they are already passed as the 'answer' parameter to ProcessMessage
			// Including them here would cause the same message to be added twice to the LLM's messages array
			continue
		}
	}

	agentResponse, err := agent.ProcessMessage(ctx, agentCtx, answer)
	if err != nil {
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

	hasQuestion := false
	var questionData string

	for _, step := range agentResponse.ExecutionSteps {
		if step.Action == "question" {
			hasQuestion = true

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(step.Input), &result); err == nil {
				if questions, ok := result["questions"]; ok {
					questionsJSON, _ := json.Marshal(questions)
					questionData = string(questionsJSON)
				}
			}
		}
	}

	questionToolCallID := ""
	for _, step := range agentResponse.ExecutionSteps {
		toolCallID := fmt.Sprintf("call_%d", time.Now().UnixNano())

		// If this is a question tool call, save the ID for later use
		if step.Action == "question" {
			questionToolCallID = toolCallID
		}

		toolCallMsg := chat_session.Message{
			ID:        toolCallID,
			Type:      "tool_call",
			Role:      "assistant",
			Content:   step.Thought,
			ToolName:  step.Action,
			ToolInput: step.Input,
			Timestamp: time.Now().Unix(),
			CompanyID: companyID,
		}
		if err := chat_session.GetDefault().AddMessage(userID, companyID, toolCallMsg); err != nil {
		}

		toolResultMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
		toolResultMsg := chat_session.Message{
			ID:         toolResultMsgID,
			Type:       "tool_result",
			Role:       "tool",
			ToolName:   step.Action,
			ToolOutput: step.Output,
			ToolCallID: toolCallID,
			Timestamp:  time.Now().Unix(),
			CompanyID:  companyID,
		}
		if err := chat_session.GetDefault().AddMessage(userID, companyID, toolResultMsg); err != nil {
		}
	}

	// Save question message if there was a question tool call
	if hasQuestion && questionData != "" {
		questionMessage := chat_session.Message{
			ID:           fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Type:         "question",
			Role:         "assistant",
			Content:      "",
			QuestionData: questionData,
			Timestamp:    time.Now().Unix(),
			CompanyID:    companyID,
			ToolCallID:   questionToolCallID,
		}
		chat_session.GetDefault().AddMessage(userID, companyID, questionMessage)
		return
	}

	assistantMessage := chat_session.Message{
		ID:                assistantMsgID,
		Type:              "assistant_response",
		Role:              "assistant",
		Content:           agentResponse.Content,
		Timestamp:         time.Now().Unix(),
		NavigationCommand: navigationCommand,
		Metadata:          metadata,
		CompanyID:         companyID,
	}

	chat_session.GetDefault().AddMessage(userID, companyID, assistantMessage)

	// Save button navigation as separate message if exists
	if agentResponse.ButtonNavigation != nil {
		buttonNavMsg := chat_session.Message{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Type:      "button_navigation",
			Role:      "assistant",
			Content:   "",
			Timestamp: time.Now().Unix(),
			ButtonNavigation: &chat_session.ButtonNavigation{
				RouteName: agentResponse.ButtonNavigation.RouteName,
				Params:    agentResponse.ButtonNavigation.Params,
				Label:     agentResponse.ButtonNavigation.Label,
			},
			CompanyID: companyID,
		}
		chat_session.GetDefault().AddMessage(userID, companyID, buttonNavMsg)
	}
}
