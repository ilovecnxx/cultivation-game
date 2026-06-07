// Package service 提供世界服务的业务逻辑
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"

	"github.com/redis/go-redis/v9"
)

// 行动力常量
const (
	ActionPointMax    = 100                // 行动力上限
	ActionPointCost   = 10                 // 每次移动消耗行动力
	ActionPointRegen  = 20                 // 每小时恢复行动力
	APRegenInterval   = 1 * time.Hour      // 恢复间隔
	ExploreAPCost     = 5                  // 探索消耗行动力
	GatherAPCost      = 10                 // 采集消耗行动力
	DefaultRegionID   = "newbie_village_01" // 默认出生区域
)

// PlayerState 玩家在游戏世界中的状态(内存存储)
type PlayerState struct {
	UserID            string    `json:"user_id"`
	RegionID          string    `json:"region_id"`
	DiscoveredRegions []string  `json:"discovered_regions"`
	ActionPoints      int       `json:"action_points"`
	LastMoveAt        time.Time `json:"last_move_at"`
	LastAPUpdate      time.Time `json:"last_ap_update"`
}

// ExploreResult 探索结果
type ExploreResult struct {
	EventType string      `json:"event_type"` // encounter / monster / resource / nothing
	Message   string      `json:"message"`
	Encounter *model.Encounter `json:"encounter,omitempty"`
	Resources []ResourceDrop   `json:"resources,omitempty"`
}

// ResourceDrop 资源掉落
type ResourceDrop struct {
	ItemID   string `json:"item_id"`
	ItemName string `json:"item_name"`
	Amount   int64  `json:"amount"`
}

// MoveResult 移动结果
type MoveResult struct {
	Success         bool     `json:"success"`
	Message         string   `json:"message"`
	CurrentRegion   *model.MapRegion `json:"current_region,omitempty"`
	DiscoveredNew   bool     `json:"discovered_new"`
}

// playerStateStore 玩家状态存储接口，支持 Redis + 文件回退。
type playerStateStore interface {
	Load(userID string) (*PlayerState, error)
	Save(userID string, state *PlayerState) error
	Delete(userID string) error
	Ping() error
}

// redisPlayerStateStore 基于 Redis 的玩家状态存储。
type redisPlayerStateStore struct {
	rdb *redis.Client
	key string // Redis key prefix
}

func newRedisPlayerStateStore(rdb *redis.Client) *redisPlayerStateStore {
	return &redisPlayerStateStore{
		rdb: rdb,
		key: "world:player_state:",
	}
}

func (s *redisPlayerStateStore) Load(userID string) (*PlayerState, error) {
	data, err := s.rdb.HGetAll(context.Background(), s.key+userID).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	state := &PlayerState{}
	if v, ok := data["user_id"]; ok {
		state.UserID = v
	}
	if v, ok := data["region_id"]; ok {
		state.RegionID = v
	}
	if v, ok := data["discovered_regions"]; ok && v != "" {
		json.Unmarshal([]byte(v), &state.DiscoveredRegions)
	}
	if v, ok := data["action_points"]; ok && v != "" {
		fmt.Sscanf(v, "%d", &state.ActionPoints)
	}
	if v, ok := data["last_move_at"]; ok && v != "" {
		state.LastMoveAt, _ = time.Parse(time.RFC3339, v)
		_ = state.LastMoveAt // 忽略解析错误，使用零值
	}
	if v, ok := data["last_ap_update"]; ok && v != "" {
		state.LastAPUpdate, _ = time.Parse(time.RFC3339, v)
	}
	if state.LastMoveAt.IsZero() {
		state.LastMoveAt = time.Now()
	}
	if state.LastAPUpdate.IsZero() {
		state.LastAPUpdate = time.Now()
	}
	return state, nil
}

func (s *redisPlayerStateStore) Save(userID string, state *PlayerState) error {
	discBytes, _ := json.Marshal(state.DiscoveredRegions)
	data := map[string]interface{}{
		"user_id":            state.UserID,
		"region_id":          state.RegionID,
		"discovered_regions": string(discBytes),
		"action_points":      state.ActionPoints,
		"last_move_at":       state.LastMoveAt.Format(time.RFC3339),
		"last_ap_update":     state.LastAPUpdate.Format(time.RFC3339),
	}
	return s.rdb.HSet(context.Background(), s.key+userID, data).Err()
}

