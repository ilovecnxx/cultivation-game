package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// PillBonusEntry 丹药突破加成配置
type PillBonusEntry struct {
	PillID  int64   `json:"id"`
	Name    string  `json:"name"`
	Quality int     `json:"quality"`
	Bonus   float64 `json:"bonus"`
}

// BodyService 炼体系统业务逻辑
type BodyService struct {
	mu         sync.RWMutex
	realmsData *model.BodyRealmsData
	playerData map[int64]*model.BodyInfo // playerID -> BodyInfo（内存存储，正式环境应使用 MySQL）

	// 丹药突破加成表 pillID -> bonus
	pillBonuses map[int64]float64

	// 每日伤害转经验限额
	dailyDamageUsed map[int64]int64  // playerID -> 今日已使用伤害量
	dailyResetDate  map[int64]string // playerID -> 最后重置日期 YYYY-MM-DD

	log *zap.Logger
}

// NewBodyService 创建 BodyService
func NewBodyService(log *zap.Logger) *BodyService {
	return &BodyService{
		playerData:      make(map[int64]*model.BodyInfo),
		pillBonuses:     make(map[int64]float64),
		dailyDamageUsed: make(map[int64]int64),
		dailyResetDate:  make(map[int64]string),
		log:             log,
	}
}

// LoadConfig 从 JSON 文件加载炼体境界配置
func (s *BodyService) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取炼体配置文件失败: %w", err)
	}

	var realmsData model.BodyRealmsData
	if err := json.Unmarshal(data, &realmsData); err != nil {
		return fmt.Errorf("解析炼体配置文件失败: %w", err)
	}

	if len(realmsData.Realms) == 0 {
		return fmt.Errorf("炼体配置文件为空")
	}

	s.mu.Lock()
	s.realmsData = &realmsData
	s.mu.Unlock()

	s.log.Info("炼体配置加载成功", zap.Int("境界数", len(realmsData.Realms)))
	return nil
}

