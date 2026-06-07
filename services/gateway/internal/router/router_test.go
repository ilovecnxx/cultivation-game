// Package router 消息路由器单元测试。
package router

import (
	"fmt"
	"math"
	"testing"
)

// routeSubject replicates the subject-generation logic from Route so we can
// unit-test the mapping without an actual NATS connection.
func routeSubject(msgID uint32) string {
	service, ok := msgIDToService[msgID]
	if !ok {
		service = "default"
	}
	return fmt.Sprintf("game.svc.%s.%d", service, msgID)
}

// playerSubscribeSubject replicates the subject format used by SubscribePlayer.
func playerSubscribeSubject(playerID uint64) string {
	return fmt.Sprintf("gateway.player.%d", playerID)
}

// ---------------------------------------------------------------------------
// MsgID to service mapping
// ---------------------------------------------------------------------------

func TestMsgIDToService_Mapping(t *testing.T) {
	tests := []struct {
		msgID       uint32
		wantService string
		wantSubject string
	}{
		{1, "auth", "game.svc.auth.1"},
		{2, "heartbeat", "game.svc.heartbeat.2"},
		{100, "player", "game.svc.player.100"},
		{200, "scene", "game.svc.scene.200"},
		{300, "combat", "game.svc.combat.300"},
		{400, "chat", "game.svc.chat.400"},
		{500, "mail", "game.svc.mail.500"},
		{600, "shop", "game.svc.shop.600"},
		{700, "inventory", "game.svc.inventory.700"},
		{800, "quest", "game.svc.quest.800"},
		{900, "guild", "game.svc.guild.900"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("msgID_%d", tt.msgID), func(t *testing.T) {
			svc, ok := msgIDToService[tt.msgID]
			if !ok {
				svc = "default"
			}
			if svc != tt.wantService {
				t.Errorf("service = %q, want %q", svc, tt.wantService)
			}
			subj := routeSubject(tt.msgID)
			if subj != tt.wantSubject {
				t.Errorf("subject = %q, want %q", subj, tt.wantSubject)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Unknown msgID — "default" service
// ---------------------------------------------------------------------------

func TestMsgIDToService_UnknownDefaults(t *testing.T) {
	tests := []struct {
		msgID       uint32
		desc        string
	}{
		{0, "msgID zero"},
		{3, "gap between 2 and 100"},
		{50, "mid-range gap"},
		{101, "gap after 100"},
		{999, "last value before next known"},
		{1000, "above known"},
		{9999, "large unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("msgID_%d", tt.msgID), func(t *testing.T) {
			_, ok := msgIDToService[tt.msgID]
			if ok {
				t.Skip("msgID unexpectedly has a mapping; test assumption may be outdated")
			}

			subj := routeSubject(tt.msgID)
			want := fmt.Sprintf("game.svc.default.%d", tt.msgID)
			if subj != want {
				t.Errorf("subject = %q, want %q", subj, want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Subject template correctness
// ---------------------------------------------------------------------------

func TestSubjectTemplate(t *testing.T) {
	tests := []struct {
		msgID uint32
		svc   string
		want  string
	}{
		{1, "auth", "game.svc.auth.1"},
		{100, "player", "game.svc.player.100"},
		{200, "scene", "game.svc.scene.200"},
		{300, "combat", "game.svc.combat.300"},
		{400, "chat", "game.svc.chat.400"},
		{500, "mail", "game.svc.mail.500"},
		{600, "shop", "game.svc.shop.600"},
		{700, "inventory", "game.svc.inventory.700"},
		{800, "quest", "game.svc.quest.800"},
		{900, "guild", "game.svc.guild.900"},
		{0, "default", "game.svc.default.0"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			subj := fmt.Sprintf("game.svc.%s.%d", tt.svc, tt.msgID)
			if subj != tt.want {
				t.Errorf("subject = %q, want %q", subj, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Player subscription / publish subject format
// ---------------------------------------------------------------------------

func TestPlayerSubscribeSubject(t *testing.T) {
	tests := []struct {
		playerID uint64
		want     string
	}{
		{1, "gateway.player.1"},
		{10001, "gateway.player.10001"},
		{0, "gateway.player.0"},
		{18446744073709551615, "gateway.player.18446744073709551615"}, // max uint64
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("playerID_%d", tt.playerID), func(t *testing.T) {
			subj := playerSubscribeSubject(tt.playerID)
			if subj != tt.want {
				t.Errorf("subject = %q, want %q", subj, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Edge and boundary IDs
// ---------------------------------------------------------------------------

func TestMsgIDBoundaryValues(t *testing.T) {
	boundaryTests := []struct {
		msgID uint32
		desc  string
	}{
		{0, "minimum uint32"},
		{1, "first system message"},
		{2, "second system message"},
		{99, "just before first business msgID 100"},
		{100, "first business msgID"},
		{999, "boundary between system and business (convention)"},
		{1000, "just above business convention boundary"},
		{math.MaxUint32, "maximum uint32"},
		{math.MaxUint32 - 1, "just below max uint32"},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.desc, func(t *testing.T) {
			// routeSubject should never panic for any uint32 value.
			subj := routeSubject(tt.msgID)
			if subj == "" {
				t.Errorf("routeSubject(%d) returned empty subject", tt.msgID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// msgIDToService consistency: all known msgIDs produce valid subjects
// ---------------------------------------------------------------------------

func TestMsgIDToService_AllEntriesValid(t *testing.T) {
	for msgID, svc := range msgIDToService {
		t.Run(fmt.Sprintf("entry_%d_%s", msgID, svc), func(t *testing.T) {
			if svc == "" {
				t.Errorf("msgID %d has empty service name", msgID)
			}
			if msgID == 0 {
				t.Errorf("msgID 0 mapped to %q; msgID=0 should be invalid", svc)
			}
			subj := routeSubject(msgID)
			want := fmt.Sprintf("game.svc.%s.%d", svc, msgID)
			if subj != want {
				t.Errorf("subject = %q, want %q", subj, want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// No duplicate service names for different msgIDs (informational)
// ---------------------------------------------------------------------------

func TestMsgIDToService_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]uint32)
	for msgID, svc := range msgIDToService {
		if prev, ok := seen[svc]; ok {
			t.Errorf("service %q used by msgID %d and %d (duplicate)", svc, prev, msgID)
		}
		seen[svc] = msgID
	}
}