func (s *redisPlayerStateStore) Delete(userID string) error {
	return s.rdb.Del(context.Background(), s.key+userID).Err()
}

func (s *redisPlayerStateStore) Ping() error {
	return s.rdb.Ping(context.Background()).Err()
}

// filePlayerStateStore 基于文件的玩家状态存储（Redis 不可用时的回退方案）。
type filePlayerStateStore struct {
	dir string
	mu  sync.RWMutex
}

func newFilePlayerStateStore(dir string) *filePlayerStateStore {
	os.MkdirAll(dir, 0755)
	return &filePlayerStateStore{dir: dir}
}

func (s *filePlayerStateStore) path(userID string) string {
	return filepath.Join(s.dir, userID+".json")
}

func (s *filePlayerStateStore) Load(userID string) (*PlayerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state PlayerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *filePlayerStateStore) Save(userID string, state *PlayerState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(userID), data, 0644)
}

func (s *filePlayerStateStore) Delete(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path(userID))
}

func (s *filePlayerStateStore) Ping() error { return nil }

// ExploreService 地图探索服务（Redis + 文件持久化）
type ExploreService struct {
	mu           sync.RWMutex
	regions      map[string]*model.MapRegion
	playerStates map[string]*PlayerState // 内存缓存
	stateStore   playerStateStore        // 持久化存储
	npcList      []*model.NPC
	npcMap       map[string]*model.NPC
	spotList     []*model.GatheringSpot
	spotMap      map[string]*model.GatheringSpot
}

// NewExploreService 创建探索服务。
// 若 rdb 不为 nil 则使用 Redis 持久化玩家状态，否则回退到文件存储。
// dataDir 用于文件回退时的数据目录。
func NewExploreService(regionsPath, npcsPath, spotsPath string, rdb *redis.Client, dataDir string) (*ExploreService, error) {
	regions, err := loadRegions(regionsPath)
	if err != nil {
		return nil, fmt.Errorf("加载地图配置失败: %w", err)
	}

	npcs, err := loadNPCs(npcsPath)
	if err != nil {
		return nil, fmt.Errorf("加载NPC配置失败: %w", err)
	}

	spots, err := loadGatheringSpots(spotsPath)
	if err != nil {
		return nil, fmt.Errorf("加载采集点配置失败: %w", err)
	}

	var store playerStateStore
	if rdb != nil {
		store = newRedisPlayerStateStore(rdb)
	} else {
		store = newFilePlayerStateStore(filepath.Join(dataDir, "player_states"))
	}

	return &ExploreService{
		regions:      regions,
		playerStates: make(map[string]*PlayerState),
		stateStore:   store,
		npcList:      npcs,
		npcMap:       makeNPCIndex(npcs),
		spotList:     spots,
		spotMap:      makeSpotIndex(spots),
	}, nil
}

// loadRegions 从 JSON 文件加载地图区域
func loadRegions(path string) (map[string]*model.MapRegion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var regions []*model.MapRegion
	if err := json.Unmarshal(data, &regions); err != nil {
		return nil, err
	}
	result := make(map[string]*model.MapRegion, len(regions))
	for _, r := range regions {
		result[r.ID] = r
	}
	return result, nil
}

// loadNPCs 从 JSON 文件加载NPC配置
func loadNPCs(path string) ([]*model.NPC, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var npcs []*model.NPC
	if err := json.Unmarshal(data, &npcs); err != nil {
		return nil, err
	}
	return npcs, nil
}

// loadGatheringSpots 从 JSON 文件加载采集点配置
func loadGatheringSpots(path string) ([]*model.GatheringSpot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spots []*model.GatheringSpot
	if err := json.Unmarshal(data, &spots); err != nil {
		return nil, err
	}
	return spots, nil
}

// makeNPCIndex 构建NPC ID索引
func makeNPCIndex(npcs []*model.NPC) map[string]*model.NPC {
	idx := make(map[string]*model.NPC, len(npcs))
	for _, n := range npcs {
		idx[n.ID] = n
	}
	return idx
}

// makeSpotIndex 构建采集点ID索引
func makeSpotIndex(spots []*model.GatheringSpot) map[string]*model.GatheringSpot {
	idx := make(map[string]*model.GatheringSpot, len(spots))
	for _, s := range spots {
		idx[s.ID] = s
	}
	return idx
}

