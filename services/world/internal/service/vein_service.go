// Package service 提供灵脉争夺系统的业务逻辑 (PVP territory control)
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"

	"github.com/redis/go-redis/v9"
)

// ============================================================
// 常量与默认配置
// ============================================================

const (
	ContestTimeLimit      = 30 * time.Minute // 争夺时限30分钟
	ContestCooldown       = 1 * time.Hour    // 争夺冷却1小时
	AbandonCooldown       = 1 * time.Hour    // 放弃灵脉冷却1小时
	DefenderHomeAdvantage = 1.10             // 防守方主场优势 +10%
	MaxPersonalVeins      = 1                // 个人最大持有灵脉数
	MaxUpgradeLevel       = 2                // 最大升级等级 (品质+2)
)

// upgradeCosts 升级消耗表 [upgradeLevel] = cost
var upgradeCosts = map[int]struct {
	Stones int64
	Hours  int64
}{
	1: {Stones: 10000, Hours: 1},
	2: {Stones: 50000, Hours: 4},
	3: {Stones: 200000, Hours: 12},
}

// ============================================================
// 持久化存储接口
// ============================================================

type veinStateStore interface {
	LoadVeins() (map[string]*model.SpiritVein, error)
	SaveVeins(veins map[string]*model.SpiritVein) error
	LoadContests() (map[string]*model.VeinContest, error)
	SaveContests(contests map[string]*model.VeinContest) error
	LoadDiscoveries() (map[string]map[string]bool, error)
	SaveDiscoveries(d map[string]map[string]bool) error
	Ping() error
}

// redisVeinStore 基于 Redis 的灵脉状态存储
type redisVeinStore struct {
	rdb *redis.Client
}

func newRedisVeinStore(rdb *redis.Client) *redisVeinStore {
	return &redisVeinStore{rdb: rdb}
}

func (s *redisVeinStore) keyVeins() string       { return "world:veins" }
func (s *redisVeinStore) keyContests() string     { return "world:vein_contests" }
func (s *redisVeinStore) keyDiscoveries() string  { return "world:vein_discoveries" }

func (s *redisVeinStore) LoadVeins() (map[string]*model.SpiritVein, error) {
	data, err := s.rdb.Get(context.Background(), s.keyVeins()).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var veins map[string]*model.SpiritVein
	if err := json.Unmarshal(data, &veins); err != nil {
		return nil, err
	}
	return veins, nil
}

func (s *redisVeinStore) SaveVeins(veins map[string]*model.SpiritVein) error {
	data, err := json.Marshal(veins)
	if err != nil {
		return err
	}
	return s.rdb.Set(context.Background(), s.keyVeins(), data, 0).Err()
}

func (s *redisVeinStore) LoadContests() (map[string]*model.VeinContest, error) {
	data, err := s.rdb.Get(context.Background(), s.keyContests()).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var contests map[string]*model.VeinContest
	if err := json.Unmarshal(data, &contests); err != nil {
		return nil, err
	}
	return contests, nil
}

func (s *redisVeinStore) SaveContests(contests map[string]*model.VeinContest) error {
	data, err := json.Marshal(contests)
	if err != nil {
		return err
	}
	return s.rdb.Set(context.Background(), s.keyContests(), data, 0).Err()
}

func (s *redisVeinStore) LoadDiscoveries() (map[string]map[string]bool, error) {
	data, err := s.rdb.Get(context.Background(), s.keyDiscoveries()).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var d map[string]map[string]bool
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *redisVeinStore) SaveDiscoveries(d map[string]map[string]bool) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return s.rdb.Set(context.Background(), s.keyDiscoveries(), data, 0).Err()
}

func (s *redisVeinStore) Ping() error {
	return s.rdb.Ping(context.Background()).Err()
}

// fileVeinStore 基于文件的灵脉状态存储（回退方案）
type fileVeinStore struct {
	dir string
	mu  sync.RWMutex
}

func newFileVeinStore(dir string) *fileVeinStore {
	os.MkdirAll(dir, 0755)
	return &fileVeinStore{dir: dir}
}

func (s *fileVeinStore) path(name string) string {
	return filepath.Join(s.dir, name+".json")
}

func (s *fileVeinStore) LoadVeins() (map[string]*model.SpiritVein, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path("veins"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var veins map[string]*model.SpiritVein
	if err := json.Unmarshal(data, &veins); err != nil {
		return nil, err
	}
	return veins, nil
}

func (s *fileVeinStore) SaveVeins(veins map[string]*model.SpiritVein) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(veins)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path("veins"), data, 0644)
}

