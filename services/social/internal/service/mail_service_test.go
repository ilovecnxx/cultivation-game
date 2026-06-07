package service

import (
	"testing"
	"time"

	"cultivation-game/services/social/internal/model"
)

func TestMailModel_Structure(t *testing.T) {
	now := time.Now()
	expireAt := now.Add(7 * 24 * time.Hour)

	mail := &model.Mail{
		ID:         "mail_01",
		MailType:   model.MailSystem,
		Title:      "Welcome!",
		Content:    "Welcome to the game!",
		SenderID:   "system",
		SenderName: "系统",
		ReceiverID: "player1",
		Attachments: []model.MailAttachment{
			{ItemID: "item_001", ItemName: "灵芝草", Quantity: 5},
			{CoinType: "spirit_stone", CoinAmount: 100},
		},
		ExpireAt:  expireAt,
		CreatedAt: now,
	}

	if mail.ID != "mail_01" {
		t.Errorf("ID = %q", mail.ID)
	}
	if mail.MailType != model.MailSystem {
		t.Errorf("MailType = %q", mail.MailType)
	}
	if mail.ReceiverID != "player1" {
		t.Errorf("ReceiverID = %q", mail.ReceiverID)
	}
	if len(mail.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(mail.Attachments))
	}
}

func TestMailModel_MailTypes(t *testing.T) {
	tests := []struct {
		name     string
		mailType model.MailType
	}{
		{"System", model.MailSystem},
		{"Player", model.MailPlayer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mailType) == "" {
				t.Errorf("mail type %s should not be empty", tt.name)
			}
		})
	}

	// Ensure distinct
	if model.MailSystem == model.MailPlayer {
		t.Error("mail types should be distinct")
	}
}

func TestMailAttachment_Structure(t *testing.T) {
	tests := []struct {
		name       string
		attachment model.MailAttachment
		checkItem  bool
		checkCoin  bool
	}{
		{
			name: "item attachment",
			attachment: model.MailAttachment{
				ItemID:   "item_001",
				ItemName: "灵芝草",
				Quantity: 5,
			},
			checkItem: true,
		},
		{
			name: "coin attachment",
			attachment: model.MailAttachment{
				CoinType:   "spirit_stone",
				CoinAmount: 100,
			},
			checkCoin: true,
		},
		{
			name: "combined attachment",
			attachment: model.MailAttachment{
				ItemID:     "item_002",
				ItemName:   "聚气丹",
				Quantity:   3,
				CoinType:   "gold",
				CoinAmount: 50,
			},
			checkItem: true,
			checkCoin: true,
		},
		{
			name: "empty attachment",
			attachment: model.MailAttachment{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkItem {
				if tt.attachment.ItemID == "" {
					t.Error("ItemID should not be empty")
				}
				if tt.attachment.Quantity <= 0 {
					t.Error("Quantity should be positive")
				}
			}
			if tt.checkCoin {
				if tt.attachment.CoinType == "" {
					t.Error("CoinType should not be empty")
				}
				if tt.attachment.CoinAmount <= 0 {
					t.Error("CoinAmount should be positive")
				}
			}
		})
	}
}

func TestMailModel_Expiration(t *testing.T) {
	now := time.Now()

	mail := &model.Mail{
		ExpireAt: now.Add(-1 * time.Hour), // expired 1 hour ago
	}

	if !mail.ExpireAt.Before(now) {
		t.Error("expired mail should have ExpireAt in the past")
	}

	mail.ExpireAt = now.Add(7 * 24 * time.Hour) // expires in 7 days
	if mail.ExpireAt.Before(now) {
		t.Error("active mail should have ExpireAt in the future")
	}
}

func TestMailModel_ReadAndClaimed(t *testing.T) {
	// Test default values
	mail := &model.Mail{}

	if mail.IsRead {
		t.Error("new mail should not be read by default")
	}
	if mail.IsClaimed {
		t.Error("new mail should not be claimed by default")
	}

	// Test transitions
	mail.IsRead = true
	if !mail.IsRead {
		t.Error("mail should be read after setting")
	}

	mail.IsClaimed = true
	if !mail.IsClaimed {
		t.Error("mail should be claimed after setting")
	}
}

func TestNewMailService(t *testing.T) {
	svc := NewMailService(nil, nil)

	if svc.repo != nil {
		t.Error("repo should be nil when passed nil")
	}
	if svc.friendSvc != nil {
		t.Error("friendSvc should be nil when passed nil")
	}
	// Player service addr should default to localhost
	if svc.playerServiceAddr == "" {
		t.Error("playerServiceAddr should have a default value")
	}
}
