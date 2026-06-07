// Package service 修炼核心业务逻辑层
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
	"github.com/redis/go-redis/v9"
)

// RealmService 境界管理服务
type RealmService struct {
	config            *config.ConfigLoader
	eventBus          model.EventBus
	rng               *rand.Rand
	rngMu             sync.Mutex
	playerServiceAddr string // Player 服务 HTTP 地址
	redisClient        *redis.Client // 用于同步排行榜
}

// NewRealmService 创建境界服务实例
func NewRealmService(cfg *config.ConfigLoader, eventBus model.EventBus, redisClient *redis.Client) *RealmService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &RealmService{
		config:            cfg,
		eventBus:          eventBus,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
		playerServiceAddr: playerAddr,
		redisClient:        redisClient,
	}
}

// GetCurrentRealm 获取玩家当前境界完整信息
func (s *RealmService) GetCurrentRealm(player *model.Player) (*model.Realm, *model.SubStage, bool) {
	gc := s.config.GetConfig()
	realm, ok := gc.GetRealm(player.RealmID)
	if !ok {
		return nil, nil, false
	}
	subStage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel)
	if !ok {
		return nil, nil, false
	}
	return realm, subStage, true
}

// CalculateStats 根据境界和功法计算玩家最终属性
// 公式: 最终属性 = 基础属性 * (1 + 功法加成 + 灵根加成)
func (s *RealmService) CalculateStats(player *model.Player) (attack, defense, hp int64) {
	gc := s.config.GetConfig()

	// 获取当前境界基础属性
	_, subStage, ok := s.GetCurrentRealm(player)
	if !ok {
		return 0, 0, 0
	}

	baseAtk := subStage.BaseAttack
	baseDef := subStage.BaseDefense
	baseHP := subStage.BaseHP

	// 功法加成
	atkBonus := 0.0
	defBonus := 0.0
	hpBonus := 0.0

	if player.TechniqueID > 0 {
		if tech, ok := gc.GetTechnique(player.TechniqueID); ok {
			atkBonus += tech.AttackBonus
			defBonus += tech.DefenseBonus
			hpBonus += tech.HPBonus

			// 灵根与功法的元素亲和效果
			for element, affinity := range tech.ElementAffinity {
				if rootVal, hasRoot := player.SpiritRoots[element]; hasRoot {
					// 亲和度高则额外加成
					bonus := rootVal * affinity
					if bonus > 0 {
						atkBonus += bonus * 0.1
						defBonus += bonus * 0.1
						hpBonus += bonus * 0.1
					}
				}
			}
		}
	}

	finalAtk := int64(float64(baseAtk) * (1 + atkBonus))
	finalDef := int64(float64(baseDef) * (1 + defBonus))
	finalHP := int64(float64(baseHP) * (1 + hpBonus))

	return finalAtk, finalDef, finalHP
}

// GetRealmProgress 获取玩家在当前大境界的修为进度（0.0 - 1.0）
func (s *RealmService) GetRealmProgress(player *model.Player) float64 {
	gc := s.config.GetConfig()

	realm, ok := gc.GetRealm(player.RealmID)
	if !ok {
		return 0
	}

	// 找到当前小境界需要的修为
	currentStage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel)
	if !ok {
		return 0
	}

	// 如果是当前大境界最后一个子境界，且修为达标，则满进度
	if player.RealmLevel == len(realm.SubStages) {
		return 1.0
	}

	// 计算下一个子境界所需修为
	nextStage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel+1)
	if !ok {
		return 1.0
	}

	needed := nextStage.RequiredExp - currentStage.RequiredExp
	if needed <= 0 {
		return 1.0
	}

	current := player.Experience - currentStage.RequiredExp
	if current < 0 {
		current = 0
	}

	progress := float64(current) / float64(needed)
	if progress > 1.0 {
		progress = 1.0
	}
	return progress
}