func (s *fileVeinStore) LoadContests() (map[string]*model.VeinContest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path("contests"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var contests map[string]*model.VeinContest
	if err := json.Unmarshal(data, &contests); err != nil {
		return nil, err
	}
	return contests, nil
}

func (s *fileVeinStore) SaveContests(contests map[string]*model.VeinContest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(contests)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path("contests"), data, 0644)
}

func (s *fileVeinStore) LoadDiscoveries() (map[string]map[string]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path("discoveries"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var d map[string]map[string]bool
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *fileVeinStore) SaveDiscoveries(d map[string]map[string]bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path("discoveries"), data, 0644)
}

func (s *fileVeinStore) Ping() error { return nil }

// ============================================================
// SpiritVeinService 灵脉争夺服务
// ============================================================

// SpiritVeinService 灵脉争夺服务
type SpiritVeinService struct {
	mu          sync.RWMutex
	veins       map[string]*model.SpiritVein  // veinID -> vein
	contests    map[string]*model.VeinContest // veinID -> active contest
	discoveries map[string]map[string]bool     // userID -> set of discovered veinIDs
	lastContest map[string]time.Time           // "userID:veinID" -> last contest time (cooldown)
	lastAbandon map[string]time.Time           // "userID" -> last abandon time
	store       veinStateStore
}

// NewSpiritVeinService 创建灵脉争夺服务。
// 如果 rdb != nil 则使用 Redis 持久化，否则回退到文件存储。
func NewSpiritVeinService(rdb *redis.Client, dataDir string) (*SpiritVeinService, error) {
	var store veinStateStore
	if rdb != nil {
		store = newRedisVeinStore(rdb)
	} else {
		store = newFileVeinStore(filepath.Join(dataDir, "vein_data"))
	}

	svs := &SpiritVeinService{
		veins:       make(map[string]*model.SpiritVein),
		contests:    make(map[string]*model.VeinContest),
		discoveries: make(map[string]map[string]bool),
		lastContest: make(map[string]time.Time),
		lastAbandon: make(map[string]time.Time),
		store:       store,
	}

	// 尝试从持久化存储加载已有数据
	if err := svs.loadFromStore(); err != nil {
		log.Printf("[灵脉争夺] 从存储加载失败，使用默认数据: %v", err)
	}

	// 如果没有数据，初始化默认灵脉
	if len(svs.veins) == 0 {
		svs.initDefaultVeins()
		svs.persistVeins()
	}

	svs.startContestExpiryLoop()
	log.Printf("[灵脉争夺] 服务初始化完成: %d 条灵脉", len(svs.veins))
	return svs, nil
}

// loadFromStore 从持久化存储加载数据
func (s *SpiritVeinService) loadFromStore() error {
	veins, err := s.store.LoadVeins()
	if err != nil {
		return err
	}
	if veins != nil {
		s.veins = veins
	}

	contests, err := s.store.LoadContests()
	if err != nil {
		return err
	}
	if contests != nil {
		s.contests = contests
	}

	disc, err := s.store.LoadDiscoveries()
	if err != nil {
		return err
	}
	if disc != nil {
		s.discoveries = disc
	}
	return nil
}

// ============================================================
// 灵脉初始化
// ============================================================

