package model

import (
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// ScoreForRealm
// ============================================================================

func TestScoreForRealm(t *testing.T) {
	tests := []struct {
		name       string
		realmID    uint32
		realmLevel uint32
		want       float64
	}{
		{name: "凡人 realm 1 level 0", realmID: 1, realmLevel: 0, want: 10000},
		{name: "筑基 realm 3 level 5", realmID: 3, realmLevel: 5, want: 30005},
		{name: "金丹 realm 4 level 9", realmID: 4, realmLevel: 9, want: 40009},
		{name: "化神 realm 6 level 3", realmID: 6, realmLevel: 3, want: 60003},
		{name: "满级 realm 10 level 0", realmID: 10, realmLevel: 0, want: 100000},
		{name: "大数值", realmID: 9999, realmLevel: 9999, want: 99990000 + 9999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreForRealm(tt.realmID, tt.realmLevel)
			if got != tt.want {
				t.Errorf("ScoreForRealm(%d, %d) = %v; want %v", tt.realmID, tt.realmLevel, got, tt.want)
			}
		})
	}
}

// ============================================================================
// IsValidType
// ============================================================================

func TestIsValidType(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want bool
	}{
		{name: "realm", typ: string(RankingTypeRealm), want: true},
		{name: "combat_power", typ: string(RankingTypeCombatPower), want: true},
		{name: "wealth", typ: string(RankingTypeWealth), want: true},
		{name: "sect", typ: string(RankingTypeSect), want: true},
		{name: "empty", typ: "", want: false},
		{name: "invalid", typ: "invalid", want: false},
		{name: "spaces", typ: " realm ", want: false},
		{name: "uppercase", typ: "REALM", want: false},
		{name: "nil string", typ: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidType(tt.typ); got != tt.want {
				t.Errorf("IsValidType(%q) = %v; want %v", tt.typ, got, tt.want)
			}
		})
	}
}

// ============================================================================
// PageRequest
// ============================================================================

func TestPageRequest_Normalize(t *testing.T) {
	tests := []struct {
		name       string
		page       int32
		pageSize   int32
		wantPage   int32
		wantSize   int32
	}{
		{name: "zero page defaults to 1", page: 0, pageSize: 0, wantPage: 1, wantSize: 20},
		{name: "negative page defaults to 1", page: -1, pageSize: 10, wantPage: 1, wantSize: 10},
		{name: "normal values unchanged", page: 3, pageSize: 15, wantPage: 3, wantSize: 15},
		{name: "page size 0 defaults to 20", page: 1, pageSize: 0, wantPage: 1, wantSize: 20},
		{name: "page size -5 defaults to 20", page: 1, pageSize: -5, wantPage: 1, wantSize: 20},
		{name: "page size > 100 clamped to 20", page: 2, pageSize: 200, wantPage: 2, wantSize: 20},
		{name: "page size = 100 is valid", page: 1, pageSize: 100, wantPage: 1, wantSize: 100},
		{name: "page size = 1 is valid", page: 5, pageSize: 1, wantPage: 5, wantSize: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PageRequest{Page: tt.page, PageSize: tt.pageSize}
			p.Normalize()
			if p.Page != tt.wantPage {
				t.Errorf("Page = %d; want %d", p.Page, tt.wantPage)
			}
			if p.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d; want %d", p.PageSize, tt.wantSize)
			}
		})
	}
}

func TestPageRequest_Offset(t *testing.T) {
	tests := []struct {
		name     string
		page     int32
		pageSize int32
		want     int64
	}{
		{name: "page 1 size 20 offset 0", page: 1, pageSize: 20, want: 0},
		{name: "page 2 size 20 offset 20", page: 2, pageSize: 20, want: 20},
		{name: "page 5 size 10 offset 40", page: 5, pageSize: 10, want: 40},
		{name: "page 1 size 100 offset 0", page: 1, pageSize: 100, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PageRequest{Page: tt.page, PageSize: tt.pageSize}
			if got := p.Offset(); got != tt.want {
				t.Errorf("Offset() = %d; want %d", got, tt.want)
			}
		})
	}
}

func TestPageRequest_Count(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int32
		want     int64
	}{
		{name: "size 20 count 20", pageSize: 20, want: 20},
		{name: "size 1 count 1", pageSize: 1, want: 1},
		{name: "size 100 count 100", pageSize: 100, want: 100},
		{name: "size 0 count 0", pageSize: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PageRequest{Page: 1, PageSize: tt.pageSize}
			if got := p.Count(); got != tt.want {
				t.Errorf("Count() = %d; want %d", got, tt.want)
			}
		})
	}
}

// ============================================================================
// GetLeaderboard
// ============================================================================

func TestGetLeaderboard(t *testing.T) {
	tests := []struct {
		name  string
		typ   RankingType
		found bool
	}{
		{name: "realm", typ: RankingTypeRealm, found: true},
		{name: "combat_power", typ: RankingTypeCombatPower, found: true},
		{name: "wealth", typ: RankingTypeWealth, found: true},
		{name: "sect", typ: RankingTypeSect, found: true},
		{name: "unknown", typ: "unknown", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := GetLeaderboard(tt.typ)
			if tt.found {
				if lb == nil {
					t.Fatal("expected leaderboard, got nil")
				}
				if lb.RankingType != tt.typ {
					t.Errorf("RankingType = %v; want %v", lb.RankingType, tt.typ)
				}
			} else {
				if lb != nil {
					t.Errorf("expected nil for unknown type, got %+v", lb)
				}
			}
		})
	}
}

