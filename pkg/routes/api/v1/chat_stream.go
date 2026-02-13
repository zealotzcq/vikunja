package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/api/pkg/modules/chat_session"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// ChatUpdate represents an update pushed to the client via SSE
type ChatUpdate struct {
	LastMessageID string                 `json:"last_message_id"`
	MessageType   string                 `json:"message_type"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

// ChatStream handles SSE connections for real-time chat updates
func ChatStream(c *echo.Context) error {
	// Try to get token from query parameter (for SSE connections)
	token := c.QueryParam("token")
	var a web.Auth
	var err error

	if token != "" {
		// Manually parse JWT token from query parameter
		parsedToken, parseErr := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.ServiceJWTSecret.GetString()), nil
		})
		if parseErr != nil {
			log.Printf("[Chat] Failed to parse token: %v", parseErr)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
		}

		// Check token type
		typ, ok := claims["type"].(float64)
		if !ok || int(typ) != auth.AuthTypeUser {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token type")
		}

		// Get user from claims
		a, err = user.GetUserFromClaims(claims)
		if err != nil {
			log.Printf("[Chat] Failed to get user from claims: %v", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid user")
		}
	} else {
		// Try to use regular auth
		a, err = auth.GetAuthFromClaims(c)
		if err != nil {
			return err
		}
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	if !isUserAllowedForChat(a) {
		return echo.NewHTTPError(http.StatusForbidden, "Chat assistant is not available for your account")
	}

	userID := a.GetID()

	log.Printf("[Chat] SSE connected for user %d", userID)

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Response().(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming not supported")
	}

	lastEventID := c.Request().Header.Get("Last-Event-ID")
	fmt.Printf("[Chat] SSE connection for user %d, Last-Event-ID: %s\n", userID, lastEventID)

	listener := make(chan chat_session.Message, 10)
	chat_session.GetDefault().RegisterListener(userID, listener)
	defer chat_session.GetDefault().UnregisterListener(userID, listener)

	if lastEventID != "" {
		session, err := chat_session.GetDefault().GetOrCreateSession(userID)
		if err == nil && len(session.Messages) > 0 {
			latestMsg := session.Messages[len(session.Messages)-1]
			if latestMsg.ID != lastEventID {
				update := ChatUpdate{
					LastMessageID: latestMsg.ID,
					MessageType:   getMessageTypeForFrontend(latestMsg.Type),
					Payload:       messageToPayload(latestMsg),
				}
				sendSSEUpdate(c, flusher, update)
			}
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := c.Request().Context()

	for {
		select {
		case msg := <-listener:
			fmt.Printf("[Chat] SSE received message for user %d: type=%s, id=%s\n", userID, msg.Type, msg.ID)
			msgType := getMessageTypeForFrontend(msg.Type)
			if msgType != "" {
				update := ChatUpdate{
					LastMessageID: msg.ID,
					MessageType:   msgType,
					Payload:       messageToPayload(msg),
				}
				fmt.Printf("[Chat] SSE sending update: last_message_id=%s, message_type=%s\n", update.LastMessageID, update.MessageType)
				sendSSEUpdate(c, flusher, update)
			} else {
				fmt.Printf("[Chat] SSE message type %s not needed for frontend\n", msg.Type)
			}

		case <-ticker.C:
			sendKeepAlive(c, flusher)

		case <-ctx.Done():
			fmt.Printf("[Chat] SSE connection closed for user %d\n", userID)
			return nil
		}
	}
}

func getMessageTypeForFrontend(msgType string) string {
	switch msgType {
	case "user_input", "assistant_response":
		return "frontend_needed"
	default:
		return "internal"
	}
}

func messageToPayload(msg chat_session.Message) map[string]interface{} {
	payload := map[string]interface{}{
		"id":        msg.ID,
		"type":      msg.Type,
		"role":      msg.Role,
		"content":   msg.Content,
		"timestamp": msg.Timestamp,
	}

	if msg.NavigationCommand != nil {
		payload["navigationCommand"] = msg.NavigationCommand
	}

	if msg.Metadata != nil {
		payload["metadata"] = msg.Metadata
	}

	return payload
}

func sendSSEUpdate(c *echo.Context, flusher http.Flusher, update ChatUpdate) {
	data := fmt.Sprintf("data: %s\n\n", toJSON(update))
	if _, err := fmt.Fprint(c.Response(), data); err != nil {
		return
	}
	flusher.Flush()
}

func sendKeepAlive(c *echo.Context, flusher http.Flusher) {
	if _, err := fmt.Fprint(c.Response(), ": keep-alive\n\n"); err != nil {
		return
	}
	flusher.Flush()
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to serialize"}`
	}
	return string(b)
}