// initDefaultVeins 初始化默认的20条灵脉
func (s *SpiritVeinService) initDefaultVeins() {
	// 灵脉名称池
	veinNames := []string{
		"九天灵脉", "地脉灵泉", "天罡灵脉", "玄黄灵脉", "紫霄灵脉",
		"青木灵脉", "赤炎灵脉", "玄冰灵脉", "厚土灵脉", "庚金灵脉",
		"星辰灵脉", "月华灵脉", "日曜灵脉", "雷音灵脉", "风啸灵脉",
		"云梦灵脉", "龙脉灵泉", "凤鸣灵脉", "麒麟灵脉", "玄武灵脉",
	}

	// 品质分布: 2x5★, 4x4★, 6x3★, 5x2★, 3x1★
	qualityDistribution := []int{5, 5, 4, 4, 4, 4, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 1, 1, 1}

	regionIDs := []string{
		"newbie_village_01", "newbie_village_01", "town_01", "town_01",
		"town_02", "town_02", "secret_realm_01", "secret_realm_01",
		"secret_realm_02", "secret_realm_02", "danger_land_01", "danger_land_01",
		"danger_land_02", "danger_land_02", "secret_realm_03", "secret_realm_03",
		"danger_land_03", "danger_land_03", "secret_realm_04", "secret_realm_04",
	}

	regionNames := map[string]string{
		"newbie_village_01": "落日森林",
		"town_01":           "青云镇",
		"town_02":           "天机城",
		"secret_realm_01":   "紫府秘境",
		"secret_realm_02":   "万妖窟",
		"secret_realm_03":   "天池秘境",
		"secret_realm_04":   "虚空裂隙",
		"danger_land_01":    "苍狼山",
		"danger_land_02":    "毒瘴沼泽",
		"danger_land_03":    "葬神渊",
	}

	now := time.Now()

	for i := 0; i < 20; i++ {
		quality := qualityDistribution[i]
		regionID := regionIDs[i]
		regionName := regionNames[regionID]
		if regionName == "" {
			regionName = "未知区域"
		}

		yieldAmount := int64(quality) * 500
		cultivationBonus := float64(quality) * 5.0

		posX := 100.0 + float64(i*50) + rand.Float64()*200.0
		posY := 200.0 + float64(i*30) + rand.Float64()*150.0

		vein := &model.SpiritVein{
			ID:               fmt.Sprintf("vein_%02d", i+1),
			Name:             veinNames[i],
			Quality:          quality,
			RegionID:         regionID,
			RegionName:       regionName,
			Position:         [2]float64{posX, posY},
			OwnerType:        "none",
			OwnerID:          "",
			OwnerName:        "",
			YieldInterval:    3600, // 1 hour
			YieldAmount:      yieldAmount,
			CultivationBonus: cultivationBonus,
			Defenders:        []string{},
			ContestedBy:      []string{},
			Status:           "idle",
			Discovered:       quality <= 2,
			UpgradeLevel:     0,
			Description:      fmt.Sprintf("品质%d星的%s，每小时产出%d灵石，修炼速度+%.0f%%", quality, veinNames[i], yieldAmount, cultivationBonus),
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		s.veins[vein.ID] = vein
	}

	log.Printf("[灵脉争夺] 初始化 %d 条灵脉完成", len(s.veins))
}

// ============================================================
// 持久化
// ============================================================

func (s *SpiritVeinService) persistVeins() {
	go func() {
		s.mu.RLock()
		veins := s.veins
		s.mu.RUnlock()
		if err := s.store.SaveVeins(veins); err != nil {
			log.Printf("[灵脉争夺] 持久化灵脉失败: %v", err)
		}
	}()
}

func (s *SpiritVeinService) persistContests() {
	go func() {
		s.mu.RLock()
		contests := s.contests
		s.mu.RUnlock()
		if err := s.store.SaveContests(contests); err != nil {
			log.Printf("[灵脉争夺] 持久化争夺记录失败: %v", err)
		}
	}()
}

func (s *SpiritVeinService) persistDiscoveries() {
	go func() {
		s.mu.RLock()
		d := s.discoveries
		s.mu.RUnlock()
		if err := s.store.SaveDiscoveries(d); err != nil {
			log.Printf("[灵脉争夺] 持久化发现记录失败: %v", err)
		}
	}()
}

// Ping 检查存储是否正常
func (s *SpiritVeinService) Ping() error {
	return s.store.Ping()
}

// ============================================================
// 查询接口
// ============================================================

// GetAllVeins 获取所有灵脉（已发现显示详情，未发现显示???）
func (s *SpiritVeinService) GetAllVeins(userID string) []*model.SpiritVein {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.SpiritVein, 0, len(s.veins))
	for _, v := range s.veins {
		vein := s.copyVein(v)
		if !vein.Discovered && !s.isDiscoveredBy(userID, vein.ID) {
			vein.Name = "???"
			vein.RegionName = "???"
			vein.Description = "未知灵脉，需要探索或购买地图才能发现"
			vein.Quality = 0
			vein.YieldAmount = 0
			vein.CultivationBonus = 0
		}
		result = append(result, vein)
	}
	return result
}