func TestLeaderboardDecaySettings(t *testing.T) {
	// realm: no decay
	lb := GetLeaderboard(RankingTypeRealm)
	if lb.DecayEnabled {
		t.Error("realm should NOT have decay enabled")
	}

	// combat: 3%/day after 7 days
	lb = GetLeaderboard(RankingTypeCombatPower)
	if !lb.DecayEnabled {
		t.Error("combat SHOULD have decay enabled")
	}
	if lb.DecayRate != 0.03 {
		t.Errorf("combat DecayRate = %v; want 0.03", lb.DecayRate)
	}
	if lb.DecayAfterDays != 7 {
		t.Errorf("combat DecayAfterDays = %d; want 7", lb.DecayAfterDays)
	}

	// wealth: 2%/day after 14 days
	lb = GetLeaderboard(RankingTypeWealth)
	if !lb.DecayEnabled {
		t.Error("wealth SHOULD have decay enabled")
	}
	if lb.DecayRate != 0.02 {
		t.Errorf("wealth DecayRate = %v; want 0.02", lb.DecayRate)
	}
	if lb.DecayAfterDays != 14 {
		t.Errorf("wealth DecayAfterDays = %d; want 14", lb.DecayAfterDays)
	}

	// sect: no decay
	lb = GetLeaderboard(RankingTypeSect)
	if lb.DecayEnabled {
		t.Error("sect should NOT have decay enabled")
	}
}

// ============================================================================
// NewEntry
// ============================================================================

func TestNewEntry(t *testing.T) {
	entry := NewEntry(42, "张三", "筑基三层", 30005.0, 1)

	if entry.PlayerID != 42 {
		t.Errorf("PlayerID = %d; want 42", entry.PlayerID)
	}
	if entry.Nickname != "张三" {
		t.Errorf("Nickname = %s; want 张三", entry.Nickname)
	}
	if entry.RealmName != "筑基三层" {
		t.Errorf("RealmName = %s; want 筑基三层", entry.RealmName)
	}
	if entry.Score != 30005.0 {
		t.Errorf("Score = %v; want 30005.0", entry.Score)
	}
	if entry.Rank != 1 {
		t.Errorf("Rank = %d; want 1", entry.Rank)
	}
	if entry.UpdatedAt == 0 {
		t.Error("UpdatedAt should be set (non-zero)")
	}
}

// ============================================================================
// Redis key templates
// ============================================================================

func TestRedisKeyFormats(t *testing.T) {
	tests := []struct {
		template string
		typ      RankingType
		want     string
	}{
		{template: RedisKeyScoreZSet, typ: RankingTypeRealm, want: "ranking:zset:realm"},
		{template: RedisKeyPlayerInfo, typ: RankingTypeCombatPower, want: "ranking:info:combat_power"},
		{template: RedisKeySnapshot, typ: RankingTypeWealth, want: "ranking:snapshot:wealth"},
		{template: RedisKeyLastActivity, typ: RankingTypeSect, want: "ranking:last_active:sect"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := key(tt.template, tt.typ)
			if got != tt.want {
				t.Errorf("got %q; want %q", got, tt.want)
			}
		})
	}
}

// key is a helper to format a redis key template.
func key(tmpl string, typ RankingType) string {
	return fmt.Sprintf(tmpl, string(typ))
}

// ============================================================================
// Snapshot
// ============================================================================

func TestSnapshotStructure(t *testing.T) {
	now := time.Now()
	entries := []*RankingEntry{
		{PlayerID: 1, Score: 100, Rank: 1},
		{PlayerID: 2, Score: 90, Rank: 2},
	}
	snap := &Snapshot{
		Entries:     entries,
		RefreshedAt: now,
	}

	if len(snap.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(snap.Entries))
	}
	if snap.RefreshedAt != now {
		t.Errorf("RefreshedAt mismatch")
	}
	if snap.Entries[0].PlayerID != 1 {
		t.Errorf("first entry PlayerID = %d; want 1", snap.Entries[0].PlayerID)
	}
}

// ============================================================================
// DefaultLeaderboards consistency
// ============================================================================

func TestDefaultLeaderboardsCount(t *testing.T) {
	if len(DefaultLeaderboards) != 4 {
		t.Errorf("expected 4 leaderboards, got %d", len(DefaultLeaderboards))
	}
}

func TestRankingTypeNames(t *testing.T) {
	for _, lb := range DefaultLeaderboards {
		name, ok := RankingTypeNames[lb.RankingType]
		if !ok {
			t.Errorf("no Chinese name for %s", lb.RankingType)
		}
		if name != lb.Name {
			t.Errorf("RankingTypeNames[%s] = %q; want %q", lb.RankingType, name, lb.Name)
		}
	}
}