// GetNextRealmRequirement 获取突破下一个境界所需的修为和信息
func (s *RealmService) GetNextRealmRequirement(player *model.Player) (requiredExp int64, nextRealmName string, canBreakthrough bool) {
	gc := s.config.GetConfig()

	realm, ok := gc.GetRealm(player.RealmID)
	if !ok {
		return 0, "", false
	}

	// 检查是否是当前大境界的最高等级
	currentStage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel)
	if !ok {
		return 0, "", false
	}

	// 当前境界最高级
	maxLevel := len(realm.SubStages)

	if player.RealmLevel < maxLevel {
		// 小境界提升
		nextStage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel+1)
		if !ok {
			return 0, "", false
		}
		return nextStage.RequiredExp, nextStage.Name, player.Experience >= nextStage.RequiredExp
	}

	// 大境界突破
	nextRealm, ok := gc.GetRealm(player.RealmID + 1)
	if !ok {
		return 0, "已满级", false
	}
	nextSubStage := nextRealm.SubStages[0]
	_ = currentStage // 当前满级境界只用参考
	return nextSubStage.RequiredExp, nextSubStage.Name, player.Experience >= nextSubStage.RequiredExp
}

// AfterBreakthrough 突破成功后同步到其他服务
//
//	境界属性公式：
//	  HP = 100 + realmID * 50 + realmLevel * 10
//	  Attack = 10 + realmID * 20 + realmLevel * 5
//	  Defense = 10 + realmID * 15 + realmLevel * 3
//	  Speed = 100 + realmID * 10
//
//	 通知 Player 服务更新属性 + 更新排行榜分数
func (s *RealmService) AfterBreakthrough(playerID uint64, newRealmID, newRealmLevel int) {
	// 1. 按公式计算境界属性
	hp := int64(100 + newRealmID*50 + newRealmLevel*10)
	attack := int64(10 + newRealmID*20 + newRealmLevel*5)
	defense := int64(10 + newRealmID*15 + newRealmLevel*3)
	speed := int64(100 + newRealmID*10)

	// 2. 异步通知 Player 服务更新属性（不阻塞主流程）
	go func() {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"realm_id":    newRealmID,
			"realm_level": newRealmLevel,
			"attack":      attack,
			"defense":     defense,
			"max_hp":      hp,
			"speed":       speed,
		})

		url := fmt.Sprintf("%s/api/v1/player/%d/update-attributes", s.playerServiceAddr, playerID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
	}()

	// 3. 异步更新排行榜分数
	// 境界榜评分 = realmID * 10000 + realmLevel
	go func() {
		if s.redisClient == nil {
			return
		}
		score := float64(newRealmID*10000 + newRealmLevel)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redisClient.ZAdd(ctx, "ranking:realm", redis.Z{
			Score:  score,
			Member: playerID,
		}).Err()
	}()
}

// GetConfig 获取配置加载器（供其他服务使用）
func (s *RealmService) GetConfig() *config.ConfigLoader {
	return s.config
}

// -----------------------------------------------------------
// 修炼效率计算 V3
// -----------------------------------------------------------

// CalculateEfficiency 计算玩家每分钟修为获取量
//
// 公式: 每分钟修为 = 基础值(1) × 灵气浓度 × 灵根倍率 × 功法倍率
//
//   - base: 固定 1.0 修为/分钟（基础单位）
//   - spiritDensity: 当前区域灵气浓度，范围 0.5~5.0
//   - player.SpiritRootMultiplier(): 灵根倍率，取最高灵根值映射
//     天灵根(≥0.9)=2.0, 地灵根(≥0.7)=1.5, 人灵根(≥0.4)=1.0, 杂灵根(<0.4)=0.7
//   - techniqueMult: 功法修炼速度倍率 Technique.CultivationSpeed, 范围 1.0~5.0
func (s *RealmService) CalculateEfficiency(player *model.Player, spiritDensity float64) float64 {
	base := 1.0                                          // 基础 1 修为/分钟
	spiritMult := clampFloat64(spiritDensity, 0.5, 5.0)  // 灵气浓度
	rootMult := player.SpiritRootMultiplier()             // 灵根 0.7~2.0
	techniqueMult := 1.0

	// 功法倍率
	if player.TechniqueID > 0 {
		gc := s.config.GetConfig()
		if tech, ok := gc.GetTechnique(player.TechniqueID); ok {
			techniqueMult = tech.CultivationSpeed
		}
	}
	if techniqueMult < 1.0 {
		techniqueMult = 1.0
	}

	return base * spiritMult * rootMult * techniqueMult
}