// GetVein 获取单个灵脉详情
func (s *SpiritVeinService) GetVein(veinID string, userID string) (*model.SpiritVein, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.veins[veinID]
	if !ok {
		return nil, fmt.Errorf("灵脉不存在")
	}

	vein := s.copyVein(v)
	if !vein.Discovered && !s.isDiscoveredBy(userID, veinID) {
		vein.Name = "???"
		vein.RegionName = "???"
		vein.Description = "未知灵脉"
		vein.Quality = 0
		vein.YieldAmount = 0
		vein.CultivationBonus = 0
	}
	return vein, nil
}

// GetVeinsByRegion 获取指定区域的灵脉
func (s *SpiritVeinService) GetVeinsByRegion(regionID string, userID string) []*model.SpiritVein {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.SpiritVein
	for _, v := range s.veins {
		if v.RegionID == regionID {
			vein := s.copyVein(v)
			if !vein.Discovered && !s.isDiscoveredBy(userID, vein.ID) {
				vein.Name = "???"
				vein.Description = "未知灵脉"
				vein.Quality = 0
			}
			result = append(result, vein)
		}
	}
	return result
}

// GetMyVeins 获取玩家/所属宗门的灵脉
func (s *SpiritVeinService) GetMyVeins(userID, sectID string) []*model.SpiritVein {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.SpiritVein
	for _, v := range s.veins {
		if v.OwnerType == "player" && v.OwnerID == userID {
			result = append(result, s.copyVein(v))
		} else if v.OwnerType == "sect" && sectID != "" && v.OwnerID == sectID {
			result = append(result, s.copyVein(v))
		}
	}
	return result
}

// GetContestStatus 获取争夺状态
func (s *SpiritVeinService) GetContestStatus(veinID string) (*model.VeinContest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contest, ok := s.contests[veinID]
	if !ok {
		return nil, fmt.Errorf("该灵脉当前没有争夺战")
	}
	contestCopy := *contest
	return &contestCopy, nil
}

// GetVeinHistory 获取灵脉占领历史
func (s *SpiritVeinService) GetVeinHistory(veinID string) []*model.VeinOccupationHistory {
	// 历史记录从文件加载（简化：不持久化到 Redis）
	return nil
}

// ============================================================
// 灵脉发现
// ============================================================

// DiscoverVein 玩家发现灵脉
func (s *SpiritVeinService) DiscoverVein(userID, veinID, method string) (*model.SpiritVein, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return nil, 0, fmt.Errorf("灵脉不存在")
	}

	if s.isDiscoveredBy(userID, veinID) {
		return nil, 0, fmt.Errorf("你已经发现过该灵脉")
	}

	if s.discoveries[userID] == nil {
		s.discoveries[userID] = make(map[string]bool)
	}
	s.discoveries[userID][veinID] = true

	rewardStones := int64(50 + v.Quality*20)

	// 持久化
	s.persistDiscoveries()

	vein := s.copyVein(v)
	return vein, rewardStones, nil
}

// ============================================================
// 占领与放弃
// ============================================================

// OccupyVein 玩家占领灵脉（无人占领时）
func (s *SpiritVeinService) OccupyVein(userID, userName, veinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return fmt.Errorf("灵脉不存在")
	}

	if v.Status == "contested" {
		return fmt.Errorf("灵脉正在被争夺中")
	}

	if v.OwnerType != "none" {
		return fmt.Errorf("该灵脉已被占领")
	}

	// 检查玩家个人已占领数量
	ownedCount := 0
	for _, ov := range s.veins {
		if ov.OwnerType == "player" && ov.OwnerID == userID {
			ownedCount++
		}
	}
	if ownedCount >= MaxPersonalVeins {
		return fmt.Errorf("你已达最大灵脉持有数(%d)，请先放弃已有灵脉", MaxPersonalVeins)
	}

	now := time.Now()

	// 记录历史
	if v.OwnerID != "" {
		s.recordOccupationHistory(v.ID, v.Name, v.Quality, v.OwnerType, v.OwnerID, v.OwnerName, v.OccupiedSince, now)
	}

	v.OwnerType = "player"
	v.OwnerID = userID
	v.OwnerName = userName
	v.OccupiedSince = now
	v.LastYieldTime = now
	v.Status = "occupied"
	v.UpdatedAt = now

	s.persistVeins()
	return nil
}

