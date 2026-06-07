package service

import (
	"context"
	"sync"
	"testing"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ---------- mock implementations ----------

type mockPlayerRepo struct {
	mu      sync.Mutex
	players map[int64]*model.Player
	byName  map[string]*model.Player
	byUser  map[string]*model.Player
	nextID  int64
}

func newMockPlayerRepo() *mockPlayerRepo {
	return &mockPlayerRepo{
		players: make(map[int64]*model.Player),
		byName:  make(map[string]*model.Player),
		byUser:  make(map[string]*model.Player),
		nextID:  1,
	}
}

func (r *mockPlayerRepo) Create(p *model.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nextID
	r.nextID++
	r.players[p.ID] = p
	r.byName[p.Name] = p
	r.byUser[p.UserID] = p
	return nil
}

func (r *mockPlayerRepo) GetByID(id int64) (*model.Player, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (r *mockPlayerRepo) GetByUserID(userID string) (*model.Player, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUser[userID]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (r *mockPlayerRepo) GetByName(name string) (*model.Player, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byName[name]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (r *mockPlayerRepo) Update(p *model.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.players[p.ID]; !ok {
		return errMockNotFound
	}
	r.players[p.ID] = p
	r.byName[p.Name] = p
	return nil
}

func (r *mockPlayerRepo) UpdateCurrency(playerID int64, gold, boundGold, jade int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[playerID]
	if !ok {
		return errMockNotFound
	}
	p.Gold = gold
	p.BoundGold = boundGold
	p.Jade = jade
	return nil
}

func (r *mockPlayerRepo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.players, id)
	return nil
}

type mockCache struct {
	mu      sync.Mutex
	players map[int64]*model.PlayerCache
}

func newMockCache() *mockCache {
	return &mockCache{
		players: make(map[int64]*model.PlayerCache),
	}
}

func (c *mockCache) SetPlayer(ctx context.Context, p *model.PlayerCache) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.players[p.ID] = p
	return nil
}

func (c *mockCache) GetPlayer(ctx context.Context, playerID int64) (*model.PlayerCache, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.players[playerID]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (c *mockCache) DelPlayer(ctx context.Context, playerID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.players, playerID)
	return nil
}

func (c *mockCache) RefreshTTL(ctx context.Context, playerID int64) error {
	return nil
}

func (c *mockCache) SetInventoryCache(ctx context.Context, playerID int64, items []*model.InventoryItem) error {
	return nil
}

func (c *mockCache) GetInventoryCache(ctx context.Context, playerID int64) ([]*model.InventoryItem, error) {
	return nil, nil
}

// ---------- helpers ----------

var errMockNotFound = &mockErr{"not found"}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

func newPlayerService() (*PlayerService, *mockPlayerRepo, *mockCache) {
	mr := newMockPlayerRepo()
	mc := newMockCache()
	svc := &PlayerService{
		playerRepo: mr,
		cache:      mc,
		log:        zap.NewNop(),
	}
	return svc, mr, mc
}

func newPlayerServiceWithLogger(logger *zap.Logger) (*PlayerService, *mockPlayerRepo, *mockCache) {
	mr := newMockPlayerRepo()
	mc := newMockCache()
	svc := &PlayerService{
		playerRepo: mr,
		cache:      mc,
		log:        logger,
	}
	return svc, mr, mc
}

func ptrPlayer(p model.Player) *model.Player {
	return &p
}

func ptrCache(c model.PlayerCache) *model.PlayerCache {
	return &c
}

const ctxKey = "test"

var testCtx = context.WithValue(context.Background(), ctxKey, "test")

// ---------- CreatePlayer tests ----------

func TestCreatePlayer_Success(t *testing.T) {
	svc, _, _ := newPlayerService()

	req := &model.CreatePlayerRequest{
		UserID:     "user_001",
		Name:       "金灵根修士",
		SpiritRoot: model.SpiritRootMetal,
	}

	player, err := svc.CreatePlayer(testCtx, req)
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	if player.ID == 0 {
		t.Error("expected player ID to be set")
	}
	if player.Name != "金灵根修士" {
		t.Errorf("Name = %s, want 金灵根修士", player.Name)
	}
	if player.Level != 1 {
		t.Errorf("Level = %d, want 1", player.Level)
	}
	if player.Realm != model.RealmMortal {
		t.Errorf("Realm = %d, want %d", player.Realm, model.RealmMortal)
	}
	if player.Gold != 100 {
		t.Errorf("Gold = %d, want 100", player.Gold)
	}
	// Metal: attack +5
	if player.Attack != 15 {
		t.Errorf("Attack = %d, want 15 (10 base + 5 metal)", player.Attack)
	}
	if player.Defense != 5 {
		t.Errorf("Defense = %d, want 5", player.Defense)
	}
	if player.MaxHP != 100 {
		t.Errorf("MaxHP = %d, want 100", player.MaxHP)
	}
	if player.MaxMP != 50 {
		t.Errorf("MaxMP = %d, want 50", player.MaxMP)
	}
}

func TestCreatePlayer_DuplicateName(t *testing.T) {
	svc, mr, _ := newPlayerService()

	// Pre-register a player with the same name
	existing := &model.Player{
		UserID: "user_other",
		Name:   "duplicate_name",
	}
	_ = mr.Create(existing)

	req := &model.CreatePlayerRequest{
		UserID:     "user_002",
		Name:       "duplicate_name",
		SpiritRoot: model.SpiritRootWater,
	}

	_, err := svc.CreatePlayer(testCtx, req)
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
	if err.Error() != "角色名 duplicate_name 已被使用" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Verify only one player exists with that name
	count := 0
	mr.mu.Lock()
	for _, p := range mr.players {
		if p.Name == "duplicate_name" {
			count++
		}
	}
	mr.mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 player with duplicate_name, got %d", count)
	}
}

func TestCreatePlayer_AllSpiritRoots(t *testing.T) {
	cases := []struct {
		name       string
		spiritRoot int32
		wantHP     int64
		wantMP     int64
		wantAtk    int64
		wantDef    int64
	}{
		{"Metal", model.SpiritRootMetal, 100, 50, 15, 5},
		{"Wood", model.SpiritRootWood, 130, 50, 10, 5},
		{"Water", model.SpiritRootWater, 100, 80, 10, 5},
		{"Fire", model.SpiritRootFire, 100, 50, 13, 3},
		{"Earth", model.SpiritRootEarth, 100, 50, 10, 10},
		{"Wind", model.SpiritRootWind, 100, 65, 14, 5},
		{"Thunder", model.SpiritRootThunder, 100, 50, 16, 7},
		{"Ice", model.SpiritRootIce, 115, 50, 10, 9},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newPlayerService()
			req := &model.CreatePlayerRequest{
				UserID:     "user_" + tc.name,
				Name:       tc.name,
				SpiritRoot: tc.spiritRoot,
			}
			p, err := svc.CreatePlayer(testCtx, req)
			if err != nil {
				t.Fatalf("CreatePlayer failed for %s: %v", tc.name, err)
			}
			if p.MaxHP != tc.wantHP {
				t.Errorf("MaxHP = %d, want %d", p.MaxHP, tc.wantHP)
			}
			if p.MaxMP != tc.wantMP {
				t.Errorf("MaxMP = %d, want %d", p.MaxMP, tc.wantMP)
			}
			if p.Attack != tc.wantAtk {
				t.Errorf("Attack = %d, want %d", p.Attack, tc.wantAtk)
			}
			if p.Defense != tc.wantDef {
				t.Errorf("Defense = %d, want %d", p.Defense, tc.wantDef)
			}
		})
	}
}