// LoadPillData 从 items.json 加载丹药突破加成数据
func (s *BodyService) LoadPillData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取物品数据文件失败: %w", err)
	}

	var raw struct {
		Items []struct {
			ID      int64              `json:"id"`
			Name    string             `json:"name"`
			Type    string             `json:"type"`
			Effects map[string]float64 `json:"effects"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析物品数据文件失败: %w", err)
	}

	s.mu.Lock()
	s.pillBonuses = make(map[int64]float64, len(raw.Items))
	for _, item := range raw.Items {
		// 只加载包含 breakthrough_bonus 效果的道具（丹药、法宝等）
		if bonus, ok := item.Effects["breakthrough_bonus"]; ok && bonus > 0 {
			s.pillBonuses[item.ID] = bonus
			s.log.Debug("加载突破加成道具",
				zap.Int64("id", item.ID),
				zap.String("name", item.Name),
				zap.Float64("bonus", bonus))
		}
	}
	s.mu.Unlock()

	s.log.Info("丹药突破加成数据加载成功", zap.Int("数量", len(s.pillBonuses)))
	return nil
}

// GetOrCreateBodyInfo 获取或创建玩家炼体信息
func (s *BodyService) GetOrCreateBodyInfo(playerID int64) *model.BodyInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.playerData[playerID]
	if !exists {
		info = &model.BodyInfo{
			PlayerID:  playerID,
			Realm:     0,
			Level:     0,
			Exp:       0,
			MaxHPLost: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.playerData[playerID] = info
	}
	return info
}

// InitBodyCultivation 初始化炼体（从铜皮1层开始）
func (s *BodyService) InitBodyCultivation(playerID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.playerData[playerID]
	if !exists {
		info = &model.BodyInfo{
			PlayerID:  playerID,
			CreatedAt: time.Now(),
		}
		s.playerData[playerID] = info
	}

	info.Realm = model.BodyRealmCopper
	info.Level = 1
	info.Exp = 0
	info.UpdatedAt = time.Now()

	s.log.Info("炼体已开启",
		zap.Int64("玩家", playerID),
		zap.Int32("境界", info.Realm),
		zap.Int32("等级", info.Level),
	)
}

// GetBodyInfo 获取玩家炼体信息（只读）
func (s *BodyService) GetBodyInfo(playerID int64) *model.BodyInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playerData[playerID]
}

// ---------- 训练 ----------

// Train 炼体训练
// trainType: damage/pill/dungeon
// amount: 伤害量/丹药经验/秘境次数
// 返回本次获得的炼体经验
func (s *BodyService) Train(playerID int64, req *model.BodyTrainRequest) (int64, error) {
	info := s.GetOrCreateBodyInfo(playerID)

	// 检查炼体是否已开启（至少需要达到铜皮1层才能训练）
	if info.Realm <= 0 {
		// 若未开启，检查玩家是否满足筑基条件（由调用方保障）
		return 0, fmt.Errorf("炼体未开启，请先完成前置任务")
	}

	cfg := s.getRealmConfig(info.Realm)
	if cfg == nil {
		return 0, fmt.Errorf("境界配置不存在: %d", info.Realm)
	}

	var expGained int64

	switch req.TrainType {
	case "damage":
		expGained = s.trainByDamage(playerID, req.Amount)
	case "pill":
		expGained = s.trainByPill(req)
	case "dungeon":
		expGained = s.trainByDungeon(req)
	default:
		return 0, fmt.Errorf("未知的训练类型: %s", req.TrainType)
	}

	if expGained <= 0 {
		return 0, fmt.Errorf("本次训练未获得炼体经验")
	}

	// 写入经验
	s.mu.Lock()
	info.Exp += expGained
	info.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.log.Info("炼体训练",
		zap.Int64("玩家", playerID),
		zap.String("类型", req.TrainType),
		zap.Int64("获得经验", expGained),
	)
	return expGained, nil
}

// trainByDamage 通过承受伤害获得炼体经验
func (s *BodyService) trainByDamage(playerID int64, damage int64) int64 {
	cfg := s.realmsData.TrainConfig
	if cfg.DamageToExpRatio <= 0 {
		return 0
	}

	// 检查每日限额
	s.mu.Lock()
	today := time.Now().Format("2006-01-02")
	if s.dailyResetDate[playerID] != today {
		s.dailyDamageUsed[playerID] = 0
		s.dailyResetDate[playerID] = today
	}
	used := s.dailyDamageUsed[playerID]
	remaining := cfg.DailyDamageCap - used
	s.mu.Unlock()

	if remaining <= 0 {
		return 0 // 今日限额已用完
	}

	if damage > remaining {
		damage = remaining
	}

	gain := int64(float64(damage) * cfg.DamageToExpRatio)
	if gain < 1 {
		gain = 1
	}

	s.mu.Lock()
	s.dailyDamageUsed[playerID] += damage
	s.mu.Unlock()

	return gain
}

// trainByPill 通过服用炼体丹获得经验
func (s *BodyService) trainByPill(req *model.BodyTrainRequest) int64 {
	cfg := s.realmsData.TrainConfig
	base := cfg.PillExpBase
	if req.Amount > 0 {
		base += req.Amount // Amount 作为额外丹药加成
	}
	return base
}

// trainByDungeon 通过秘境修炼获得经验
func (s *BodyService) trainByDungeon(req *model.BodyTrainRequest) int64 {
	cfg := s.realmsData.TrainConfig
	base := cfg.DungeonExpBase
	if req.Amount > 1 {
		base *= req.Amount // 多次秘境
	}
	return base
}

// ---------- 突破 ----------

// Breakthrough 尝试炼体突破
// 返回: 是否成功, 错误信息
func (s *BodyService) Breakthrough(playerID int64, req *model.BodyBreakthroughRequest) (bool, error) {
	s.mu.Lock()
	info, exists := s.playerData[playerID]
	if !exists || info.Realm <= 0 {
		s.mu.Unlock()
		return false, fmt.Errorf("炼体未开启")
	}

	currentRealm := info.Realm
	currentLevel := info.Level
	s.mu.Unlock()

	realmCfg := s.getRealmConfig(currentRealm)
	if realmCfg == nil {
		return false, fmt.Errorf("境界配置不存在: %d", currentRealm)
	}

	// 计算需要的经验值（每层需要 exp_per_train * 层数，递增难度）
	neededExp := int64(realmCfg.ExpPerTrain) * int64(currentLevel+1)

	s.mu.RLock()
	exp := info.Exp
	s.mu.RUnlock()

	if exp < neededExp {
		return false, fmt.Errorf("炼体经验不足，需要 %d，当前 %d", neededExp, exp)
	}

	// 判断是否为跨大境界突破
	isMajor := currentLevel >= realmCfg.LevelCap

	// 计算成功率
	rate := s.calcBreakthroughRate(currentRealm, currentLevel, isMajor, req.PillID)

	// 随机判定
	success := rand.Float64() < rate

	s.mu.Lock()
	defer s.mu.Unlock()

	if success {
		// 突破成功
		info.Exp -= neededExp
		if info.Exp < 0 {
			info.Exp = 0
		}

		if isMajor {
			// 跨大境界：检查是否有下一境界
			nextCfg := s.getRealmConfig(currentRealm + 1)
			if nextCfg == nil {
				return false, fmt.Errorf("已达最高炼体境界")
			}
			info.Realm = currentRealm + 1
			info.Level = 1
		} else {
			info.Level = currentLevel + 1
		}
		info.UpdatedAt = time.Now()

		s.log.Info("炼体突破成功",
			zap.Int64("玩家", playerID),
			zap.Int32("境界", info.Realm),
			zap.Int32("等级", info.Level),
		)
		return true, nil
	}

	// 突破失败：扣除HP上限
	cfg := s.realmsData.TrainConfig
	penalty := cfg.FailureHPPenalty * int64(1+currentRealm/2) // 境界越高惩罚越大
	info.MaxHPLost += penalty
	info.UpdatedAt = time.Now()

	s.log.Warn("炼体突破失败",
		zap.Int64("玩家", playerID),
		zap.Int32("境界", currentRealm),
		zap.Int32("等级", currentLevel),
		zap.Int64("HP上限损失", penalty),
	)
	return false, nil
}

// calcBreakthroughRate 计算突破成功率
func (s *BodyService) calcBreakthroughRate(realm, level int32, isMajor bool, pillID int64) float64 {
	cfg := s.getRealmConfig(realm)
	if cfg == nil {
		return 0
	}

	rate := cfg.BaseBreakthroughRate

	// 小层越高，成功率略降（每层-2%）
	rate -= float64(level-1) * 0.02

	// 跨大境界额外降10%
	if isMajor {
		rate -= 0.10
	}

	// 丹药加成：查表获得突破丹药的真实加成
	if pillID > 0 {
		s.mu.RLock()
		bonus, ok := s.pillBonuses[pillID]
		s.mu.RUnlock()
		if ok {
			s.log.Debug("炼体突破丹药加成",
				zap.Int64("pill_id", pillID),
				zap.Float64("bonus", bonus))
			rate += bonus
		} else {
			s.log.Warn("炼体突破丹药ID未找到",
				zap.Int64("pill_id", pillID))
			// 未知丹药仍给予微量保底加成
			rate += 0.02
		}
	}

	// 限制范围
	rate = math.Max(0.05, math.Min(0.98, rate))
	return rate
}

// ---------- 状态查询 ----------

// GetStatus 获取炼体状态
func (s *BodyService) GetStatus(playerID int64) *model.BodyStatusResponse {
	info := s.GetOrCreateBodyInfo(playerID)
	configs := s.getConfigs()

	// 计算属性加成
	bonuses := info.CalcBonuses(configs)

	resp := &model.BodyStatusResponse{
		BodyInfo:  info,
		Bonuses:   bonuses,
		RealmName: s.getRealmName(info.Realm),
	}

	// 如果炼体未开启，用第一个境界作为 next
	if info.Realm <= 0 {
		if len(configs) > 0 {
			resp.NextRealm = configs[0].Name
		}
		resp.NextLevel = 1
		resp.Rate = 0
		resp.MaxExp = 0
		return resp
	}

	realmCfg := s.getRealmConfig(info.Realm)
	if realmCfg == nil {
		return resp
	}

	// 当前等级已满，准备突破下境界
	if info.Level >= realmCfg.LevelCap {
		nextCfg := s.getRealmConfig(info.Realm + 1)
		if nextCfg != nil {
			resp.NextRealm = nextCfg.Name
		}
		resp.NextLevel = 1
	} else {
		resp.NextRealm = realmCfg.Name
		resp.NextLevel = info.Level + 1
	}

	// 经验上限 = exp_per_train * (level+1)
	resp.MaxExp = int64(realmCfg.ExpPerTrain) * int64(info.Level+1)

	// 当前突破成功率
	isMajor := info.Level >= realmCfg.LevelCap && s.getRealmConfig(info.Realm+1) != nil
	resp.Rate = s.calcBreakthroughRate(info.Realm, info.Level, isMajor, 0)

	return resp
}

// RecoverMaxHP 恢复因突破失败损失的HP上限（自然恢复，每小时恢复固定值）
// 由外部定时任务调用
func (s *BodyService) RecoverMaxHP(playerID int64, hours int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.playerData[playerID]
	if !exists || info.MaxHPLost <= 0 {
		return 0
	}

	cfg := s.realmsData.TrainConfig
	recoverAmount := cfg.RecoverHPPerHour * hours
	if recoverAmount > info.MaxHPLost {
		recoverAmount = info.MaxHPLost
	}

	info.MaxHPLost -= recoverAmount
	info.UpdatedAt = time.Now()

	return recoverAmount
}

// ---------- 内部工具方法 ----------

func (s *BodyService) getConfigs() []model.BodyRealmConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.realmsData == nil {
		return nil
	}
	return s.realmsData.Realms
}

func (s *BodyService) getRealmConfig(id int32) *model.BodyRealmConfig {
	configs := s.getConfigs()
	for i := range configs {
		if configs[i].ID == id {
			return &configs[i]
		}
	}
	return nil
}

func (s *BodyService) getRealmName(id int32) string {
	if name, ok := model.BodyRealmNames[id]; ok {
		return name
	}
	return ""
}