// AbandonVein 放弃灵脉
func (s *SpiritVeinService) AbandonVein(userID, veinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return fmt.Errorf("灵脉不存在")
	}

	if v.OwnerID != userID {
		return fmt.Errorf("你不是该灵脉的拥有者")
	}

	key := userID + "_abandon"
	if last, exists := s.lastAbandon[key]; exists {
		if time.Since(last) < AbandonCooldown {
			remaining := AbandonCooldown - time.Since(last)
			return fmt.Errorf("放弃灵脉冷却中，剩余 %.0f 分钟", remaining.Minutes())
		}
	}

	now := time.Now()

	s.recordOccupationHistory(v.ID, v.Name, v.Quality, v.OwnerType, v.OwnerID, v.OwnerName, v.OccupiedSince, now)

	v.OwnerType = "none"
	v.OwnerID = ""
	v.OwnerName = ""
	v.OccupiedSince = time.Time{}
	v.Status = "idle"
	v.UpdatedAt = now
	v.Defenders = []string{}

	s.lastAbandon[key] = now
	s.persistVeins()
	return nil
}

// ============================================================
// 争夺系统
// ============================================================

// InitiateContest 发起灵脉争夺
func (s *SpiritVeinService) InitiateContest(attackerID, attackerName, veinID string) (*model.VeinContest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return nil, fmt.Errorf("灵脉不存在")
	}

	if v.Status == "contested" {
		return nil, fmt.Errorf("该灵脉正在争夺中")
	}

	if v.OwnerType == "none" {
		return nil, fmt.Errorf("该灵脉无人占领，请使用占领功能")
	}

	if v.OwnerID == attackerID {
		return nil, fmt.Errorf("你已经是该灵脉的拥有者")
	}

	// 检查冷却
	key := attackerID + ":" + veinID
	if last, exists := s.lastContest[key]; exists {
		if time.Since(last) < ContestCooldown {
			remaining := ContestCooldown - time.Since(last)
			return nil, fmt.Errorf("争夺冷却中，剩余 %.0f 分钟", remaining.Minutes())
		}
	}

	now := time.Now()
	baseHP := int64(10000)
	defenderHP := int64(float64(baseHP) * DefenderHomeAdvantage)

	contest := &model.VeinContest{
		ID:            fmt.Sprintf("contest_%s_%d", veinID, now.Unix()),
		VeinID:        veinID,
		VeinName:      v.Name,
		AttackerID:    attackerID,
		AttackerName:  attackerName,
		DefenderID:    v.OwnerID,
		DefenderName:  v.OwnerName,
		StartTime:     now,
		EndTime:       now.Add(ContestTimeLimit),
		AttackerHP:    baseHP,
		AttackerMaxHP: baseHP,
		DefenderHP:    defenderHP,
		DefenderMaxHP: defenderHP,
		Status:        "active",
		Spectators:    []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	v.Status = "contested"
	v.ContestedBy = append(v.ContestedBy, attackerID)
	v.UpdatedAt = now

	s.contests[veinID] = contest
	s.persistVeins()
	s.persistContests()

	contestCopy := *contest
	log.Printf("[灵脉争夺] 争夺开始: %s -> %s (灵脉: %s)", attackerName, v.OwnerName, v.Name)
	return &contestCopy, nil
}

// SubmitContestAction 提交争夺战行动
func (s *SpiritVeinService) SubmitContestAction(veinID, userID string, action string, damage int64) (*model.VeinContest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contest, ok := s.contests[veinID]
	if !ok {
		return nil, fmt.Errorf("该灵脉没有活跃的争夺战")
	}

	if contest.Status != "active" {
		return nil, fmt.Errorf("争夺战已结束")
	}

	if time.Now().After(contest.EndTime) {
		contest.Status = "timeout"
		contest.UpdatedAt = time.Now()
		s.resolveContest(contest)
		return nil, fmt.Errorf("争夺战已超时")
	}

	switch action {
	case "attack":
		if userID == contest.AttackerID {
			contest.DefenderHP -= damage
			if contest.DefenderHP < 0 {
				contest.DefenderHP = 0
			}
		} else if userID == contest.DefenderID {
			contest.AttackerHP -= damage
			if contest.AttackerHP < 0 {
				contest.AttackerHP = 0
			}
		} else {
			return nil, fmt.Errorf("你不是参与方")
		}
	case "skill":
		bonusDamage := int64(float64(damage) * 1.5)
		if userID == contest.AttackerID {
			contest.DefenderHP -= bonusDamage
			if contest.DefenderHP < 0 {
				contest.DefenderHP = 0
			}
		} else if userID == contest.DefenderID {
			contest.AttackerHP -= bonusDamage
			if contest.AttackerHP < 0 {
				contest.AttackerHP = 0
			}
		} else {
			return nil, fmt.Errorf("你不是参与方")
		}
	case "heal":
		if userID == contest.AttackerID {
			contest.AttackerHP += damage
			if contest.AttackerHP > contest.AttackerMaxHP {
				contest.AttackerHP = contest.AttackerMaxHP
			}
		} else if userID == contest.DefenderID {
			contest.DefenderHP += damage
			if contest.DefenderHP > contest.DefenderMaxHP {
				contest.DefenderHP = contest.DefenderMaxHP
			}
		} else {
			return nil, fmt.Errorf("你不是参与方")
		}
	default:
		return nil, fmt.Errorf("未知行动类型: %s", action)
	}

	contest.UpdatedAt = time.Now()

	if contest.DefenderHP <= 0 {
		contest.Status = "attacker_win"
		s.resolveContest(contest)
	} else if contest.AttackerHP <= 0 {
		contest.Status = "defender_win"
		s.resolveContest(contest)
	}

	s.persistContests()
	s.persistVeins()

	contestCopy := *contest
	return &contestCopy, nil
}

