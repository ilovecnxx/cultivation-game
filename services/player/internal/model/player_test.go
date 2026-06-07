package model

import (
	"testing"
)

func TestPlayerToCache(t *testing.T) {
	p := &Player{
		ID:          1001,
		UserID:      "user_abc",
		Name:        "test_player",
		Level:       5,
		Realm:       RealmGolden,
		SpiritRoot:  SpiritRootFire,
		HP:          150,
		MaxHP:       200,
		MP:          80,
		MaxMP:       100,
		Attack:      25,
		Defense:     12,
		SpiritPower: 500,
		Experience:  1200,
		Gold:        1000,
		BoundGold:   50,
		Jade:        10,
	}

	cache := p.ToCache()
	if cache == nil {
		t.Fatal("ToCache() returned nil")
	}

	cases := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", cache.ID, int64(1001)},
		{"UserID", cache.UserID, "user_abc"},
		{"Name", cache.Name, "test_player"},
		{"Level", cache.Level, int32(5)},
		{"Realm", cache.Realm, int32(RealmGolden)},
		{"SpiritRoot", cache.SpiritRoot, int32(SpiritRootFire)},
		{"HP", cache.HP, int64(150)},
		{"MaxHP", cache.MaxHP, int64(200)},
		{"MP", cache.MP, int64(80)},
		{"MaxMP", cache.MaxMP, int64(100)},
		{"Attack", cache.Attack, int64(25)},
		{"Defense", cache.Defense, int64(12)},
		{"SpiritPower", cache.SpiritPower, int64(500)},
		{"Experience", cache.Experience, int64(1200)},
		{"Gold", cache.Gold, int64(1000)},
		{"BoundGold", cache.BoundGold, int64(50)},
		{"Jade", cache.Jade, int64(10)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("ToCache().%s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestPlayerFromCache(t *testing.T) {
	original := &Player{
		ID:          2002,
		UserID:      "user_xyz",
		Name:        "from_cache_player",
		Level:       10,
		Realm:       RealmNascent,
		SpiritRoot:  SpiritRootThunder,
		HP:          500,
		MaxHP:       600,
		MP:          300,
		MaxMP:       400,
		Attack:      80,
		Defense:     40,
		SpiritPower: 2000,
		Experience:  5000,
		Gold:        9999,
		BoundGold:   100,
		Jade:        50,
	}

	cache := original.ToCache()

	restored := &Player{}
	restored.FromCache(cache)

	cases := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", restored.ID, int64(2002)},
		{"UserID", restored.UserID, "user_xyz"},
		{"Name", restored.Name, "from_cache_player"},
		{"Level", restored.Level, int32(10)},
		{"Realm", restored.Realm, int32(RealmNascent)},
		{"SpiritRoot", restored.SpiritRoot, int32(SpiritRootThunder)},
		{"HP", restored.HP, int64(500)},
		{"MaxHP", restored.MaxHP, int64(600)},
		{"MP", restored.MP, int64(300)},
		{"MaxMP", restored.MaxMP, int64(400)},
		{"Attack", restored.Attack, int64(80)},
		{"Defense", restored.Defense, int64(40)},
		{"SpiritPower", restored.SpiritPower, int64(2000)},
		{"Experience", restored.Experience, int64(5000)},
		{"Gold", restored.Gold, int64(9999)},
		{"BoundGold", restored.BoundGold, int64(100)},
		{"Jade", restored.Jade, int64(50)},
		{"CreatedAt remained zero", restored.CreatedAt.IsZero(), true},
		{"UpdatedAt remained zero", restored.UpdatedAt.IsZero(), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("FromCache().%s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestPlayerToCacheFromCacheRoundTrip(t *testing.T) {
	p := &Player{
		ID:          1,
		UserID:      "roundtrip",
		Name:        "roundtrip_player",
		Level:       3,
		Realm:       RealmQiRef,
		SpiritRoot:  SpiritRootWood,
		HP:          140,
		MaxHP:       140,
		MP:          60,
		MaxMP:       60,
		Attack:      12,
		Defense:     6,
		SpiritPower: 100,
		Experience:  250,
		Gold:        500,
		BoundGold:   0,
		Jade:        0,
	}

	cache := p.ToCache()
	restored := &Player{}
	restored.FromCache(cache)

	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", restored.ID, p.ID},
		{"Name", restored.Name, p.Name},
		{"HP", restored.HP, p.HP},
		{"Attack", restored.Attack, p.Attack},
		{"Defense", restored.Defense, p.Defense},
		{"Experience", restored.Experience, p.Experience},
	}

	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("round trip %s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestRealmNamesAllPresent(t *testing.T) {
	cases := []struct {
		realm    int32
		name     string
		expected string
	}{
		{RealmMortal, "RealmMortal", "凡人"},
		{RealmQiRef, "RealmQiRef", "练气"},
		{RealmBase, "RealmBase", "筑基"},
		{RealmGolden, "RealmGolden", "金丹"},
		{RealmNascent, "RealmNascent", "元婴"},
		{RealmSpirit, "RealmSpirit", "化神"},
		{RealmMerge, "RealmMerge", "合体"},
		{RealmAscend, "RealmAscend", "大乘"},
		{RealmTrib, "RealmTrib", "渡劫"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RealmNames[tc.realm]
			if !ok {
				t.Fatalf("RealmNames missing entry for %s (%d)", tc.name, tc.realm)
			}
			if got != tc.expected {
				t.Errorf("RealmNames[%d] = %q, want %q", tc.realm, got, tc.expected)
			}
		})
	}
}

func TestSpiritRootNamesAllPresent(t *testing.T) {
	cases := []struct {
		root     int32
		name     string
		expected string
	}{
		{SpiritRootNone, "SpiritRootNone", "无灵根"},
		{SpiritRootMetal, "SpiritRootMetal", "金灵根"},
		{SpiritRootWood, "SpiritRootWood", "木灵根"},
		{SpiritRootWater, "SpiritRootWater", "水灵根"},
		{SpiritRootFire, "SpiritRootFire", "火灵根"},
		{SpiritRootEarth, "SpiritRootEarth", "土灵根"},
		{SpiritRootWind, "SpiritRootWind", "风灵根"},
		{SpiritRootThunder, "SpiritRootThunder", "雷灵根"},
		{SpiritRootIce, "SpiritRootIce", "冰灵根"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SpiritRootNames[tc.root]
			if !ok {
				t.Fatalf("SpiritRootNames missing entry for %s (%d)", tc.name, tc.root)
			}
			if got != tc.expected {
				t.Errorf("SpiritRootNames[%d] = %q, want %q", tc.root, got, tc.expected)
			}
		})
	}
}

func TestRealmNamesUnknown(t *testing.T) {
	_, ok := RealmNames[999]
	if ok {
		t.Error("RealmNames should not contain entry for unknown realm 999")
	}
}

func TestSpiritRootNamesUnknown(t *testing.T) {
	_, ok := SpiritRootNames[999]
	if ok {
		t.Error("SpiritRootNames should not contain entry for unknown root 999")
	}
}
