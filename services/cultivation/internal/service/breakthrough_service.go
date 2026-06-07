package service
import (
	"fmt"
	"log/slog"
	"math/rand"
	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
)
// PlayerRepository 玩家数据仓储接口（服务层使用）
type PlayerRepository interface {
	GetPlayer(id uint64) (*model.Player, error)
	SavePlayer(player *model.Player) error
}
// ProtectionChecker 新手保护检查接口
type ProtectionChecker interface {
	// GetBreakthroughGraceReduction 获取突破惩罚减免比例(0.0-1.0)，同时会消耗一次免罚次数
	UseBreakthroughGrace(playerID uint64) (reduction float64, err error)
}
// BreakthroughService 突破系统服务
type BreakthroughService struct {
	logger     *slog.Logger
	config     *config.ConfigLoader
	realmSvc   *RealmService
	eventBus   model.EventBus
	repo       PlayerRepository
	tribulationMgr *TribulationManager
	heartDemonSvc *HeartDemonService
	protection ProtectionChecker
}
// NewBreakthroughService 创建突破服务实例
func NewBreakthroughService(logger *slog.Logger, cfg *config.ConfigLoader, realmSvc *RealmService, eventBus model.EventBus, repo PlayerRepository, tribulationMgr *TribulationManager) *BreakthroughService {
	return &BreakthroughService{
		logger:        logger,
		config:        cfg,
		realmSvc:      realmSvc,
		eventBus:      eventBus,
		repo:          repo,
		tribulationMgr: tribulationMgr,
	}
}
// SetProtectionChecker 设置新手保护检查器（避免循环依赖）
func (s *BreakthroughService) SetProtectionChecker(p ProtectionChecker) {
	s.protection = p
}
// SetHeartDemonService 设置心魔服务（避免循环依赖）
func (s *BreakthroughService) SetHeartDemonService(hdSvc *HeartDemonService) {
	s.heartDemonSvc = hdSvc
}
// CalculateBreakthroughRate 计算最终突破成功率
//   - base: 根据境界和是否为跨大境界从配置读取
//   - 加成: 丹药 + 气运 + 护法
//   - 惩罚: 业力
//   - 仓促突破: 累计修为 < 突破所需修为 * 1.2 时 -10%
//   - 最终裁剪至 [5%, 95%]
func (s *BreakthroughService) CalculateBreakthroughRate(
	player *model.Player,
	isMajorRealm bool,
	pillBonus float64,
	luckBonus float64,
	guardianBonus float64,
	karmaPenalty float64,
) float64 {
	base := s.getBaseRate(player.RealmID, player.RealmLevel, isMajorRealm)
	final := base + pillBonus + luckBonus + guardianBonus - karmaPenalty
	// 仓促突破惩罚：累计修为未达到突破所需修为的 1.2 倍
	if player.MaxExpForLevel > 0 && player.AccumulatedExp < player.MaxExpForLevel*6/5 {
		final -= 0.10
	}
	if final < 0.05 {
		final = 0.05
	}
	if final > 0.95 {
		final = 0.95
	}
	return final
}
// AttemptBreakthrough 执行突破判定
// 参数:
//   - player: 玩家对象（会被修改）
//   - rate: 最终成功率（由 CalculateBreakthroughRate 计算）
//   - isMajorRealm: 是否为大境界突破
//
// 返回结果，突破成功后玩家境界自动提升，失败则执行惩罚。
func (s *BreakthroughService) AttemptBreakthrough(
	player *model.Player,
	rate float64,
	isMajorRealm bool,
) (*model.BreakthroughResult, error) {
	roll := rand.Float64()
	success := roll < rate
	result := &model.BreakthroughResult{
		Success:   success,
		FinalRate: rate,
	}
	if success {
		// ---- 突破成功 ----
		player.RealmLevel++
		if player.RealmLevel > 10 {
			player.RealmID++
			player.RealmLevel = 1
		}
		// 消耗气运
		luckCost := int64(20)
		if isMajorRealm {
			luckCost += 150
		}
		player.Luck = maxInt64(0, player.Luck-luckCost)
		result.LuckCost = luckCost
		// 更新属性
		result.NewRealmID = player.RealmID
		result.NewRealmLevel = player.RealmLevel
		atk, def, hp := s.realmSvc.CalculateStats(player)
		player.BaseAttack = atk
		player.BaseDefense = def
		player.BaseHP = hp
		// 更新 MaxExpForLevel
		player.MaxExpForLevel = s.getLevelExp(player)
		// 发布突破事件
		if s.eventBus != nil {
			s.eventBus.Publish("player.breakthrough", &model.BreakthroughEvent{
				PlayerID:   player.ID,
				NewRealmID: player.RealmID,
			})
		}
		// 通知 Player 服务更新境界和属性
		s.realmSvc.AfterBreakthrough(player.ID, player.RealmID, player.RealmLevel)
		s.logger.Info("玩家突破成功", "player_id", player.ID, "realm_id", player.RealmID, "realm_level", player.RealmLevel, "luck_cost", luckCost)
	} else {
		// ---- 突破失败 ----
		if roll > rate*0.5 {
			// 轻微失败：修为损失 30%
			loss := int64(float64(player.Experience) * 0.3)
			player.Experience -= loss
			if player.Experience < 0 {
				player.Experience = 0
			}
			result.ExpLoss = loss
			s.logger.Warn("玩家突破轻微失败", "player_id", player.ID, "exp_loss", loss)
		} else {
			// 严重失败
			oldExp := player.Experience
			player.Experience = 0
			player.Luck = maxInt64(0, player.Luck-50)
			lvlExp := s.getLevelExp(player)
			if isMajorRealm && lvlExp > 0 {
				player.Experience = int64(float64(lvlExp) * 0.5)
			}
			result.ExpLoss = oldExp - player.Experience
			// 大境界严重失败产生心魔
			if isMajorRealm {
				result.HeartDemon = s.generateHeartDemon(player)
				s.logger.Warn("玩家大境界突破严重失败，产生心魔", "player_id", player.ID)
			} else {
				s.logger.Warn("玩家小境界突破严重失败，修为清零，气运-50", "player_id", player.ID)
			}
		}
	}
	if err := s.repo.SavePlayer(player); err != nil {
		return result, fmt.Errorf("保存玩家突破结果失败: %w", err)
	}
	return result, nil
}
// AttemptMajorBreakthroughWithTribulation 大境界突破（通过渡劫）
// 这是V2交互式渡劫系统的入口，大境界突破必须先通过渡劫才能成功
// 流程：
//   1. 尝试突破判定（与普通突破相同）
//   2. 突破成功后提升境界并创建渡劫会话
//   3. 调用方需要引导玩家完成渡劫（前端交互）
//   4. 渡劫成功后应用奖励
// 返回突破结果（不含渡劫奖励，奖励由 TribulationManager.ApplyTribulationBonus 应用）
func (s *BreakthroughService) AttemptMajorBreakthroughWithTribulation(
	player *model.Player,
	rate float64,
) (*model.BreakthroughResult, error) {
	result := &model.BreakthroughResult{
		FinalRate: rate,
	}
	roll := rand.Float64()
	success := roll < rate
	if !success {
		// ---- 突破失败（不触发渡劫） ----
		result.Success = false
		s.applyBreakthroughPenalty(player, result, true)
		return result, nil
	}
	// ---- 突破概率判定通过，进入渡劫阶段 ----
	result.Success = true
	// 计算下一境界
	nextRealmID := player.RealmID + 1
	nextRealmLevel := 1
	// 消耗气运
	luckCost := int64(170) // 大境界固定消耗
	player.Luck = maxInt64(0, player.Luck-luckCost)
	result.LuckCost = luckCost
	// 预先提升境界（渡劫成功才算正式突破）
	oldRealmID := player.RealmID
	oldRealmLevel := player.RealmLevel
	player.RealmID = nextRealmID
	player.RealmLevel = nextRealmLevel
	// 预计算新境界属性
	atk, def, hp := s.realmSvc.CalculateStats(player)
	player.BaseAttack = atk
	player.BaseDefense = def
	player.BaseHP = hp
	result.NewRealmID = nextRealmID
	result.NewRealmLevel = nextRealmLevel
	// 创建交互式渡劫会话
	playerIDStr := fmt.Sprintf("%d", player.ID)
	playerName := player.Name
	if playerName == "" {
		playerName = fmt.Sprintf("玩家%d", player.ID)
	}
	session, err := s.tribulationMgr.StartTribulation(playerIDStr, playerName, player)
	if err != nil {
		// 渡劫创建失败，回退境界
		player.RealmID = oldRealmID
		player.RealmLevel = oldRealmLevel
		player.BaseAttack = 0
		player.BaseDefense = 0
		player.BaseHP = 0
		atk, def, hp = s.realmSvc.CalculateStats(player)
		player.BaseAttack = atk
		player.BaseDefense = def
		player.BaseHP = hp
		return result, fmt.Errorf("创建渡劫会话失败: %w", err)
	}
	s.logger.Info("突破概率判定通过，进入交互式渡劫", "player_id", player.ID, "target_realm", fmt.Sprintf("%d级%d", nextRealmID, nextRealmLevel), "tribulation_type", session.TypeName)
	result.Tribulation = &model.TribulationResult{
		Triggered: true,
	}
	return result, nil
}
// applyBreakthroughPenalty 应用突破失败惩罚
// 支持新手保护：如果有突破免罚次数，降低惩罚幅度
func (s *BreakthroughService) applyBreakthroughPenalty(player *model.Player, result *model.BreakthroughResult, isMajorRealm bool) {
	penaltyMultiplier := 1.0
	// 检查新手保护免罚次数
	if s.protection != nil {
		reduction, err := s.protection.UseBreakthroughGrace(uint64(player.ID))
		if err != nil {
			s.logger.Warn("检查新手保护失败", "player_id", player.ID, "error", err)
		} else if reduction > 0 {
			penaltyMultiplier = 1.0 - reduction
			s.logger.Info("新手保护触发，突破惩罚降低", "player_id", player.ID, "reduction", reduction, "multiplier", penaltyMultiplier)
		}
	}
	roll := rand.Float64()
	if roll > 0.5 {
		// 轻微失败：修为损失 30%（受新手保护减免）
		loss := int64(float64(player.Experience) * 0.3 * penaltyMultiplier)
		player.Experience -= loss
		if player.Experience < 0 {
			player.Experience = 0
		}
		result.ExpLoss = loss
		s.logger.Warn("玩家突破轻微失败", "player_id", player.ID, "exp_loss", loss, "penalty_multiplier", penaltyMultiplier)
	} else {
		// 严重失败（受新手保护减免）
		oldExp := player.Experience
		player.Experience = int64(float64(player.Experience) * (1.0 - penaltyMultiplier))
		player.Luck = maxInt64(0, player.Luck-int64(50*penaltyMultiplier))
		lvlExp := s.getLevelExp(player)
		if isMajorRealm && lvlExp > 0 {
			player.Experience = int64(float64(lvlExp) * 0.5 * penaltyMultiplier)
		}
		result.ExpLoss = oldExp - player.Experience
		// 大境界严重失败产生心魔
		if isMajorRealm {
			result.HeartDemon = s.generateHeartDemon(player)
			s.logger.Warn("玩家大境界突破严重失败，产生心魔", "player_id", player.ID, "penalty_multiplier", penaltyMultiplier)
		} else {
			s.logger.Warn("玩家小境界突破严重失败", "player_id", player.ID, "penalty_multiplier", penaltyMultiplier)
		}
	}
}
// UseBreakthroughItem 使用辅助物品增加突破概率
// 返回当前总加成
func (s *BreakthroughService) UseBreakthroughItem(player *model.Player, itemID string) (float64, error) {
	gc := s.config.GetConfig()
	item, ok := gc.GetBonusItem(itemID)
	if !ok {
		return 0, fmt.Errorf("辅助物品 %s 不存在", itemID)
	}
	switch {
	case len(itemID) >= 4 && itemID[:4] == "pill":
		if player.PillBonuses == nil {
			player.PillBonuses = make(map[string]float64)
		}
		player.PillBonuses[itemID] = item.RateBonus
	case len(itemID) >= 8 && itemID[:8] == "artifact":
		if player.ArtifactBonuses == nil {
			player.ArtifactBonuses = make(map[string]float64)
		}
		player.ArtifactBonuses[itemID] = item.RateBonus
	default:
		if player.PillBonuses == nil {
			player.PillBonuses = make(map[string]float64)
		}
		player.PillBonuses[itemID] = item.RateBonus
	}
	return player.GetBreakthroughBonus(), nil
}
// getBaseRate 获取基础突破概率
func (s *BreakthroughService) getBaseRate(realmID, realmLevel int, isMajorRealm bool) float64 {
	gc := s.config.GetConfig()
	return gc.GetBreakthroughRate(realmID, realmLevel)
}
// getLevelExp 获取玩家当前等级所需的修为值（用于严重失败时设置残留修为）
func (s *BreakthroughService) getLevelExp(player *model.Player) int64 {
	gc := s.config.GetConfig()
	stage, ok := gc.GetRealmByLevel(player.RealmID, player.RealmLevel)
	if !ok {
		return 0
	}
	return stage.RequiredExp
}
// generateHeartDemon 生成心魔
func (s *BreakthroughService) generateHeartDemon(player *model.Player) *model.HeartDemon {
	scenarios := []model.HeartDemonScenario{
		{
			ID: 1, Name: "贪欲之魔",
			Scenario: fmt.Sprintf("你在修炼中看到无数天材地宝，内心充满贪婪。一位神秘人出现，要用你全部修为换取一件宝物，你如何选择？"),
			OptionA: "拒绝诱惑，坚守本心",
			OptionB: "交换一部分修为换取宝物",
			OptionC: "全部交换！富贵险中求",
			KarmaCost: 30,
			Damage:    500,
		},
		{
			ID: 2, Name: "嗔怒之魔",
			Scenario: "修行路上宿敌出现，嘲讽你资质低微、修为浅薄。你感到怒火中烧，难以自控。",
			OptionA: "冷静离开，不为所动",
			OptionB: "与宿敌理论，维护尊严",
			OptionC: "全力出手，一决生死",
			KarmaCost: 20,
			Damage:    800,
		},
		{
			ID: 3, Name: "痴念之魔",
			Scenario: "你梦见已故的亲人，他们希望你放弃修仙，回归凡尘过平凡生活。醒来后道心不稳，修为有衰退迹象。",
			OptionA: "坚定道心，化思念为力量",
			OptionB: "为亲人立碑祭奠，了却尘缘",
			OptionC: "放弃修仙，回归凡尘",
			KarmaCost: 25,
			Damage:    600,
		},
		{
			ID: 4, Name: "傲慢之魔",
			Scenario: "你战胜了一位同阶修士，周围人纷纷称赞。你开始觉得自己天赋异禀，远超常人，连师长的教诲也听不进去了。",
			OptionA: "谦虚自省，继续努力",
			OptionB: "接受赞美，但保持清醒",
			OptionC: "确实如此，我本就该高人一等",
			KarmaCost: 35,
			Damage:    400,
		},
		{
			ID: 5, Name: "疑虑之魔",
			Scenario: "修炼多年仍无突破迹象，你开始怀疑自己是否选错了道路，或许从一开始就不该修仙。",
			OptionA: "回顾初心，坚定道心",
			OptionB: "寻找名师指点迷津",
			OptionC: "改修他法，另寻出路",
			KarmaCost: 15,
			Damage:    700,
		},
	}
	idx := rand.Intn(len(scenarios))
	sc := scenarios[idx]
	return &model.HeartDemon{
		ID:        sc.ID,
		Name:      sc.Name,
		Scenario:  sc.Scenario,
		Options:   []string{sc.OptionA, sc.OptionB, sc.OptionC},
		KarmaCost: sc.KarmaCost,
		Damage:    sc.Damage,
	}
}
// maxInt64 返回两个 int64 中的较大值
func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}