// resolveContest 解决争夺结果
func (s *SpiritVeinService) resolveContest(contest *model.VeinContest) {
	v, ok := s.veins[contest.VeinID]
	if !ok {
		return
	}

	now := time.Now()

	switch contest.Status {
	case "attacker_win":
		if v.OwnerID != "" {
			s.recordOccupationHistory(v.ID, v.Name, v.Quality, v.OwnerType, v.OwnerID, v.OwnerName, v.OccupiedSince, now)
		}
		v.OwnerType = "player"
		v.OwnerID = contest.AttackerID
		v.OwnerName = contest.AttackerName
		v.OccupiedSince = now
		v.LastYieldTime = now
		v.Status = "occupied"
		log.Printf("[灵脉争夺] %s 战胜 %s，成功占领灵脉 %s", contest.AttackerName, contest.DefenderName, v.Name)

	case "defender_win":
		v.Status = "occupied"
		key := contest.AttackerID + ":" + contest.VeinID
		s.lastContest[key] = now
		log.Printf("[灵脉争夺] %s 成功防守灵脉 %s", contest.DefenderName, v.Name)

	case "timeout":
		v.Status = "occupied"
		key := contest.AttackerID + ":" + contest.VeinID
		s.lastContest[key] = now
		log.Printf("[灵脉争夺] 争夺超时, %s 保留灵脉 %s", contest.DefenderName, v.Name)
	}

	v.ContestedBy = []string{}
	v.UpdatedAt = now
	delete(s.contests, contest.VeinID)
}

// ============================================================
// 灵脉升级
// ============================================================

// UpgradeVein 开始灵脉升级
func (s *SpiritVeinService) UpgradeVein(ownerID, veinID string) (costStones int64, durationHours int64, endTime time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return 0, 0, time.Time{}, fmt.Errorf("灵脉不存在")
	}

	if v.OwnerID != ownerID {
		return 0, 0, time.Time{}, fmt.Errorf("你不是该灵脉的拥有者")
	}

	nextLevel := v.UpgradeLevel + 1
	if nextLevel > MaxUpgradeLevel {
		return 0, 0, time.Time{}, fmt.Errorf("灵脉已达最大升级等级")
	}

	cost, ok := upgradeCosts[nextLevel]
	if !ok {
		return 0, 0, time.Time{}, fmt.Errorf("该灵脉无法继续升级")
	}

	now := time.Now()
	endTime = now.Add(time.Duration(cost.Hours) * time.Hour)

	// 启动goroutine模拟升级完成
	go func(vID string, dur time.Duration) {
		time.Sleep(dur)
		if err2 := s.completeUpgrade(vID); err2 != nil {
			log.Printf("[灵脉争夺] 升级完成处理失败: %v", err2)
		}
	}(veinID, time.Duration(cost.Hours)*time.Hour)

	log.Printf("[灵脉争夺] 灵脉 %s 开始升级(Lv.%d -> Lv.%d), 耗时 %dh, 消耗 %d 灵石",
		v.Name, v.UpgradeLevel, v.UpgradeLevel+1, cost.Hours, cost.Stones)

	return cost.Stones, cost.Hours, endTime, nil
}

