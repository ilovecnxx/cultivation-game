// Package service 修炼核心业务逻辑层
package service

import (
	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

// TribulationManager 交互式渡劫系统 V2
// 管理完整的渡劫会话生命周期：开始 → 每波处理 → 成功/失败判定
type TribulationManager struct {
	logger   *slog.Logger
	config   *config.ConfigLoader
	realmSvc *RealmService
	eventBus model.EventBus
	mu       sync.RWMutex
	sessions map[string]*model.TribulationSession // playerID -> session
	rng      *rand.Rand
	rngMu    sync.Mutex
}

// waveConfig 每波雷劫的配置
type waveConfig struct {
	strikes    int     // 雷电击打次数
	multiplier float64 // 伤害倍率
}

// tribulationConfig 渡劫类型对应的完整配置
type tribulationConfig struct {
	name       string
	totalWaves int
	waves      []waveConfig
}

// tribulationConfigs 三九/六九/九九雷劫配置
var tribulationConfigs = map[model.TribulationType]*tribulationConfig{
	model.Tribulation39: {
		name:       "三九雷劫",
		totalWaves: 3,
		waves: []waveConfig{
			{strikes: 3, multiplier: 1.0},
			{strikes: 5, multiplier: 1.5},
			{strikes: 7, multiplier: 2.0},
		},
	},
	model.Tribulation69: {
		name:       "六九雷劫",
		totalWaves: 6,
		waves: []waveConfig{
			{strikes: 3, multiplier: 1.0},
			{strikes: 4, multiplier: 1.3},
			{strikes: 5, multiplier: 1.6},
			{strikes: 6, multiplier: 2.0},
			{strikes: 7, multiplier: 2.5},
			{strikes: 9, multiplier: 3.0},
		},
	},
	model.Tribulation99: {
		name:       "九九雷劫",
		totalWaves: 9,
		waves: []waveConfig{
			{strikes: 3, multiplier: 1.0},
			{strikes: 4, multiplier: 1.3},
			{strikes: 5, multiplier: 1.6},
			{strikes: 6, multiplier: 2.0},
			{strikes: 7, multiplier: 2.4},
			{strikes: 7, multiplier: 2.8},
			{strikes: 8, multiplier: 3.3},
			{strikes: 8, multiplier: 3.8},
			{strikes: 9, multiplier: 4.5},
		},
	},
}

// realmTribulationMap 境界ID → 渡劫类型映射
// 金丹(3)及以下: 三九雷劫, 元婴(4)-化神(5): 六九雷劫, 合体(6)+: 九九雷劫
func getTribulationTypeForRealm(realmID int) model.TribulationType {
	switch {
	case realmID <= 3:
		return model.Tribulation39
	case realmID <= 5:
		return model.Tribulation69
	default:
		return model.Tribulation99
	}
}

// NewTribulationManager 创建渡劫管理器
func NewTribulationManager(logger *slog.Logger, cfg *config.ConfigLoader, realmSvc *RealmService, eventBus model.EventBus) *TribulationManager {
	return &TribulationManager{
		logger:   logger,
		config:   cfg,
		realmSvc: realmSvc,
		eventBus: eventBus,
		sessions: make(map[string]*model.TribulationSession),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// StartTribulation 开始渡劫
// playerID: 玩家ID, realmID: 目标境界ID, realmLevel: 目标境界等级
// 返回创建的渡劫会话
func (m *TribulationManager) StartTribulation(playerID string, playerName string, player *model.Player) (*model.TribulationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 检查是否已有进行中的渡劫
	if existing, ok := m.sessions[playerID]; ok && existing.Status == "active" {
		return nil, fmt.Errorf("玩家 %s 正在进行渡劫（第%d波），不能重复开始", playerID, existing.CurrentWave)
	}
	tribType := getTribulationTypeForRealm(player.RealmID)
	tribCfg, ok := tribulationConfigs[tribType]
	if !ok {
		return nil, fmt.Errorf("找不到对应的渡劫配置: type=%d", tribType)
	}
	maxHP := player.BaseHP
	if maxHP <= 0 {
		maxHP = 1000 // 默认值，防止除以零
	}
	session := &model.TribulationSession{
		PlayerID:       playerID,
		PlayerName:     playerName,
		Type:           tribType,
		TypeName:       tribCfg.name,
		CurrentWave:    1,
		TotalWaves:     tribCfg.totalWaves,
		StrikesPerWave: tribCfg.waves[0].strikes,
		PlayerHP:       maxHP,
		MaxHP:          maxHP,
		DamageTaken:    0,
		StartTime:      time.Now().Unix(),
		Status:         "active",
		Guardians:      make([]string, 0),
		RealmID:        player.RealmID,
		RealmLevel:     player.RealmLevel,
	}
	m.sessions[playerID] = session
	m.logger.Info("玩家开始渡劫", "player_id", playerID, "tribulation_type", tribCfg.name, "player_name", playerName, "total_waves", tribCfg.totalWaves, "hp", session.PlayerHP, "max_hp", session.MaxHP)
	// 通过 EventBus 发送全服公告
	if m.eventBus != nil {
		m.eventBus.Publish("tribulation.started", &model.TribulationEvent{
			PlayerID:   playerID,
			PlayerName: playerName,
			Type:       tribType,
			TypeName:   tribCfg.name,
			Status:     "started",
		})
	}
	return session, nil
}

// ProcessWave 处理一波雷劫
// playerID: 玩家ID, action: 玩家选择的行动, itemID: 使用的法宝ID(可选)
// 返回本波结果
func (m *TribulationManager) ProcessWave(playerID string, action model.WaveAction) (*model.WaveResult, error) {
	m.mu.Lock()
	session, ok := m.sessions[playerID]
	if !ok || session.Status != "active" {
		m.mu.Unlock()
		return nil, fmt.Errorf("玩家 %s 没有进行中的渡劫", playerID)
	}
	if session.CurrentWave > session.TotalWaves {
		m.mu.Unlock()
		return nil, fmt.Errorf("渡劫已结束，当前状态: %s", session.Status)
	}
	// 深拷贝当前session状态用于计算
	waveNum := session.CurrentWave
	tribType := session.Type
	guardianCount := len(session.Guardians)
	currentHP := session.PlayerHP
	maxHP := session.MaxHP
	m.mu.Unlock()
	tribCfg, ok := tribulationConfigs[tribType]
	if !ok {
		return nil, fmt.Errorf("未知的渡劫类型: %d", tribType)
	}
	if waveNum-1 >= len(tribCfg.waves) {
		return nil, fmt.Errorf("波次 %d 超出配置范围", waveNum)
	}
	wc := tribCfg.waves[waveNum-1]
	// 1. 计算本波基础伤害
	baseDamage := m.CalculateThunderDamage(waveNum, session.RealmID, guardianCount)
	// 2. 根据玩家行动计算减免
	damageBefore := baseDamage
	damageReduced := int64(0)
	dodged := false
	switch action.Action {
	case "endure": // 硬抗：减少30-50%伤害
		reduction := 0.3 + float64(guardianCount)*0.05
		if reduction > 0.5 {
			reduction = 0.5
		}
		// 有护法加成
		if guardianCount > 0 {
			reduction += float64(guardianCount) * 0.01
			if reduction > 0.7 {
				reduction = 0.7
			}
		}
		damageReduced = int64(float64(baseDamage) * reduction)
		damageAfter := baseDamage - damageReduced
		currentHP -= damageAfter
	case "dodge": // 闪避：概率完全躲避
		dodgeChance := 0.15 + float64(guardianCount)*0.02
		if dodgeChance > 0.4 {
			dodgeChance = 0.4
		}
		m.rngMu.Lock()
		roll := m.rng.Float64()
		m.rngMu.Unlock()
		if roll < dodgeChance {
			dodged = true
			damageReduced = baseDamage
			// 闪避成功不受伤害
		} else {
			// 闪避失败：受到全额伤害
			damageReduced = 0
			currentHP -= baseDamage
		}
	case "artifact": // 法宝抵抗：根据物品减免伤害
		// 根据itemID计算减免（简化版：基础减免40-70%）
		reduction := 0.4
		if action.ItemID != "" {
			// 检查是否有该法宝配置
			gc := m.config.GetConfig()
			if item, ok := gc.GetBonusItem(action.ItemID); ok {
				reduction += item.RateBonus * 0.8
			}
		}
		// 护法额外加成
		if guardianCount > 0 {
			reduction += float64(guardianCount) * 0.03
		}
		if reduction > 0.85 {
			reduction = 0.85
		}
		damageReduced = int64(float64(baseDamage) * reduction)
		damageAfter := baseDamage - damageReduced
		currentHP -= damageAfter
	default:
		// 默认：硬抗
		reduction := 0.3 + float64(guardianCount)*0.05
		if reduction > 0.5 {
			reduction = 0.5
		}
		damageReduced = int64(float64(baseDamage) * reduction)
		damageAfter := baseDamage - damageReduced
		currentHP -= damageAfter
	}
	if currentHP < 0 {
		currentHP = 0
	}
	damageAfter := baseDamage - damageReduced
	if dodged {
		damageAfter = 0
	}
	isFinal := waveNum >= session.TotalWaves
	survived := currentHP > 0
	result := &model.WaveResult{
		Wave:          waveNum,
		Strikes:       wc.strikes,
		DamageBefore:  damageBefore,
		DamageAfter:   damageAfter,
		DamageReduced: damageReduced,
		Dodged:        dodged,
		Action:        action.Action,
		HPRemaining:   currentHP,
		MaxHP:         maxHP,
		Survived:      survived,
		IsFinal:       isFinal,
	}
	// 更新会话状态
	m.mu.Lock()
	defer m.mu.Unlock()
	// 重新获取session（可能有其他协程修改）
	session, ok = m.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("玩家 %s 渡劫会话已消失", playerID)
	}
	session.PlayerHP = currentHP
	session.DamageTaken += damageAfter
	if !survived {
		// 渡劫失败
		session.Status = "failed"
		m.logger.Warn("玩家渡劫失败", "player_id", playerID, "wave", waveNum, "total_damage", session.DamageTaken)
	} else if isFinal {
		// 渡劫成功 - 计算额外属性奖励
		session.Status = "success"
		bonus := m.calculateBonusStats(session)
		session.BonusStats = bonus
		m.logger.Info("玩家渡劫成功", "player_id", playerID, "hp_remaining", currentHP, "max_hp", maxHP, "hp_bonus", bonus.HPPermanent, "attack_bonus", bonus.AttackBonus, "defense_bonus", bonus.DefenseBonus, "speed_bonus", bonus.SpeedBonus)
		// 通过 EventBus 发送全服公告
		if m.eventBus != nil {
			m.eventBus.Publish("tribulation.completed", &model.TribulationEvent{
				PlayerID:   playerID,
				PlayerName: session.PlayerName,
				Type:       session.Type,
				TypeName:   session.TypeName,
				Status:     "success",
			})
		}
		m.logger.Info("玩家成功渡劫", "player_id", playerID, "tribulation_type", session.TypeName)
	} else {
		// 进入下一波
		session.CurrentWave++
		if session.CurrentWave <= session.TotalWaves {
			session.StrikesPerWave = tribCfg.waves[session.CurrentWave-1].strikes
		}
		m.logger.Info("玩家通过渡劫波次", "player_id", playerID, "wave", waveNum, "hp_remaining", currentHP, "max_hp", maxHP)
	}
	return result, nil
}

// CalculateThunderDamage 计算雷劫伤害
// 公式: base * realm_multiplier * wave_multiplier
//   - base: 基础伤害 = 200 + playerHP * 0.2
//   - realm_multiplier: 境界倍率 = 1.0 + realmID * 0.3
//   - wave_multiplier: 波次倍率，随波次递增
//   - guardian_reduction: 每名护法减少10-30%伤害
func (m *TribulationManager) CalculateThunderDamage(wave int, realmID int, guardianCount int) int64 {
	// 基础伤害（基于玩家最大HP）
	base := int64(200)
	// 境界倍率
	realmMultiplier := 1.0 + float64(realmID)*0.3
	// 波次倍率 (从1.0开始，每波+0.2)
	waveMultiplier := 1.0 + float64(wave-1)*0.2
	// 护法减免
	guardianReduction := 1.0
	if guardianCount > 0 {
		// 每名护法减少10%基础伤害，最多30%
		reduction := float64(guardianCount) * 0.10
		if reduction > 0.30 {
			reduction = 0.30
		}
		guardianReduction = 1.0 - reduction
	}
	damage := int64(float64(base) * realmMultiplier * waveMultiplier * guardianReduction)
	if damage < 50 {
		damage = 50
	}
	return damage
}

// CheckTribulationResult 检查渡劫结果
// 返回 (成功与否, 奖励属性)
func (m *TribulationManager) CheckTribulationResult(playerID string) (bool, *model.TribulationBonus) {
	m.mu.RLock()
	session, ok := m.sessions[playerID]
	m.mu.RUnlock()
	if !ok {
		return false, nil
	}
	return session.Status == "success", session.BonusStats
}

// GetActiveTribulation 获取玩家当前渡劫会话
func (m *TribulationManager) GetActiveTribulation(playerID string) *model.TribulationSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[playerID]
	if !ok {
		return nil
	}
	if session.Status != "active" {
		return nil
	}
	return session
}

// GetTribulationStatus 获取玩家渡劫完整状态（包括已结束的）
func (m *TribulationManager) GetTribulationStatus(playerID string) *model.TribulationSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[playerID]
	if !ok {
		return nil
	}
	return session
}