// GetAllRegions 获取所有地图区域
func (s *ExploreService) GetAllRegions() []*model.MapRegion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.MapRegion, 0, len(s.regions))
	for _, r := range s.regions {
		result = append(result, r)
	}
	return result
}

// Ping 检查持久化存储（Redis/文件）是否正常。
func (s *ExploreService) Ping() error {
	return s.stateStore.Ping()
}

// GetRegion 获取指定区域
func (s *ExploreService) GetRegion(regionID string) (*model.MapRegion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.regions[regionID]
	return r, ok
}

// GetRegionConnections 获取相邻区域列表
func (s *ExploreService) GetRegionConnections(regionID string) []*model.MapRegion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	region, ok := s.regions[regionID]
	if !ok {
		return nil
	}
	var connections []*model.MapRegion
	for _, connID := range region.Connections {
		if conn, ok := s.regions[connID]; ok {
			connections = append(connections, conn)
		}
	}
	return connections
}

// GetPlayerExploreInfo 获取玩家探索信息，优先从内存缓存读取，未命中则从持久化存储加载。
func (s *ExploreService) GetPlayerExploreInfo(userID string) *PlayerState {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.playerStates[userID]
	if !ok {
		// 尝试从持久化存储加载
		loaded, err := s.stateStore.Load(userID)
		if err == nil && loaded != nil {
			state = loaded
		} else {
			state = &PlayerState{
				UserID:            userID,
				RegionID:          DefaultRegionID,
				DiscoveredRegions: []string{DefaultRegionID},
				ActionPoints:      ActionPointMax,
				LastMoveAt:        time.Now(),
				LastAPUpdate:      time.Now(),
			}
		}
		s.playerStates[userID] = state
	}
	// 每次查询时恢复行动力
	s.regenAP(state)
	return state
}

// regenAP 计算行动力恢复
func (s *ExploreService) regenAP(state *PlayerState) {
	now := time.Now()
	elapsed := now.Sub(state.LastAPUpdate)
	hours := int(elapsed / APRegenInterval)
	if hours > 0 {
		regen := hours * ActionPointRegen
		state.ActionPoints += regen
		if state.ActionPoints > ActionPointMax {
			state.ActionPoints = ActionPointMax
		}
		state.LastAPUpdate = state.LastAPUpdate.Add(time.Duration(hours) * APRegenInterval)
	}
}

// MoveTo 移动玩家到目标区域, 消耗行动力
func (s *ExploreService) MoveTo(userID, targetRegionID string) (*MoveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.playerStates[userID]
	if !ok {
		state = &PlayerState{
			UserID:            userID,
			RegionID:          DefaultRegionID,
			DiscoveredRegions: []string{DefaultRegionID},
			ActionPoints:      ActionPointMax,
			LastMoveAt:        time.Now(),
			LastAPUpdate:      time.Now(),
		}
		s.playerStates[userID] = state
	}

	// 恢复行动力
	s.regenAP(state)

	// 检查当前区域
	currentRegion, ok := s.regions[state.RegionID]
	if !ok {
		return nil, fmt.Errorf("当前区域不存在")
	}

	// 检查目标区域
	targetRegion, ok := s.regions[targetRegionID]
	if !ok {
		return nil, fmt.Errorf("目标区域不存在")
	}

	// 检查是否相邻
	isConnected := false
	for _, connID := range currentRegion.Connections {
		if connID == targetRegionID {
			isConnected = true
			break
		}
	}
	if !isConnected {
		return nil, fmt.Errorf("无法从 '%s' 直接到达 '%s'，请通过相邻区域移动", currentRegion.Name, targetRegion.Name)
	}

	// 消耗行动力
	if state.ActionPoints < ActionPointCost {
		return nil, fmt.Errorf("行动力不足，需要 %d 点，当前 %d 点。行动力每小时恢复 %d 点",
			ActionPointCost, state.ActionPoints, ActionPointRegen)
	}

	// 移动冷却检查
	if time.Since(state.LastMoveAt) < 2*time.Second {
		return nil, fmt.Errorf("移动过于频繁，请稍后再试")
	}

	// 执行移动
	state.ActionPoints -= ActionPointCost
	state.RegionID = targetRegionID
	state.LastMoveAt = time.Now()

	// 更新发现区域
	discovered := false
	for _, d := range state.DiscoveredRegions {
		if d == targetRegionID {
			discovered = true
			break
		}
	}
	if !discovered {
		state.DiscoveredRegions = append(state.DiscoveredRegions, targetRegionID)
		discovered = true
	}

	// 持久化状态变化
	s.persistState(userID, state)

	return &MoveResult{
		Success:       true,
		Message:       fmt.Sprintf("你消耗了 %d 点行动力，成功到达 '%s'", ActionPointCost, targetRegion.Name),
		CurrentRegion: targetRegion,
		DiscoveredNew: discovered,
	}, nil
}

