package chat_session

import (
	"fmt"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/modules/keyvalue"
)

const (
	sessionPrefix   = "chat_session:"
	sessionTTL      = time.Hour
	cleanupInterval = 5 * time.Minute
)

// ChatSession represents a chat session in memory
type ChatSession struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Messages  []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// Manager manages chat sessions in memory
type Manager struct {
	mu sync.RWMutex
}

var defaultManager = &Manager{}

// getExistingSession gets an existing session without locking
func (m *Manager) getExistingSession(userID int64) (*ChatSession, error) {
	sessionKey := getSessionKey(userID)

	sessionData, exists, err := keyvalue.Get(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if !exists {
		return nil, nil
	}

	session, ok := sessionData.(ChatSession)
	if !ok {
		return nil, fmt.Errorf("invalid session data type")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Session expired
		return nil, nil
	}

	// Session is still valid, extend expiration and store updated session
	updatedSession := session
	updatedSession.ExpiresAt = time.Now().Add(sessionTTL)
	if err := keyvalue.Put(sessionKey, updatedSession); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &updatedSession, nil
}

// createNewSessionWithoutLock creates a new chat session without locking
func (m *Manager) createNewSessionWithoutLock(userID int64) (*ChatSession, error) {
	now := time.Now()
	session := ChatSession{
		ID:        generateSessionID(),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
		Messages:  []Message{},
	}

	sessionKey := getSessionKey(userID)
	if err := keyvalue.Put(sessionKey, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &session, nil
}

// GetOrCreateSession gets an existing session for a user or creates a new one
func (m *Manager) GetOrCreateSession(userID int64) (*ChatSession, error) {
	session, err := m.getExistingSession(userID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		return session, nil
	}

	return m.createNewSessionWithoutLock(userID)
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(userID int64, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.GetOrCreateSession(userID)
	if err != nil {
		return err
	}

	// Create a new session with the message appended
	updatedSession := *session
	updatedSession.Messages = append(updatedSession.Messages, msg)
	updatedSession.ExpiresAt = time.Now().Add(sessionTTL)

	sessionKey := getSessionKey(userID)
	if err := keyvalue.Put(sessionKey, updatedSession); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// ClearSession removes a session for a user
func (m *Manager) ClearSession(userID int64) error {
	sessionKey := getSessionKey(userID)
	if err := keyvalue.Del(sessionKey); err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes all expired chat sessions
func (m *Manager) CleanupExpiredSessions() error {
	keys, err := keyvalue.ListKeys(sessionPrefix)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	now := time.Now()
	var keysToDelete []string

	for _, key := range keys {
		sessionData, exists, err := keyvalue.Get(key)
		if err != nil || !exists {
			continue
		}

		session, ok := sessionData.(ChatSession)
		if !ok {
			continue
		}

		if now.After(session.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if err := keyvalue.Del(key); err != nil {
			return fmt.Errorf("failed to delete expired session %s: %w", key, err)
		}
	}

	return nil
}

// GetDefault returns the default session manager
func GetDefault() *Manager {
	return defaultManager
}

// getSessionKey generates a storage key for a user's session
func getSessionKey(userID int64) string {
	return fmt.Sprintf("%s%d", sessionPrefix, userID)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("ses_%d", time.Now().UnixNano())
}

// StartCleanupWorker starts a background worker to clean up expired sessions
func StartCleanupWorker() {
	ticker := time.NewTicker(cleanupInterval)
	go func() {
		for range ticker.C {
			if err := defaultManager.CleanupExpiredSessions(); err != nil {
				fmt.Printf("Failed to cleanup expired chat sessions: %v\n", err)
			}
		}
	}()
}