func TestCreatePlayer_SpiritRootNone(t *testing.T) {
	svc, _, _ := newPlayerService()

	req := &model.CreatePlayerRequest{
		UserID:     "user_none",
		Name:       "无灵根",
		SpiritRoot: model.SpiritRootNone,
	}

	p, err := svc.CreatePlayer(testCtx, req)
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	// SpiritRootNone uses base attributes only
	if p.MaxHP != 100 {
		t.Errorf("MaxHP = %d, want 100", p.MaxHP)
	}
	if p.MaxMP != 50 {
		t.Errorf("MaxMP = %d, want 50", p.MaxMP)
	}
	if p.Attack != 10 {
		t.Errorf("Attack = %d, want 10", p.Attack)
	}
	if p.Defense != 5 {
		t.Errorf("Defense = %d, want 5", p.Defense)
	}
}

// ---------- GetPlayer tests ----------

func TestGetPlayer_FoundFromCache(t *testing.T) {
	svc, mr, mc := newPlayerService()

	// Create player in repo
	player := &model.Player{
		ID:     100,
		UserID: "user_get",
		Name:   "cache_player",
		Level:  3,
		Realm:  model.RealmQiRef,
		HP:     120,
		MaxHP:  120,
		Attack: 12,
	}
	_ = mr.Create(player)

	// Pre-populate cache
	_ = mc.SetPlayer(testCtx, player.ToCache())

	got, err := svc.GetPlayer(testCtx, 1)
	if err != nil {
		t.Fatalf("GetPlayer failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPlayer returned nil")
	}
	if got.Name != "cache_player" {
		t.Errorf("Name = %s, want cache_player", got.Name)
	}
	if got.Level != 3 {
		t.Errorf("Level = %d, want 3", got.Level)
	}
	if got.HP != 120 {
		t.Errorf("HP = %d, want 120", got.HP)
	}
}