// AddGuardian 添加护法
// guardianID: 护法玩家ID, guardianName: 护法玩家名称
// 返回当前护法数量
func (m *TribulationManager) AddGuardian(playerID string, guardianID string, guardianName string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[playerID]
	if !ok {
		return 0, fmt.Errorf("玩家 %s 没有进行中的渡劫", playerID)
	}
	if session.Status != "active" {
		return 0, fmt.Errorf("玩家 %s 的渡劫已结束（%s）", playerID, session.Status)
	}
	// 检查是否重复添加
	for _, g := range session.Guardians {
		if g == guardianID {
			return len(session.Guardians), nil
		}
	}
	// 最多5名护法
	if len(session.Guardians) >= 5 {
		return len(session.Guardians), fmt.Errorf("护法已满（最多5人）")
	}
	session.Guardians = append(session.Guardians, guardianID)
	m.logger.Info("玩家加入渡劫护法", "guardian_name", guardianName, "player_id", playerID, "guardian_count", len(session.Guardians))
	return len(session.Guardians), nil
}

// calculateBonusStats 计算渡劫成功后的额外属性奖励
func (m *TribulationManager) calculateBonusStats(session *model.TribulationSession) *model.TribulationBonus {
	// 伤害承受越多，奖励越丰厚
	survivalRatio := float64(session.PlayerHP) / float64(session.MaxHP)
	if survivalRatio > 1.0 {
		survivalRatio = 1.0
	}
	// 血量越低说明越接近极限，奖励更多
	hardshipBonus := 1.0 - survivalRatio
	// 境界越高奖励越多
	realmBonus := 1.0 + float64(session.RealmID)*0.1
	// 计算属性奖励
	hpBonus := int64(100 * realmBonus * (1 + hardshipBonus*2))
	if hpBonus < 50 {
		hpBonus = 50
	}
	attackBonus := 0.05*realmBonus + 0.05*hardshipBonus
	defenseBonus := 0.05*realmBonus + 0.05*hardshipBonus
	speedBonus := 0.03*realmBonus + 0.03*hardshipBonus
	// 道心恢复：渡劫成功恢复1层道心
	daoXinRecover := 1
	// 限制上限
	if attackBonus > 0.30 {
		attackBonus = 0.30
	}
	if defenseBonus > 0.30 {
		defenseBonus = 0.30
	}
	if speedBonus > 0.15 {
		speedBonus = 0.15
	}
	return &model.TribulationBonus{
		HPPermanent:   hpBonus,
		AttackBonus:   math.Round(attackBonus*1000) / 1000,
		DefenseBonus:  math.Round(defenseBonus*1000) / 1000,
		SpeedBonus:    math.Round(speedBonus*1000) / 1000,
		DaoXinRecover: daoXinRecover,
	}
}

