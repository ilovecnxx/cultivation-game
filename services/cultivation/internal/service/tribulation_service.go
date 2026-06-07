package service

import (
	"log/slog"
	"math"
	"math/rand"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
)

// TribulationService 天劫/心魔引擎
// 管理突破大境界时的天劫判定、心魔生成以及渡劫期特殊逻辑。
type TribulationService struct {
	logger *slog.Logger
	config   *config.ConfigLoader
	realmSvc *RealmService
}

// NewTribulationService 创建天劫服务实例
func NewTribulationService(logger *slog.Logger, cfg *config.ConfigLoader, realmSvc *RealmService) *TribulationService {
	return &TribulationService{
		logger: logger,
		config:   cfg,
		realmSvc: realmSvc,
	}
}

// ProcessTribulation 处理天劫判定
// 仅在玩家大境界突破时调用（突破到筑基及以上）
// 流程：
//  1. 根据目标境界确定劫雷数量（劫雷数量 = 1 + 大境界序号）
//  2. 依次判定每道劫雷是否通过
//  3. 生成心魔情景
//  4. 计算总伤害/业力
//  5. 渡劫期额外处理劫力积累
func (s *TribulationService) ProcessTribulation(player *model.Player) *model.TribulationResult {
	result := &model.TribulationResult{}

	// 获取当前境界（突破后的新境界）
	realm, _, ok := s.realmSvc.GetCurrentRealm(player)
	if !ok {
		result.Triggered = false
		return result
	}

	// 只有大境界才触发天劫
	if !realm.HasTribulation {
		result.Triggered = false
		return result
	}

	result.Triggered = true

	// ---- 劫雷数量 = 1 + 大境界序号 ----
	thunderCount := 1 + player.RealmID
	result.ThunderCount = thunderCount

	// ---- 计算玩家综合渡劫能力 ----
	baseRate := s.calcBaseTribulationRate(realm)
	statBonus := s.calcStatBonus(player)
	techBonus := s.calcTechniqueBonus(player)
	rootBonus := s.calcSpiritRootBonus(player)
	itemBonus := player.GetBreakthroughBonus() * 0.5

	// 劫雷通过率衰减：每道劫雷降低通过率
	thunderPassed := 0
	totalDamage := int64(0)
	maxHP := player.BaseHP
	if maxHP <= 0 {
		maxHP = 100
	}

	for i := 0; i < thunderCount; i++ {
		strikeRate := baseRate + statBonus + techBonus + rootBonus + itemBonus
		// 劫雷序号衰减: 第1道无衰减，后续每道 -5%
		strikeRate -= float64(i) * 0.05
		if strikeRate < 0.05 {
			strikeRate = 0.05
		}
		if strikeRate > 0.95 {
			strikeRate = 0.95
		}

		roll := rand.Float64()
		if roll < strikeRate {
			thunderPassed++
		} else {
			// 未通过：计算伤害
			damageRatio := realm.TribulationDamage
			if damageRatio <= 0 {
				damageRatio = 0.3
			}
			strikeDamage := int64(float64(maxHP) * damageRatio)
			// 防御减免
			defMitigation := float64(player.BaseDefense) / (float64(player.BaseDefense) + 1000)
			strikeDamage = int64(float64(strikeDamage) * (1 - defMitigation))
			if strikeDamage < 1 {
				strikeDamage = 1
			}
			totalDamage += strikeDamage
		}
	}

	result.ThunderPassed = thunderPassed
	result.Damage = totalDamage
	result.Survived = totalDamage < maxHP

	// ---- 渡劫期特殊逻辑 ----
	tribPower := s.handleTribulationTranscendence(player, player.RealmID, result)

	// ---- 生成心魔 ----
	heartDemons := s.generateHeartDemonScenarios(player, player.RealmID, tribPower)
	result.HeartDemons = heartDemons

	// ---- 业力增加 ----
	karmaGained := int64(thunderCount - thunderPassed) * 5
	if totalDamage > 0 {
		karmaGained += totalDamage / 100
	}
	result.KarmaGained = karmaGained

	// ---- 最终判定成功条件 ----
	// 全部劫雷通过且存活 + 无心魔(或能破除) = 成功
	result.Success = thunderPassed == thunderCount && result.Survived
	result.Rate = calcFinalRate(result, thunderCount)

	if result.Success {
		s.logger.Info("玩家成功渡过天劫", "player_id", player.ID, "realm", realm.Name, "thunder_passed", thunderPassed, "thunder_count", thunderCount)
	} else {
		if !result.Survived {
			s.logger.Warn("玩家渡劫失败", "player_id", player.ID, "total_damage", totalDamage, "thunder_passed", thunderPassed, "thunder_count", thunderCount)
		} else {
			s.logger.Warn("玩家渡劫未完全通过", "player_id", player.ID, "thunder_passed", thunderPassed, "thunder_count", thunderCount, "total_damage", totalDamage)
		}
	}

	return result
}