func TestGetPlayer_FoundFromDB(t *testing.T) {
	svc, mr, _ := newPlayerService()

	// Create player only in repo (not in cache)
	player := &model.Player{
		ID:     200,
		UserID: "user_db",
		Name:   "db_player",
		Level:  5,
		Realm:  model.RealmBase,
		Attack: 20,
	}
	_ = mr.Create(player)

	got, err := svc.GetPlayer(testCtx, 1)
	if err != nil {
		t.Fatalf("GetPlayer failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPlayer returned nil")
	}
	if got.Name != "db_player" {
		t.Errorf("Name = %s, want db_player", got.Name)
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	svc, _, _ := newPlayerService()

	_, err := svc.GetPlayer(testCtx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent player, got nil")
	}
}

// ---------- GetPlayerByUserID tests ----------

func TestGetPlayerByUserID_Found(t *testing.T) {
	svc, mr, _ := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID: "uid_test",
		Name:   "userid_player",
	})

	got, err := svc.GetPlayerByUserID(testCtx, "uid_test")
	if err != nil {
		t.Fatalf("GetPlayerByUserID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPlayerByUserID returned nil")
	}
	if got.Name != "userid_player" {
		t.Errorf("Name = %s, want userid_player", got.Name)
	}
}

func TestGetPlayerByUserID_NotFound(t *testing.T) {
	svc, _, _ := newPlayerService()

	_, err := svc.GetPlayerByUserID(testCtx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

// ---------- AddExp tests ----------

func TestAddExp_NormalGain(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID:     "exp_user",
		Name:       "exp_player",
		Experience: 100,
	}
	_ = mr.Create(player)

	updated, err := svc.AddExp(testCtx, 1, 50)
	if err != nil {
		t.Fatalf("AddExp failed: %v", err)
	}
	if updated.Experience != 150 {
		t.Errorf("Experience = %d, want 150", updated.Experience)
	}
}

func TestAddExp_NegativeToZero(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID:     "exp_neg",
		Name:       "neg_player",
		Experience: 30,
	}
	_ = mr.Create(player)

	updated, err := svc.AddExp(testCtx, 1, -50)
	if err != nil {
		t.Fatalf("AddExp failed: %v", err)
	}
	// Should clamp to 0
	if updated.Experience != 0 {
		t.Errorf("Experience = %d, want 0 (clamped)", updated.Experience)
	}
}

func TestAddExp_LargeGain(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID:     "exp_large",
		Name:       "large_player",
		Experience: 0,
	}
	_ = mr.Create(player)

	updated, err := svc.AddExp(testCtx, 1, 999999)
	if err != nil {
		t.Fatalf("AddExp failed: %v", err)
	}
	if updated.Experience != 999999 {
		t.Errorf("Experience = %d, want 999999", updated.Experience)
	}
}

func TestAddExp_PlayerNotFound(t *testing.T) {
	svc, _, _ := newPlayerService()

	_, err := svc.AddExp(testCtx, 999, 50)
	if err == nil {
		t.Fatal("expected error for non-existent player, got nil")
	}
}

// ---------- UpdateCurrency tests ----------

func TestUpdateCurrency_AddGold(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID: "gold_user",
		Name:   "gold_player",
		Gold:   100,
	}
	_ = mr.Create(player)

	updated, err := svc.UpdateCurrency(testCtx, 1, &model.CurrencyChangeRequest{Gold: 50})
	if err != nil {
		t.Fatalf("UpdateCurrency failed: %v", err)
	}
	if updated.Gold != 150 {
		t.Errorf("Gold = %d, want 150", updated.Gold)
	}
}

func TestUpdateCurrency_DeductGold(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID: "gold_deduct",
		Name:   "deduct_player",
		Gold:   100,
	}
	_ = mr.Create(player)

	updated, err := svc.UpdateCurrency(testCtx, 1, &model.CurrencyChangeRequest{Gold: -30})
	if err != nil {
		t.Fatalf("UpdateCurrency failed: %v", err)
	}
	if updated.Gold != 70 {
		t.Errorf("Gold = %d, want 70", updated.Gold)
	}
}

