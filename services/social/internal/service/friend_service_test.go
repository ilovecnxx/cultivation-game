package service

import (
	"testing"

	"cultivation-game/services/social/internal/model"
)

func TestNewFriendService(t *testing.T) {
	svc := NewFriendService(nil, nil)

	if svc.db != nil {
		t.Error("db should be nil when passed nil")
	}
	if svc.redis != nil {
		t.Error("redis should be nil when passed nil")
	}
}

func TestFriendModel_StatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status model.FriendStatus
	}{
		{"Normal", model.FriendStatusNormal},
		{"Blacked", model.FriendStatusBlacked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) == "" {
				t.Errorf("status %s should not be empty", tt.name)
			}
		})
	}

	// Verify they are distinct
	if model.FriendStatusNormal == model.FriendStatusBlacked {
		t.Error("friend status constants should be distinct")
	}
}

func TestFriendApply_Structure(t *testing.T) {
	apply := &model.FriendApply{
		ID:       "apply_01",
		FromID:   "user1",
		FromName: "Player1",
		ToID:     "user2",
		Message:  "Let's be friends!",
		Status:   "pending",
	}

	if apply.ID != "apply_01" {
		t.Errorf("ID = %q", apply.ID)
	}
	if apply.FromID != "user1" {
		t.Errorf("FromID = %q", apply.FromID)
	}
	if apply.ToID != "user2" {
		t.Errorf("ToID = %q", apply.ToID)
	}
	if apply.Status != "pending" {
		t.Errorf("Status = %q", apply.Status)
	}
}

func TestFriend_Structure(t *testing.T) {
	friend := &model.Friend{
		UserID:   "user1",
		FriendID: "friend1",
		Status:   model.FriendStatusNormal,
		Remark:   "best friend",
	}

	if friend.UserID != "user1" {
		t.Errorf("UserID = %q", friend.UserID)
	}
	if friend.FriendID != "friend1" {
		t.Errorf("FriendID = %q", friend.FriendID)
	}
	if friend.Status != model.FriendStatusNormal {
		t.Errorf("Status = %q", friend.Status)
	}
	if friend.Remark != "best friend" {
		t.Errorf("Remark = %q", friend.Remark)
	}
}

func TestFriendApply_StatusTransitions(t *testing.T) {
	// Verify valid status transitions
	validStatuses := map[string]bool{
		"pending":  true,
		"accepted": true,
		"rejected": true,
	}

	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{"pending", "pending", true},
		{"accepted", "accepted", true},
		{"rejected", "rejected", true},
		{"invalid", "unknown", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if validStatuses[tt.status] != tt.valid {
				t.Errorf("status %q validity = %v, want %v", tt.status, !tt.valid, tt.valid)
			}
		})
	}
}
