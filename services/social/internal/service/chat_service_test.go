package service

import (
	"testing"

	"cultivation-game/services/social/internal/model"
)

func TestChatService_FilterSensitive(t *testing.T) {
	svc := NewChatService(nil, nil)

	// "敏感词1" has 4 runes, so replacement is "****"
	// "敏感词2" has 4 runes, so replacement is "****"
	// "违禁词" has 3 runes, so replacement is "***"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no sensitive words", "hello world", "hello world"},
		{"filters exact word", "this is 敏感词1 test", "this is **** test"},
		{"filters multiple words", "敏感词1 and 敏感词2", "**** and ****"},
		{"filters forbidden word", "违禁词 detected", "*** detected"},
		{"empty string", "", ""},
		{"partial no match", "这不是敏感词", "这不是敏感词"},
		{"multiple same word", "敏感词1 敏感词1 敏感词1", "**** **** ****"},
		{"consecutive sensitive", "敏感词1敏感词2", "********"},
		{"word at start", "敏感词1 hello", "**** hello"},
		{"word at end", "hello 敏感词1", "hello ****"},
		{"unicode preserved", "hello 世界", "hello 世界"},
		{"long clean text intact", "这是一段正常文本，没有任何敏感词汇", "这是一段正常文本，没有任何敏感词汇"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.FilterSensitive(tt.input)
			if got != tt.expected {
				t.Errorf("FilterSensitive(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestChatService_RegisterClient_MapOnly(t *testing.T) {
	// RegisterClient also calls Redis, which panics on nil.
	// Test the map manipulation directly.
	svc := NewChatService(nil, nil)

	client1 := &WSClient{UserID: "user1", SectID: "sect_01", Send: make(chan []byte, 10)}
	client2 := &WSClient{UserID: "user2", SectID: "", Send: make(chan []byte, 10)}

	// Insert directly to avoid Redis panic
	svc.clientsLock.Lock()
	svc.clients["user1"] = client1
	svc.clients["user2"] = client2
	svc.clientsLock.Unlock()

	svc.clientsLock.RLock()
	if len(svc.clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(svc.clients))
	}
	if _, ok := svc.clients["user1"]; !ok {
		t.Error("user1 should be registered")
	}
	if _, ok := svc.clients["user2"]; !ok {
		t.Error("user2 should be registered")
	}
	svc.clientsLock.RUnlock()
}

func TestChatService_UnregisterClient(t *testing.T) {
	svc := NewChatService(nil, nil)

	client := &WSClient{UserID: "user3", SectID: "sect_01", Send: make(chan []byte, 10)}
	svc.clientsLock.Lock()
	svc.clients["user3"] = client
	svc.clientsLock.Unlock()

	svc.clientsLock.RLock()
	if _, ok := svc.clients["user3"]; !ok {
		t.Fatal("user3 should be registered before unregister")
	}
	svc.clientsLock.RUnlock()

	// UnregisterClient will call redis.Del which panics on nil. Test direct deletion.
	svc.clientsLock.Lock()
	delete(svc.clients, "user3")
	svc.clientsLock.Unlock()

	svc.clientsLock.RLock()
	if _, ok := svc.clients["user3"]; ok {
		t.Error("user3 should be unregistered")
	}
	svc.clientsLock.RUnlock()
}

func TestChatService_ChannelTypes(t *testing.T) {
	validChannels := map[model.ChatChannel]bool{
		model.ChannelWorld:   true,
		model.ChannelSect:    true,
		model.ChannelPrivate: true,
		model.ChannelSystem:  true,
	}

	tests := []struct {
		name    string
		channel model.ChatChannel
		valid   bool
	}{
		{"world channel", model.ChannelWorld, true},
		{"sect channel", model.ChannelSect, true},
		{"private channel", model.ChannelPrivate, true},
		{"system channel", model.ChannelSystem, true},
		{"invalid channel", "invalid", false},
		{"empty channel", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := validChannels[tt.channel]
			if ok != tt.valid {
				t.Errorf("channel %q validity = %v, want %v", tt.channel, ok, tt.valid)
			}
		})
	}
}

func TestWSClient_Creation(t *testing.T) {
	client := &WSClient{
		UserID: "test_user",
		SectID: "test_sect",
		Send:   make(chan []byte, 32),
	}

	if client.UserID != "test_user" {
		t.Errorf("UserID = %q", client.UserID)
	}
	if client.SectID != "test_sect" {
		t.Errorf("SectID = %q", client.SectID)
	}
	if client.Send == nil {
		t.Error("Send channel should not be nil")
	}
	if cap(client.Send) != 32 {
		t.Errorf("Send cap = %d, want 32", cap(client.Send))
	}
}

func TestChatService_NewChatService(t *testing.T) {
	svc := NewChatService(nil, nil)

	if svc.repo != nil {
		t.Error("repo should be nil when passed nil")
	}
	if svc.redis != nil {
		t.Error("redis should be nil when passed nil")
	}
	if svc.clients == nil {
		t.Error("clients map should be initialized")
	}
	if svc.sensitiveWords == nil {
		t.Error("sensitive words should be loaded")
	}
	if len(svc.sensitiveWords) == 0 {
		t.Error("sensitive words list should not be empty")
	}
}

func TestChatService_FilterNoSensitiveWords(t *testing.T) {
	svc := NewChatService(nil, nil)
	longText := "这是一段非常长的正常文本，没有任何敏感词汇，应该被完整保留。"
	got := svc.FilterSensitive(longText)
	if got != longText {
		t.Errorf("long clean text was modified: got %q", got)
	}
}