func TestUpdateCurrency_InsufficientGold(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID: "gold_insuff",
		Name:   "insuff_player",
		Gold:   10,
	}
	_ = mr.Create(player)

	_, err := svc.UpdateCurrency(testCtx, 1, &model.CurrencyChangeRequest{Gold: -100})
	if err == nil {
		t.Fatal("expected insufficient gold error, got nil")
	}
}

func TestUpdateCurrency_BoundGoldAndJade(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID:    "multi_cur",
		Name:      "multi_cur_player",
		Gold:      100,
		BoundGold: 50,
		Jade:      10,
	}
	_ = mr.Create(player)

	updated, err := svc.UpdateCurrency(testCtx, 1, &model.CurrencyChangeRequest{
		Gold:      20,
		BoundGold: -10,
		Jade:      5,
	})
	if err != nil {
		t.Fatalf("UpdateCurrency failed: %v", err)
	}
	if updated.Gold != 120 {
		t.Errorf("Gold = %d, want 120", updated.Gold)
	}
	if updated.BoundGold != 40 {
		t.Errorf("BoundGold = %d, want 40", updated.BoundGold)
	}
	if updated.Jade != 15 {
		t.Errorf("Jade = %d, want 15", updated.Jade)
	}
}

// ---------- UpdateRealm tests ----------

func TestUpdateRealm(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID: "realm_user",
		Name:   "realm_player",
		Realm:  model.RealmMortal,
		Level:  1,
		HP:     80,
		MaxHP:  100,
		Attack: 10,
		Defense: 5,
	}
	_ = mr.Create(player)

	err := svc.UpdateRealm(testCtx, 1, model.RealmBase, 10, 50, 30, 500)
	if err != nil {
		t.Fatalf("UpdateRealm failed: %v", err)
	}

	updated, _ := mr.GetByID(1)
	if updated.Realm != model.RealmBase {
		t.Errorf("Realm = %d, want %d", updated.Realm, model.RealmBase)
	}
	if updated.Level != 10 {
		t.Errorf("Level = %d, want 10", updated.Level)
	}
	if updated.Attack != 50 {
		t.Errorf("Attack = %d, want 50", updated.Attack)
	}
	if updated.Defense != 30 {
		t.Errorf("Defense = %d, want 30", updated.Defense)
	}
	if updated.MaxHP != 500 {
		t.Errorf("MaxHP = %d, want 500", updated.MaxHP)
	}
	// HP should be unchanged since 80 < 500
	if updated.HP != 80 {
		t.Errorf("HP = %d, want 80 (unchanged)", updated.HP)
	}
}

func TestUpdateRealm_ClampsHP(t *testing.T) {
	svc, mr, _ := newPlayerService()

	player := &model.Player{
		UserID: "realm_clamp",
		Name:   "clamp_player",
		Realm:  model.RealmMortal,
		HP:     900,
		MaxHP:  1000,
	}
	_ = mr.Create(player)

	err := svc.UpdateRealm(testCtx, 1, model.RealmQiRef, 2, 20, 10, 200)
	if err != nil {
		t.Fatalf("UpdateRealm failed: %v", err)
	}

	updated, _ := mr.GetByID(1)
	if updated.MaxHP != 200 {
		t.Errorf("MaxHP = %d, want 200", updated.MaxHP)
	}
	// HP should be clamped to new MaxHP
	if updated.HP != 200 {
		t.Errorf("HP = %d, want 200 (clamped)", updated.HP)
	}
}

// ---------- GetPlayerWithDetails tests ----------

func TestGetPlayerWithDetails(t *testing.T) {
	svc, mr, _ := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID:    "detail_usr",
		Name:      "detail_player",
		Realm:     model.RealmGolden,
		SpiritRoot: model.SpiritRootFire,
	})

	resp, err := svc.GetPlayerWithDetails(testCtx, 1)
	if err != nil {
		t.Fatalf("GetPlayerWithDetails failed: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.RealmName != "金丹" {
		t.Errorf("RealmName = %s, want 金丹", resp.RealmName)
	}
	if resp.SpiritName != "火灵根" {
		t.Errorf("SpiritName = %s, want 火灵根", resp.SpiritName)
	}
}