// GetTribulationInfo 获取玩家当前天劫信息（渡劫前预览）
func (s *TribulationService) GetTribulationInfo(player *model.Player) map[string]interface{} {
	gc := s.config.GetConfig()

	nextRealmID := player.RealmID + 1
	nextRealm, ok := gc.GetRealm(nextRealmID)
	if !ok {
		return map[string]interface{}{
			"has_tribulation": false,
			"message":         "已满级，无需渡劫",
		}
	}

	info := map[string]interface{}{
		"has_tribulation": nextRealm.HasTribulation,
		"realm_name":      nextRealm.Name,
	}

	if nextRealm.HasTribulation {
		thunderCount := 1 + nextRealmID
		baseRate := s.calcBaseTribulationRate(nextRealm)
		statBonus := s.calcStatBonus(player)
		techBonus := s.calcTechniqueBonus(player)
		rootBonus := s.calcSpiritRootBonus(player)

		estimatedRate := baseRate + statBonus + techBonus + rootBonus
		if estimatedRate > 0.95 {
			estimatedRate = 0.95
		}
		if estimatedRate < 0.05 {
			estimatedRate = 0.05
		}

		estimatedDamage := int64(float64(player.BaseHP) * nextRealm.TribulationDamage)
		defMitigation := float64(player.BaseDefense) / (float64(player.BaseDefense) + 1000)
		estimatedDamage = int64(float64(estimatedDamage) * (1-defMitigation)) * int64(thunderCount)

		info["thunder_count"] = thunderCount
		info["base_rate"] = baseRate
		info["estimated_rate"] = estimatedRate
		info["estimated_damage"] = estimatedDamage
		info["tribulation_name"] = s.getTribulationName(nextRealmID)
		info["warning"] = "天劫凶险，每道劫雷都会造成伤害，请做好万全准备"

		// 渡劫期特殊信息
		if nextRealmID == 9 {
			info["is_transcendence"] = true
			info["tribulation_power"] = map[string]interface{}{
				"current": player.Karma / 10,
				"max":     1000,
				"hint":    "渡劫期可积累劫力，劫力越高突破成功率越大，但天劫伤害也越高",
			}
		}
	}

	return info
}

// ResolveHeartDemon 玩家选择心魔应对方式
// demonIndex: 心魔在 result.HeartDemons 中的索引
// optionIndex: 选项索引 (0, 1, 2)
// 返回 true 表示心魔被破除
func (s *TribulationService) ResolveHeartDemon(demon *model.HeartDemon, optionIndex int) bool {
	if demon == nil || optionIndex < 0 || optionIndex >= len(demon.Options) {
		return false
	}
	// 选项0 = 正面选项（消耗业力破除心魔）
	// 选项1 = 中性选项（部分伤害）
	// 选项2 = 负面选项（全额伤害）
	switch optionIndex {
	case 0:
		return true
	case 1:
		return false
	default:
		return false
	}
}

// ---- 内部辅助方法 ----