// GetTribulationTypeForDisplay 获取用于展示的渡劫类型名称
func GetTribulationTypeForDisplay(realmID, realmLevel int) string {
	tribType := getTribulationTypeForRealm(realmID)
	cfg, ok := tribulationConfigs[tribType]
	if !ok {
		return "天劫"
	}
	return cfg.name
}

// GetTribulationPreview 获取渡劫预览信息
func (m *TribulationManager) GetTribulationPreview(player *model.Player) map[string]interface{} {
	tribType := getTribulationTypeForRealm(player.RealmID)
	tribCfg, ok := tribulationConfigs[tribType]
	if !ok {
		return map[string]interface{}{
			"has_tribulation": false,
			"message":         "无需渡劫",
		}
	}
	maxHP := player.BaseHP
	if maxHP <= 0 {
		maxHP = 1000
	}
	totalWaves := tribCfg.totalWaves
	totalStrikes := 0
	wavePreviews := make([]map[string]interface{}, 0, totalWaves)
	totalEstimatedDamage := int64(0)
	for i, wc := range tribCfg.waves {
		waveDamage := m.CalculateThunderDamage(i+1, player.RealmID, 0)
		totalStrikes += wc.strikes
		totalEstimatedDamage += waveDamage
		wavePreviews = append(wavePreviews, map[string]interface{}{
			"wave":     i + 1,
			"strikes":  wc.strikes,
			"damage":   waveDamage,
			"survival": int64(maxHP) - totalEstimatedDamage,
		})
	}
	survivalChance := 0.3
	if maxHP > totalEstimatedDamage {
		survivalChance = 0.5 + float64(maxHP-totalEstimatedDamage)/float64(maxHP)*0.4
		if survivalChance > 0.90 {
			survivalChance = 0.90
		}
	} else if maxHP > totalEstimatedDamage/2 {
		survivalChance = 0.20
	}
	return map[string]interface{}{
		"has_tribulation":        true,
		"tribulation_type":       tribType,
		"type_name":              tribCfg.name,
		"total_waves":            totalWaves,
		"total_strikes":          totalStrikes,
		"total_estimated_damage": totalEstimatedDamage,
		"player_max_hp":          maxHP,
		"survival_chance":        math.Round(survivalChance*100) / 100,
		"wave_details":           wavePreviews,
	}
}

