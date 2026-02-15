package chat_session

import (
	"fmt"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/company"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/modules/keyvalue"
	"code.vikunja.io/api/pkg/user"
)

const (
	sessionPrefix   = "chat_session:"
	sessionTTL      = time.Hour
	cleanupInterval = 5 * time.Minute
)

// SubordinateStaffInfo represents information about a subordinate staff member
type SubordinateStaffInfo struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	ProjectID int64  `json:"project_id"`
}

// ChatSession represents a chat session in memory
type ChatSession struct {
	ID               string                 `json:"id"`
	UserID           int64                  `json:"user_id"`
	CompanyID        int64                  `json:"company_id"`
	CreatedAt        time.Time              `json:"created_at"`
	ExpiresAt        time.Time              `json:"expires_at"`
	Messages         []Message              `json:"messages"`
	SubordinateStaff []SubordinateStaffInfo `json:"subordinate_staff"`
}

// Message represents a chat message
type Message struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"` // "user_input", "tool_call", "tool_result", "assistant_response", "question", "button_navigation"
	Role              string                 `json:"role"` // "user" | "assistant" | "tool"
	Content           string                 `json:"content"`
	Timestamp         int64                  `json:"timestamp"`
	NavigationCommand *NavigationCommand     `json:"navigationCommand,omitempty"`
	ToolName          string                 `json:"toolName,omitempty"`
	ToolInput         string                 `json:"toolInput,omitempty"`
	ToolOutput        string                 `json:"toolOutput,omitempty"`
	ToolCallID        string                 `json:"toolCallID,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CompanyID         int64                  `json:"company_id,omitempty"`
	QuestionData      string                 `json:"questionData,omitempty"`
	ButtonNavigation  *ButtonNavigation      `json:"buttonNavigation,omitempty"`
}

// NavigationCommand represents a navigation action
type NavigationCommand struct {
	RouteName string                 `json:"routeName"`
	Params    map[string]interface{} `json:"params"`
	Label     string                 `json:"label"`
	Title     string                 `json:"title,omitempty"`
}

// ButtonNavigation represents a button-based navigation action
type ButtonNavigation struct {
	RouteName string                 `json:"routeName"`
	Params    map[string]interface{} `json:"params"`
	Label     string                 `json:"label"`
	Title     string                 `json:"title,omitempty"`
}

// Manager manages chat sessions in memory
type Manager struct {
	mu        sync.RWMutex
	listeners map[string][]chan Message
}

var defaultManager = &Manager{}

// getExistingSession gets an existing session without locking
func (m *Manager) getExistingSession(userID, companyID int64) (*ChatSession, error) {
	sessionKey := getSessionKey(userID, companyID)

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

// getSubordinateStaff retrieves subordinate staff information for a user in a company
func getSubordinateStaff(userID, companyID int64) (result []SubordinateStaffInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[Chat] Recovered from panic in getSubordinateStaff: %v\n", r)
			result = []SubordinateStaffInfo{}
			err = nil
		}
	}()

	if companyID <= 0 {
		return []SubordinateStaffInfo{}, nil
	}

	s := db.NewSession()
	if s == nil {
		return []SubordinateStaffInfo{}, nil
	}
	defer s.Close()

	var relations []*company.CompanyRelation
	err = s.Where("company_id = ? AND superior_user_id = ?", companyID, userID).Find(&relations)
	if err != nil {
		return []SubordinateStaffInfo{}, nil
	}

	if len(relations) == 0 {
		return []SubordinateStaffInfo{}, nil
	}

	subordinateUserIDs := make([]int64, len(relations))
	for i, rel := range relations {
		subordinateUserIDs[i] = rel.SubordinateUserID
	}

	users, err := user.GetUsersByIDs(s, subordinateUserIDs)
	if err != nil {
		return []SubordinateStaffInfo{}, nil
	}

	projectIDMap := make(map[int64]int64)
	for _, rel := range relations {
		projectIDMap[rel.SubordinateUserID] = rel.ProjectID
	}

	staffInfo := make([]SubordinateStaffInfo, 0, len(users))
	for _, u := range users {
		staffInfo = append(staffInfo, SubordinateStaffInfo{
			UserID:    u.ID,
			Username:  u.Username,
			Name:      u.Name,
			ProjectID: projectIDMap[u.ID],
		})
	}

	return staffInfo, nil
}

// createNewSessionWithoutLock creates a new chat session without locking
func (m *Manager) createNewSessionWithoutLock(userID, companyID int64) (*ChatSession, error) {
	now := time.Now()

	subordinateStaff, err := getSubordinateStaff(userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load subordinate staff: %w", err)
	}

	currentUser, err := user.GetUserByID(db.NewSession(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load current user: %w", err)
	}

	allStaff := append(subordinateStaff, SubordinateStaffInfo{
		UserID:    currentUser.ID,
		Username:  currentUser.Username,
		Name:      currentUser.Name,
		ProjectID: currentUser.DefaultProjectID,
	})

	session := ChatSession{
		ID:               generateSessionID(),
		UserID:           userID,
		CompanyID:        companyID,
		CreatedAt:        now,
		ExpiresAt:        now.Add(sessionTTL),
		Messages:         []Message{},
		SubordinateStaff: allStaff,
	}

	sessionKey := getSessionKey(userID, companyID)
	if err := keyvalue.Put(sessionKey, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &session, nil
}

// GetOrCreateSession gets an existing session for a user or creates a new one
func (m *Manager) GetOrCreateSession(userID, companyID int64) (*ChatSession, error) {
	session, err := m.getExistingSession(userID, companyID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		return session, nil
	}

	return m.createNewSessionWithoutLock(userID, companyID)
}

// RegisterListener registers a listener for a user's session updates
func (m *Manager) RegisterListener(userID, companyID int64, listener chan Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listeners == nil {
		m.listeners = make(map[string][]chan Message)
	}
	key := fmt.Sprintf("%d:%d", userID, companyID)
	m.listeners[key] = append(m.listeners[key], listener)
	fmt.Printf("[Chat] Registered listener for user %d, company %d, total listeners: %d\n", userID, companyID, len(m.listeners[key]))
}

// UnregisterListener removes a listener for a user
func (m *Manager) UnregisterListener(userID, companyID int64, listener chan Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d:%d", userID, companyID)
	listeners, exists := m.listeners[key]
	if !exists {
		return
	}
	for i, l := range listeners {
		if l == listener {
			m.listeners[key] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

// notifyListeners notifies all registered listeners of a new message
func (m *Manager) notifyListeners(userID, companyID int64, msg Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%d:%d", userID, companyID)
	listeners, exists := m.listeners[key]
	if !exists {
		fmt.Printf("[Chat] No listeners for user %d, company %d\n", userID, companyID)
		return
	}
	fmt.Printf("[Chat] Notifying %d listeners for user %d, company %d, message type: %s, id: %s\n", len(listeners), userID, companyID, msg.Type, msg.ID)
	sentCount := 0
	for _, listener := range listeners {
		select {
		case listener <- msg:
			sentCount++
		default:
			fmt.Printf("[Chat] Listener channel full, skipping\n")
		}
	}
	fmt.Printf("[Chat] Sent message to %d/%d listeners\n", sentCount, len(listeners))
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(userID, companyID int64, msg Message) error {
	m.mu.Lock()

	session, err := m.GetOrCreateSession(userID, companyID)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	// Create a new session with the message appended
	updatedSession := *session
	updatedSession.Messages = append(updatedSession.Messages, msg)
	updatedSession.ExpiresAt = time.Now().Add(sessionTTL)

	sessionKey := getSessionKey(userID, companyID)
	if err := keyvalue.Put(sessionKey, updatedSession); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Save message data for notification
	notifyMsg := msg

	// Release lock before notifying listeners to avoid deadlock
	m.mu.Unlock()
	m.notifyListeners(userID, companyID, notifyMsg)

	return nil
}

// ClearSession removes a session for a user in a company
func (m *Manager) ClearSession(userID, companyID int64) error {
	sessionKey := getSessionKey(userID, companyID)
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

// getSessionKey generates a storage key for a user's session in a company
func getSessionKey(userID, companyID int64) string {
	return fmt.Sprintf("%s%d:%d", sessionPrefix, userID, companyID)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("ses_%d_%d", time.Now().UnixNano(), time.Now().Unix())
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
