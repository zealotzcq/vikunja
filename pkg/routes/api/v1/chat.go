package v1

import (
	"fmt"
	"net/http"
	"time"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/ai"
	"code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/api/pkg/modules/chat_session"

	"github.com/labstack/echo/v5"
)

// SendMessageRequest represents a request to send a chat message
type SendMessageRequest struct {
	Message  string    `json:"message" validate:"required"`
	PageInfo *PageInfo `json:"page_info"`
}

// PageInfo represents current page context
type PageInfo struct {
	RouteName string                 `json:"route_name" validate:"required"`
	Params    map[string]interface{} `json:"params"`
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
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

	// Generate user message ID
	userMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Add user message to session
	userMessage := chat_session.Message{
		ID:        userMsgID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now().Unix(),
	}

	if err := chat_session.GetDefault().AddMessage(userID, userMessage); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to save message: %v", err))
	}

	// Generate AI response based on message and page context
	routeName := ""
	var routeParams map[string]interface{}
	if req.PageInfo != nil {
		routeName = req.PageInfo.RouteName
		routeParams = req.PageInfo.Params
	}

	aiResponse, _, _ := ai.GenerateResponse(
		req.Message,
		routeName,
		routeParams,
	)

	// Generate assistant message ID
	assistantMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Add assistant message to session
	assistantMessage := chat_session.Message{
		ID:        assistantMsgID,
		Role:      "assistant",
		Content:   aiResponse,
		Timestamp: time.Now().Unix(),
	}

	if err := chat_session.GetDefault().AddMessage(userID, assistantMessage); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to save assistant message: %v", err))
	}

	// Return the assistant message
	return c.JSON(http.StatusOK, ChatMessage{
		ID:        assistantMsgID,
		Role:      "assistant",
		Content:   aiResponse,
		Timestamp: time.Now().Unix(),
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

	userID := a.GetID()

	// Clear the session
	if err := chat_session.GetDefault().ClearSession(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to clear session: %v", err))
	}

	return c.JSON(http.StatusOK, models.Message{Message: "Session cleared successfully"})
}