// completeUpgrade 完成灵脉升级
func (s *SpiritVeinService) completeUpgrade(veinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return fmt.Errorf("灵脉不存在")
	}

	v.UpgradeLevel++
	v.YieldAmount = int64(v.Quality) * 500 * (1 + int64(v.UpgradeLevel))
	v.CultivationBonus = float64(v.Quality) * 5.0 * (1 + float64(v.UpgradeLevel)*0.2)
	v.Description = fmt.Sprintf("品质%d星(强化Lv.%d)的%s，每小时产出%d灵石，修炼速度+%.0f%%",
		v.Quality, v.UpgradeLevel, v.Name, v.YieldAmount, v.CultivationBonus)
	v.UpdatedAt = time.Now()

	s.persistVeins()
	log.Printf("[灵脉争夺] 灵脉 %s 升级完成! 当前强化等级: %d, 产出: %d/h", v.Name, v.UpgradeLevel, v.YieldAmount)
	return nil
}

// ============================================================
// 灵脉产出
// ============================================================

// CollectYield 收取灵脉产出
func (s *SpiritVeinService) CollectYield(veinID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.veins[veinID]
	if !ok {
		return 0, fmt.Errorf("灵脉不存在")
	}

	if v.Status != "occupied" {
		return 0, fmt.Errorf("灵脉未被占领，无产出")
	}

	now := time.Now()
	elapsed := now.Sub(v.LastYieldTime)
	if elapsed <= 0 {
		return 0, nil
	}

	// 每小时产出
	hours := int64(elapsed.Hours())
	if hours <= 0 {
		return 0, nil
	}

	totalYield := hours * v.YieldAmount
	v.LastYieldTime = v.LastYieldTime.Add(time.Duration(hours) * time.Hour)
	v.UpdatedAt = now

	s.persistVeins()
	return totalYield, nil
}

// ============================================================
// 辅助方法
// ============================================================

func (s *SpiritVeinService) copyVein(v *model.SpiritVein) *model.SpiritVein {
	c := *v
	c.Defenders = make([]string, len(v.Defenders))
	copy(c.Defenders, v.Defenders)
	c.ContestedBy = make([]string, len(v.ContestedBy))
	copy(c.ContestedBy, v.ContestedBy)
	return &c
}

func (s *SpiritVeinService) isDiscoveredBy(userID, veinID string) bool {
	if discoveries, ok := s.discoveries[userID]; ok {
		return discoveries[veinID]
	}
	return false
}

func (s *SpiritVeinService) recordOccupationHistory(veinID, veinName string, quality int, ownerType, ownerID, ownerName string, occupiedSince, lostAt time.Time) {
	var duration int64
	if !occupiedSince.IsZero() {
		duration = int64(lostAt.Sub(occupiedSince).Hours())
	}
	_ = duration
	// 简化为日志记录
	log.Printf("[灵脉争夺] 占领历史: %s 被 %s(%s) 从 %s 占领到 %s (共%dh)",
		veinName, ownerName, ownerType, occupiedSince.Format("2006-01-02 15:04"), lostAt.Format("2006-01-02 15:04"), duration)
}

func (s *SpiritVeinService) startContestExpiryLoop() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.checkExpiredContests()
		}
	}()
}

func (s *SpiritVeinService) checkExpiredContests() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for veinID, contest := range s.contests {
		if now.After(contest.EndTime) && contest.Status == "active" {
			contest.Status = "timeout"
			contest.UpdatedAt = now
			s.resolveContest(contest)
			s.persistContests()
			s.persistVeins()
			log.Printf("[灵脉争夺] 争夺超时自动处理: 灵脉 %s", veinID)
		}
	}
}

// GetSectMaxVeins 根据宗门等级获取最大灵脉数
func (s *SpiritVeinService) GetSectMaxVeins(sectLevel int) int {
	switch {
	case sectLevel >= 10:
		return 4
	case sectLevel >= 7:
		return 3
	case sectLevel >= 4:
		return 2
	default:
		return 1
	}
}

// CalculatePlayerCombatPower 计算玩家战力（简化版）
func (s *SpiritVeinService) CalculatePlayerCombatPower(level int, attack, defense float64) int64 {
	base := float64(level) * 100
	power := base + attack*2 + defense*1.5
	return int64(math.Max(power, 100))
}