// Explore 探索当前区域，根据概率触发不同事件
func (s *ExploreService) Explore(userID string, playerLevel int, encSvc *EncounterService) *ExploreResult {
	s.mu.Lock()
	state := s.getOrCreateState(userID)
	s.regenAP(state)

	// 检查行动力
	if state.ActionPoints < ExploreAPCost {
		s.mu.Unlock()
		return &ExploreResult{
			EventType: "nothing",
			Message:   fmt.Sprintf("行动力不足，需要 %d 点，当前 %d 点", ExploreAPCost, state.ActionPoints),
		}
	}

	// 消耗行动力
	state.ActionPoints -= ExploreAPCost
	regionID := state.RegionID

	// 持久化行动力变化（在解锁前防止死锁，但 state 指针已经安全）
	s.persistState(userID, state)
	s.mu.Unlock()

	// 概率判定
	roll := rand.Float64()

	// 奇遇 30% | 遇怪 25% | 发现资源 25% | 无事 20%
	switch {
	case roll < 0.30:
		// 尝试触发奇遇
		if encSvc != nil {
			enc := encSvc.TriggerEncounter(userID, regionID, playerLevel)
			if enc != nil {
				return &ExploreResult{
					EventType: "encounter",
					Message:   enc.Description,
					Encounter: enc,
				}
			}
		}
		// 没有触发奇遇，降级为资源发现或无事件
		return s.randomResourceOrNothing(regionID)

	case roll < 0.55:
		// 遇怪(简化为获得战斗掉落)
		return &ExploreResult{
			EventType: "monster",
			Message:   "你遇到了一只游荡的妖兽，经过一番战斗将它击败，获得了一些修炼资源。",
			Resources: s.generateCombatDrops(regionID),
		}

	case roll < 0.80:
		// 发现资源
		return s.randomResourceOrNothing(regionID)

	default:
		// 无事发生
		return &ExploreResult{
			EventType: "nothing",
			Message:   "你仔细探查了周围，但什么也没发现。也许该换个地方看看。",
		}
	}
}

// randomResourceOrNothing 随机发现资源或无事
func (s *ExploreService) randomResourceOrNothing(regionID string) *ExploreResult {
	if rand.Float64() < 0.6 {
		return &ExploreResult{
			EventType: "resource",
			Message:   "你发现了一些可采集的资源！",
			Resources: s.generateExploreDrops(regionID),
		}
	}
	return &ExploreResult{
		EventType: "nothing",
		Message:   "你仔细探查了周围，但什么也没发现。也许该换个地方看看。",
	}
}

// generateExploreDrops 生成探索发现的资源掉落
func (s *ExploreService) generateExploreDrops(regionID string) []ResourceDrop {
	s.mu.RLock()
	region, ok := s.regions[regionID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}

	var drops []ResourceDrop
	for _, item := range region.Resources.Items {
		if rand.Float64() < item.Rate {
			amount := int64(1)
			if rand.Float64() < 0.3 {
				amount = int64(rand.Intn(3) + 1)
			}
			drops = append(drops, ResourceDrop{
				ItemID:   item.ItemID,
				ItemName: item.ItemID, // 实际应从物品配置获取名称
				Amount:   amount,
			})
		}
	}
	return drops
}

// generateCombatDrops 生成战斗掉落
func (s *ExploreService) generateCombatDrops(regionID string) []ResourceDrop {
	s.mu.RLock()
	region, ok := s.regions[regionID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}

	var drops []ResourceDrop
	// 战斗掉落一般比探索更好
	for _, item := range region.Resources.Items {
		rate := item.Rate * 1.5
		if rate > 1.0 {
			rate = 1.0
		}
		if rand.Float64() < rate {
			amount := int64(rand.Intn(int(item.Rate*10)) + 1)
			if amount < 1 {
				amount = 1
			}
			drops = append(drops, ResourceDrop{
				ItemID:   item.ItemID,
				ItemName: item.ItemID,
				Amount:   amount,
			})
		}
	}
	return drops
}

