package chat_session

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/modules/keyvalue"
)

func init() {
	keyvalue.InitStorage()
}

func TestSubordinateStaffInSession(t *testing.T) {
	userID := int64(1001)
	companyID := int64(1001)

	GetDefault().ClearSession(userID, companyID)

	session, err := GetDefault().GetOrCreateSession(userID, companyID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, session.UserID)
	}

	if session.CompanyID != companyID {
		t.Errorf("Expected CompanyID %d, got %d", companyID, session.CompanyID)
	}

	if session.SubordinateStaff == nil {
		t.Error("SubordinateStaff should not be nil")
	}

	GetDefault().ClearSession(userID, companyID)
}

func TestSessionWithCompanyID(t *testing.T) {
	userID := int64(1002)
	companyID1 := int64(2001)
	companyID2 := int64(2002)

	GetDefault().ClearSession(userID, companyID1)
	GetDefault().ClearSession(userID, companyID2)

	session1, err := GetDefault().GetOrCreateSession(userID, companyID1)
	if err != nil {
		t.Fatalf("Failed to create session1: %v", err)
	}

	time.Sleep(1 * time.Millisecond)

	session2, err := GetDefault().GetOrCreateSession(userID, companyID2)
	if err != nil {
		t.Fatalf("Failed to create session2: %v", err)
	}

	if session1.ID == session2.ID {
		t.Errorf("Sessions should have different IDs for different companies, got %s for both", session1.ID)
	}

	if session1.CompanyID == session2.CompanyID {
		t.Error("Sessions should have different company IDs")
	}

	msg := Message{
		ID:        "msg_test_1",
		Type:      "user_input",
		Role:      "user",
		Content:   "Test message for company 1",
		Timestamp: time.Now().Unix(),
		CompanyID: companyID1,
	}

	if err := GetDefault().AddMessage(userID, companyID1, msg); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	session1, err = GetDefault().GetOrCreateSession(userID, companyID1)
	if err != nil {
		t.Fatalf("Failed to get session1: %v", err)
	}

	session2, err = GetDefault().GetOrCreateSession(userID, companyID2)
	if err != nil {
		t.Fatalf("Failed to get session2: %v", err)
	}

	if len(session1.Messages) != 1 {
		t.Errorf("Expected 1 message in session1, got %d", len(session1.Messages))
	}

	if len(session2.Messages) != 0 {
		t.Errorf("Expected 0 messages in session2, got %d", len(session2.Messages))
	}

	GetDefault().ClearSession(userID, companyID1)
	GetDefault().ClearSession(userID, companyID2)
}