func TestGetPlayerWithDetails_UnknownRealm(t *testing.T) {
	svc, mr, _ := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID:     "unknown_realm",
		Name:       "unknown_realm_player",
		Realm:      999,
		SpiritRoot: model.SpiritRootNone,
	})

	resp, err := svc.GetPlayerWithDetails(testCtx, 1)
	if err != nil {
		t.Fatalf("GetPlayerWithDetails failed: %v", err)
	}
	if resp.RealmName != "未知" {
		t.Errorf("RealmName = %s, want 未知", resp.RealmName)
	}
	if resp.SpiritName != "无灵根" {
		t.Errorf("SpiritName = %s, want 无灵根", resp.SpiritName)
	}
}

// ---------- UpdatePlayer tests ----------

func TestUpdatePlayer(t *testing.T) {
	svc, mr, mc := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID: "upd_usr",
		Name:   "upd_player",
		HP:     100,
		Attack: 10,
	})

	player, _ := mr.GetByID(1)
	player.HP = 200
	player.Attack = 99

	err := svc.UpdatePlayer(testCtx, player)
	if err != nil {
		t.Fatalf("UpdatePlayer failed: %v", err)
	}

	// Verify DB update
	updated, _ := mr.GetByID(1)
	if updated.HP != 200 {
		t.Errorf("HP = %d, want 200", updated.HP)
	}
	if updated.Attack != 99 {
		t.Errorf("Attack = %d, want 99", updated.Attack)
	}

	// Verify cache update
	cached, _ := mc.GetPlayer(testCtx, 1)
	if cached == nil {
		t.Fatal("player not in cache after update")
	}
	if cached.HP != 200 {
		t.Errorf("cached HP = %d, want 200", cached.HP)
	}
}

// ---------- AddRewards tests ----------

func TestAddRewards_ExpAndGold(t *testing.T) {
	svc, mr, _ := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID:     "reward_usr",
		Name:       "reward_player",
		Experience: 0,
		Gold:       0,
	})

	err := svc.AddRewards(testCtx, 1, 500, 200, nil)
	if err != nil {
		t.Fatalf("AddRewards failed: %v", err)
	}

	player, _ := mr.GetByID(1)
	if player.Experience != 500 {
		t.Errorf("Experience = %d, want 500", player.Experience)
	}
	if player.Gold != 200 {
		t.Errorf("Gold = %d, want 200", player.Gold)
	}
}

func TestAddRewards_ZeroValues(t *testing.T) {
	svc, mr, _ := newPlayerService()

	_ = mr.Create(&model.Player{
		UserID:     "zero_reward",
		Name:       "zero_reward_player",
		Experience: 100,
		Gold:       100,
	})

	err := svc.AddRewards(testCtx, 1, 0, 0, nil)
	if err != nil {
		t.Fatalf("AddRewards failed: %v", err)
	}

	// Values should be unchanged
	player, _ := mr.GetByID(1)
	if player.Experience != 100 {
		t.Errorf("Experience = %d, want 100", player.Experience)
	}
	if player.Gold != 100 {
		t.Errorf("Gold = %d, want 100", player.Gold)
	}
}

// ---------- calcInitAttributes tests (pure logic) ----------

func TestCalcInitAttributes(t *testing.T) {
	svc, _, _ := newPlayerService()

	cases := []struct {
		name       string
		spiritRoot int32
		wantHP     int64
		wantMP     int64
		wantAtk    int64
		wantDef    int64
	}{
		{"default (none)", model.SpiritRootNone, 100, 50, 10, 5},
		{"metal", model.SpiritRootMetal, 100, 50, 15, 5},
		{"wood", model.SpiritRootWood, 130, 50, 10, 5},
		{"water", model.SpiritRootWater, 100, 80, 10, 5},
		{"fire", model.SpiritRootFire, 100, 50, 13, 3},
		{"earth", model.SpiritRootEarth, 100, 50, 10, 10},
		{"wind", model.SpiritRootWind, 100, 65, 14, 5},
		{"thunder", model.SpiritRootThunder, 100, 50, 16, 7},
		{"ice", model.SpiritRootIce, 115, 50, 10, 9},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := svc.calcInitAttributes(tc.spiritRoot)
			if attrs.hp != tc.wantHP {
				t.Errorf("hp = %d, want %d", attrs.hp, tc.wantHP)
			}
			if attrs.mp != tc.wantMP {
				t.Errorf("mp = %d, want %d", attrs.mp, tc.wantMP)
			}
			if attrs.attack != tc.wantAtk {
				t.Errorf("attack = %d, want %d", attrs.attack, tc.wantAtk)
			}
			if attrs.defense != tc.wantDef {
				t.Errorf("defense = %d, want %d", attrs.defense, tc.wantDef)
			}
		})
	}
}