// getOrCreateState 获取或创建玩家状态(调用者需持有锁)
func (s *ExploreService) getOrCreateState(userID string) *PlayerState {
	state, ok := s.playerStates[userID]
	if !ok {
		// 尝试从持久化存储加载
		loaded, err := s.stateStore.Load(userID)
		if err == nil && loaded != nil {
			state = loaded
		} else {
			state = &PlayerState{
				UserID:            userID,
				RegionID:          DefaultRegionID,
				DiscoveredRegions: []string{DefaultRegionID},
				ActionPoints:      ActionPointMax,
				LastMoveAt:        time.Now(),
				LastAPUpdate:      time.Now(),
			}
		}
		s.playerStates[userID] = state
	}
	return state
}

// persistState 异步持久化玩家状态到存储后端。
func (s *ExploreService) persistState(userID string, state *PlayerState) {
	go func() {
		if err := s.stateStore.Save(userID, state); err != nil {
			fmt.Printf("[ExploreService] 持久化玩家状态失败 user=%s: %v\n", userID, err)
		}
	}()
}

// GetRegionNPCs 获取指定区域的所有NPC
func (s *ExploreService) GetRegionNPCs(regionID string) []*model.NPC {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*model.NPC
	for _, npc := range s.npcList {
		if npc.RegionID == regionID {
			result = append(result, npc)
		}
	}
	return result
}

// GetRegionGatheringSpots 获取指定区域的所有采集点
func (s *ExploreService) GetRegionGatheringSpots(regionID string) []*model.GatheringSpot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*model.GatheringSpot
	for _, spot := range s.spotList {
		if spot.RegionID == regionID {
			result = append(result, spot)
		}
	}
	return result
}

// GetAllNPCs 获取所有NPC
func (s *ExploreService) GetAllNPCs() []*model.NPC {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.NPC, len(s.npcList))
	copy(result, s.npcList)
	return result
}

// GetNPC 获取指定NPC
func (s *ExploreService) GetNPC(npcID string) (*model.NPC, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	npc, ok := s.npcMap[npcID]
	return npc, ok
}

// GetGatheringSpot 获取指定采集点
func (s *ExploreService) GetGatheringSpot(spotID string) (*model.GatheringSpot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spot, ok := s.spotMap[spotID]
	return spot, ok
}

// GetPlayerActionPoints 获取玩家当前行动力
func (s *ExploreService) GetPlayerActionPoints(userID string) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.getOrCreateState(userID)
	s.regenAP(state)
	return state.ActionPoints, ActionPointMax
}

// Gather 采集指定资源点
func (s *ExploreService) Gather(userID, spotID string) ([]ResourceDrop, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateState(userID)
	s.regenAP(state)

	// 检查行动力
	if state.ActionPoints < GatherAPCost {
		return nil, "", fmt.Errorf("行动力不足，需要 %d 点，当前 %d 点", GatherAPCost, state.ActionPoints)
	}

	// 查找采集点
	spot, ok := s.spotMap[spotID]
	if !ok {
		return nil, "", fmt.Errorf("采集点不存在")
	}

	// 检查是否在正确区域
	if spot.RegionID != state.RegionID {
		region, ok := s.regions[spot.RegionID]
		regionName := "未知区域"
		if ok {
			regionName = region.Name
		}
		return nil, "", fmt.Errorf("该采集点在 '%s'，你当前不在那里", regionName)
	}

	// 消耗行动力
	state.ActionPoints -= GatherAPCost

	// 持久化行动力变化
	s.persistState(userID, state)

	// 采集成功率判定 (基于难度)
	successRate := 1.0 - float64(spot.Difficulty)*0.08
	if successRate < 0.1 {
		successRate = 0.1
	}

	var drops []ResourceDrop
	if rand.Float64() < successRate {
		amount := spot.MinAmount + int64(rand.Intn(int(spot.MaxAmount-spot.MinAmount+1)))
		drops = append(drops, ResourceDrop{
			ItemID:   spot.ItemID,
			ItemName: spot.ItemName,
			Amount:   amount,
		})
		msg := fmt.Sprintf("采集成功！你获得了 %d 个 %s", amount, spot.ItemName)
		return drops, msg, nil
	}

	return nil, "采集失败，请再试一次。", nil
}