// calcBaseTribulationRate 计算基础天劫通过率
func (s *TribulationService) calcBaseTribulationRate(realm *model.Realm) float64 {
	rate := realm.TribulationBaseRate
	if rate <= 0 {
		rate = 0.5
	}
	return rate
}

// calcStatBonus 计算玩家属性对天劫的加成
func (s *TribulationService) calcStatBonus(player *model.Player) float64 {
	if player.BaseAttack <= 0 && player.BaseDefense <= 0 {
		return 0
	}
	statRatio := float64(player.BaseAttack+player.BaseDefense) / 10000.0
	return math.Min(statRatio*0.05, 0.20)
}

// calcTechniqueBonus 计算功法对天劫的加成
func (s *TribulationService) calcTechniqueBonus(player *model.Player) float64 {
	if player.TechniqueID <= 0 {
		return 0
	}
	gc := s.config.GetConfig()
	tech, ok := gc.GetTechnique(player.TechniqueID)
	if !ok {
		return 0
	}
	bonus := tech.CultivationSpeed * 0.05
	if bonus > 0.075 {
		bonus = 0.075
	}
	return bonus
}

// calcSpiritRootBonus 计算灵根对天劫的加成
func (s *TribulationService) calcSpiritRootBonus(player *model.Player) float64 {
	bonus := 0.0
	for _, val := range player.SpiritRoots {
		bonus += val * 0.05
	}
	if bonus > 0.15 {
		bonus = 0.15
	}
	return bonus
}

// handleTribulationTranscendence 渡劫期特殊逻辑：劫力积累
// 渡劫期 (RealmID == 9) 的玩家每次触发天劫都会积累劫力
// 劫力 = player.Karma / 10（业力转化）
// 高劫力增加通过率但也增加伤害
func (s *TribulationService) handleTribulationTranscendence(player *model.Player, realmID int, result *model.TribulationResult) int64 {
	if realmID != 9 {
		return 0
	}

	// 劫力 = 业力 / 10
	tribPower := player.Karma / 10
	if tribPower < 0 {
		tribPower = 0
	}
	if tribPower > 1000 {
		tribPower = 1000
	}

	result.TribPower = &model.TribulationPower{
		Current: tribPower,
		Max:     1000,
	}

	// 劫力影响：每100点劫力提升5%通过率，但增加10%伤害
	if tribPower > 0 {
		rateBoost := float64(tribPower) / 100.0 * 0.05
		result.Rate += rateBoost
		if result.Rate > 0.95 {
			result.Rate = 0.95
		}

		damageBoost := 1.0 + float64(tribPower)/100.0*0.10
		result.Damage = int64(float64(result.Damage) * damageBoost)
	}

	s.logger.Info("玩家渡劫期劫力", "player_id", player.ID, "trib_power", tribPower, "rate_boost", float64(tribPower)/100.0*0.05, "damage_boost", 1.0+float64(tribPower)/100.0*0.10)

	return tribPower
}

// generateHeartDemonScenarios 生成心魔情景
// 根据玩家境界生成对应的心魔，每境界最多2个心魔
func (s *TribulationService) generateHeartDemonScenarios(player *model.Player, realmID int, tribPower int64) []*model.HeartDemon {
	var demons []*model.HeartDemon

	// 境界越高心魔越强
	demonCount := 0
	if realmID >= 3 {
		demonCount = 1
	}
	if realmID >= 5 {
		demonCount = 2
	}

	for i := 0; i < demonCount; i++ {
		demon := s.randomHeartDemon(player, realmID, tribPower)
		if demon != nil {
			demons = append(demons, demon)
		}
	}

	return demons
}