// ApplyTribulationBonus 将渡劫奖励应用到玩家
func (m *TribulationManager) ApplyTribulationBonus(playerID string, player *model.Player) (*model.TribulationBonus, error) {
	m.mu.Lock()
	session, ok := m.sessions[playerID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("玩家 %s 没有渡劫记录", playerID)
	}
	if session.Status != "success" {
		return nil, fmt.Errorf("玩家 %s 渡劫未成功，状态: %s", playerID, session.Status)
	}
	if session.BonusStats == nil {
		return nil, fmt.Errorf("玩家 %s 渡劫奖励未计算", playerID)
	}
	bonus := session.BonusStats
	// 应用奖励
	player.BaseHP += bonus.HPPermanent
	player.RealmID = session.RealmID
	player.RealmLevel = session.RealmLevel
	// 道心恢复
	player.DaoXinStacks -= bonus.DaoXinRecover
	if player.DaoXinStacks < 0 {
		player.DaoXinStacks = 0
	}
	// 清除会话
	m.mu.Lock()
	delete(m.sessions, playerID)
	m.mu.Unlock()
	return bonus, nil
}

// RemoveSession 清除指定玩家的渡劫会话（用于超时或取消）
func (m *TribulationManager) RemoveSession(playerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, playerID)
}

// CleanupExpiredSessions 清理超时会话（超过30分钟未操作）
func (m *TribulationManager) CleanupExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().Unix()
	timeout := int64(1800) // 30分钟
	for id, session := range m.sessions {
		if session.Status == "active" && now-session.StartTime > timeout {
			session.Status = "failed"
			m.logger.Warn("渡劫会话超时，自动判定为失败", "player_id", id)
			// 超时保留记录但不删除，以便查询
		}
	}
}