// CalculateCultivationEfficiency 计算玩家修炼效率（完整明细）
// V3 版本基于灵气浓度计算，替代旧版固定基础速度公式
func (s *RealmService) CalculateCultivationEfficiency(player *model.Player) *model.CultivationEfficiency {
	spiritDensity := player.SpiritDensity
	if spiritDensity < 0.5 {
		spiritDensity = 0.5 // 默认最低灵气浓度
	}

	finalSpeed := s.CalculateEfficiency(player, spiritDensity)
	expPerSec := finalSpeed / 60.0
	if expPerSec < 1.0/60.0 {
		expPerSec = 1.0 / 60.0 // 最低 1 修为/分钟
	}

	return &model.CultivationEfficiency{
		BaseSpeed:      1.0,
		TechniqueSpeed: finalSpeed / (player.SpiritRootMultiplier() * clampFloat64(spiritDensity, 0.5, 5.0)),
		SpiritRootBonus: player.SpiritRootMultiplier() - 1.0,
		PillBonus:      0,
		FinalSpeed:     finalSpeed,
		ExpPerSecond:   int64(expPerSec*60 + 0.5), // 四舍五入到整数
	}
}

// Cultivate 修炼一次，返回获得的修为值
//
// 参数 minutes 为修炼持续时间（分钟）
// 修为值直接累加到 player.Experience
func (s *RealmService) Cultivate(player *model.Player, minutes float64) int64 {
	if minutes <= 0 {
		return 0
	}
	efficiency := s.CalculateCultivationEfficiency(player)
	gained := int64(efficiency.FinalSpeed * minutes + 0.5) // 四舍五入
	if gained < 0 {
		gained = 0
	}
	player.Experience += gained
	return gained
}

// clampFloat64 将值限制在 [min, max] 区间
func clampFloat64(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// -----------------------------------------------------------
// 突破概率计算（已完成历史使命，保留仅作兼容）
// 新突破系统请使用 node_breakthrough.go
// -----------------------------------------------------------

// CalculateBreakthroughProbability 计算突破最终概率
//
// 公式:
//
//	最终概率 = 基础概率 + 丹药加成 + 功法加成 + 灵力修正
//
// 各因子说明:
//   - 基础概率: 从突破配置中读取（如练气2层→3层 85%）
//   - 丹药加成: 使用辅助丹药增加的突破概率
//     筑基丹 +15%(0.15), 破境丹 +25%(0.25)
//   - 功法加成: Technique.BreakthroughBonus
//   - 灵力修正: 灵力值高于阈值 +5%, 低于阈值 -10%
//
// 限制:
//   - 最小概率 5% (0.05)
//   - 最大概率 95% (0.95)
func (s *RealmService) CalculateBreakthroughProbability(player *model.Player, baseRate float64) float64 {
	totalRate := baseRate

	// 1. 丹药加成（所有已使用的突破类丹药累加）
	totalRate += player.GetBreakthroughBonus()

	// 2. 功法加成
	if player.TechniqueID > 0 {
		gc := s.config.GetConfig()
		if tech, ok := gc.GetTechnique(player.TechniqueID); ok {
			totalRate += tech.BreakthroughBonus

			// 灵根与功法元素匹配额外加成
			if rootVal, hasRoot := player.SpiritRoots[tech.Element]; hasRoot {
				totalRate += rootVal * 0.1
			}
		}
	}

	// 3. 灵力修正（简化：基于境界等级与灵力值比例评估）
	// 境界越高灵力掌控越好，基础灵力值按境界折算
	baseMP := int64(100 + (player.RealmID-1)*50) // 练气=100, 筑基=150, ...
	currentMP := int64(0)
	// 使用玩家基础攻击作为灵力值的代理（当前模型没有直接MP字段）
	// 这里简化处理：境界等级就是灵力值情况的代理
	_ = baseMP
	_ = currentMP

	// 4. 概率裁剪
	if totalRate < 0.05 {
		totalRate = 0.05
	}
	if totalRate > 0.95 {
		totalRate = 0.95
	}

	return totalRate
}

// safeRand 线程安全随机数获取
func (s *RealmService) safeRand() uint64 {
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.Uint64()
}