// randomHeartDemon 随机生成一个心魔
func (s *TribulationService) randomHeartDemon(player *model.Player, realmID int, tribPower int64) *model.HeartDemon {
	scenarios := []struct {
		id        int
		name      string
		scenario  string
		options   []string
		karmaCost int64
		damage    int64
	}{
		{
			id: 1, name: "贪欲之魔",
			scenario: "天劫之际，心魔化作无上至宝诱惑于你。若你肯放弃抵抗天劫，便可获得此宝。",
			options:  []string{"宁心静气，不为所动", "分心应对，试图兼得", "放弃抵抗，收取宝物"},
			karmaCost: 30, damage: 800,
		},
		{
			id: 2, name: "嗔怒之魔",
			scenario: "劫雷之中，你看到曾经杀害你至亲的仇人正在渡劫。愤怒瞬间充满你的内心。",
			options:  []string{"放下仇恨，专注渡劫", "怒视仇人，心神动摇", "放弃渡劫，冲过去复仇"},
			karmaCost: 40, damage: 1200,
		},
		{
			id: 3, name: "痴念之魔",
			scenario: "你突然看到已故的道侣在劫云中向你招手，说要带你远离天劫之苦。",
			options:  []string{"闭目凝神，认清幻象", "泪流满面，心神失守", "伸手去抓，随她而去"},
			karmaCost: 25, damage: 1000,
		},
		{
			id: 4, name: "傲慢之魔",
			scenario: "天劫不过如此！你已轻松渡过数道劫雷，心中升起「我命由我不由天」的豪情。",
			options:  []string{"收敛心神，谨慎应对", "仰天长笑，士气大振", "狂妄自大，放松警惕"},
			karmaCost: 20, damage: 600,
		},
		{
			id: 5, name: "疑虑之魔",
			scenario: "你突然怀疑自己：我真的能渡过此劫吗？前面还有更强的劫雷，我是不是该放弃了？",
			options:  []string{"坚定道心，无悔前行", "心生犹豫，气势减弱", "道心崩溃，放弃抵抗"},
			karmaCost: 35, damage: 900,
		},
		{
			id: 6, name: "轮回之魔",
			scenario: "你看到自己前世今生，无数轮回记忆涌入脑海。每一世你都在渡劫，每一世都失败了。",
			options:  []string{"打破轮回，今生必胜", "感悟轮回，境界微升", "沉溺回忆，道心蒙尘"},
			karmaCost: 50, damage: 1500,
		},
		{
			id: 7, name: "因果之魔",
			scenario: "曾经被你斩杀的妖兽魂魄齐来索命，它们说你的手上沾满因果，不配渡劫成仙。",
			options:  []string{"口诵真经，超度亡灵", "以力破法，强行驱散", "心生愧疚，引颈受戮"},
			karmaCost: 45, damage: 1100,
		},
	}

	// 渡劫期心魔更强大
	idx := rand.Intn(len(scenarios))
	sc := scenarios[idx]

	// 境界影响心魔伤害
	realmMultiplier := 1.0 + float64(realmID)*0.1
	damage := int64(float64(sc.damage) * realmMultiplier)

	// 劫力影响：高劫力心魔更强
	if tribPower > 0 {
		damage = int64(float64(damage) * (1.0 + float64(tribPower)/1000.0))
	}

	return &model.HeartDemon{
		ID:        sc.id,
		Name:      sc.name,
		Scenario:  sc.scenario,
		Options:   sc.options,
		KarmaCost: sc.karmaCost,
		Damage:    damage,
	}
}

// getTribulationName 获取天劫名称
func (s *TribulationService) getTribulationName(realmID int) string {
	names := map[int]string{
		2: "筑基天劫",
		3: "金丹天劫",
		4: "元婴天劫",
		5: "化神天劫",
		6: "合体天劫",
		7: "大乘天劫",
		8: "渡劫天劫",
		9: "飞升天劫",
	}
	if name, ok := names[realmID]; ok {
		return name
	}
	return "天道考验"
}

// calcFinalRate 计算最终通过率（用于信息展示）
func calcFinalRate(result *model.TribulationResult, thunderCount int) float64 {
	if thunderCount <= 0 {
		return 0
	}
	return float64(result.ThunderPassed) / float64(thunderCount)
}
