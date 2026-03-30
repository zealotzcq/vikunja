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

	allowed := role == "creator" || role == "admin"
	return allowed
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

// SetCurrentTaskRequest represents a request to set the current task
type SetCurrentTaskRequest struct {
	TaskID    int64  `json:"task_id" validate:"required"`
	Title     string `json:"title" validate:"required"`
	ProjectID int64  `json:"project_id" validate:"required"`
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

	chat_session.GetDefault().PushPendingMessage(userID, req.CompanyID, userMessage)

	go processPendingMessagesAsync(context.Background(), userID, req.CompanyID, req)

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

// CheckNewMessages checks if there are new messages in the session
func CheckNewMessages(c *echo.Context) error {
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

	lastMessageId := c.QueryParam("last_message_id")

	var hasNew bool
	currentLastMessageId := ""

	if len(sessionData.Messages) > 0 {
		latestMsg := sessionData.Messages[len(sessionData.Messages)-1]
		currentLastMessageId = latestMsg.ID

		if lastMessageId == "" {
			hasNew = true
		} else {
			hasNew = latestMsg.ID != lastMessageId
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"has_new":         hasNew,
		"last_message_id": currentLastMessageId,
	})
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
		if msg.Type == "user_input" || msg.Type == "assistant_response" || msg.Type == "question" || msg.Type == "button_navigation" || msg.Type == "question_answer" {
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

	var currentTask map[string]interface{}
	if sessionData.CurrentTask != nil {
		currentTask = map[string]interface{}{
			"task_id":    sessionData.CurrentTask.TaskID,
			"title":      sessionData.CurrentTask.Title,
			"project_id": sessionData.CurrentTask.ProjectID,
		}
	}

	response := map[string]interface{}{
		"id":           sessionData.ID,
		"user_id":      sessionData.UserID,
		"created_at":   sessionData.CreatedAt.Unix(),
		"messages":     frontendMessages,
		"expires_at":   sessionData.ExpiresAt.Unix(),
		"current_task": currentTask,
	}

	return c.JSON(http.StatusOK, response)
}

// processPendingMessagesAsync processes all pending messages for a session
func processPendingMessagesAsync(ctx context.Context, userID, companyID int64, req *SendMessageRequest) {
	mu, pendingMessages := chat_session.GetDefault().PopAllPendingMessages(userID, companyID)

	if len(pendingMessages) == 0 {
		chat_session.GetDefault().ReleaseSessionLock(mu)
		return
	}

	// Save all pending messages to session
	for _, msg := range pendingMessages {
		if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, msg); err != nil {
		}
	}

	// Notify listeners once after saving all messages
	for _, msg := range pendingMessages {
		chat_session.GetDefault().NotifyListeners(userID, companyID, msg)
	}

	// Call agent once to process all messages
	processAgentInternal(ctx, userID, companyID, req)
	chat_session.GetDefault().ReleaseSessionLock(mu)
}

// processAgentInternal processes agent internally (assumes lock is held)
func processAgentInternal(ctx context.Context, userID, companyID int64, req *SendMessageRequest) {
	routeName := ""
	var routeParams map[string]interface{}
	if req != nil && req.PageInfo != nil {
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
		CompanyID:      companyID,
		CurrentRoute:   routeName,
		RouteParams:    routeParams,
		SessionData:    make(map[string]interface{}),
		MessageHistory: []ai.Message{},
		Language:       u.Language,
	}

	session, err := chat_session.GetDefault().GetOrCreateSessionWithoutLock(userID, companyID)
	if err != nil {
		return
	}

	agentCtx.SubordinateStaff = session.SubordinateStaff
	agentCtx.CurrentTask = session.CurrentTask

	// Build message history from session (including all messages)
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
			agentCtx.MessageHistory = append(agentCtx.MessageHistory, ai.Message{
				Role:    "user",
				Content: msg.Content,
			})
		}
	}

	agentResponse, err := agent.ProcessMessage(ctx, agentCtx)
	if err != nil {
		return
	}

	for _, response := range agentResponse.Responses {
		assistantMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

		metadata := make(map[string]interface{})
		if len(response.ExecutionSteps) > 0 {
			metadata["execution_steps"] = response.ExecutionSteps
		}
		if response.TokensUsed > 0 {
			metadata["tokens_used"] = response.TokensUsed
		}

		hasQuestion := false
		var questionData string

		for _, step := range response.ExecutionSteps {
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
		for _, step := range response.ExecutionSteps {
			toolCallID := fmt.Sprintf("call_%d", time.Now().UnixNano())

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
			if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, toolCallMsg); err != nil {
			}
			chat_session.GetDefault().NotifyListeners(userID, companyID, toolCallMsg)

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
			if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, toolResultMsg); err != nil {
			}
			chat_session.GetDefault().NotifyListeners(userID, companyID, toolResultMsg)
		}

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
			if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, questionMessage); err != nil {
			}
			chat_session.GetDefault().NotifyListeners(userID, companyID, questionMessage)
			continue
		}

		if response.Content != "" {
			assistantMessage := chat_session.Message{
				ID:        assistantMsgID,
				Type:      "assistant_response",
				Role:      "assistant",
				Content:   response.Content,
				Timestamp: time.Now().Unix(),
				Metadata:  metadata,
				CompanyID: companyID,
			}
			if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, assistantMessage); err != nil {
			}
			chat_session.GetDefault().NotifyListeners(userID, companyID, assistantMessage)
		}

		if response.ButtonNavigation != nil {
			buttonNavMsg := chat_session.Message{
				ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
				Type:      "button_navigation",
				Role:      "assistant",
				Content:   "",
				Timestamp: time.Now().Unix(),
				ButtonNavigation: &chat_session.ButtonNavigation{
					RouteName: response.ButtonNavigation.RouteName,
					Params:    response.ButtonNavigation.Params,
					Label:     response.ButtonNavigation.Label,
					Title:     response.ButtonNavigation.Title,
				},
				CompanyID: companyID,
			}
			if err := chat_session.GetDefault().SaveMessageToSession(userID, companyID, buttonNavMsg); err != nil {
			}
			chat_session.GetDefault().NotifyListeners(userID, companyID, buttonNavMsg)
		}
	}
}

// processQuestionAnswerInternal processes a question answer internally (assumes lock is held)

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

	chat_session.GetDefault().PushPendingMessage(userID, req.CompanyID, answerMessage)

	go processPendingMessagesAsync(context.Background(), userID, req.CompanyID, nil)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        answerMsgID,
		"status":    "processing",
		"timestamp": time.Now().Unix(),
	})
}

// SetCurrentTask sets the current task for a chat session
func SetCurrentTask(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	userID := a.GetID()

	req := new(SetCurrentTaskRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	if !isUserAllowedForChat(a, req.CompanyID) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	if err := chat_session.GetDefault().SetCurrentTask(userID, req.CompanyID, req.TaskID, req.Title, req.ProjectID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set current task: %v", err))
	}

	return c.JSON(http.StatusOK, models.Message{Message: "Current task set successfully"})
}
