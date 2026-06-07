// Package service 心魔系统 - 突破失败产生的持久Debuff
package service
import (
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"sync"
	"time"
	"cultivation-game/services/cultivation/internal/model"
)
// HeartDemonService 持久心魔管理服务
//
// 五大心魔（贪嗔痴疑慢）：
//   - greed（贪）: 降低掉落率，增加灵石消耗
//   - wrath（嗔）: 降低防御，增加受伤
//   - ignor（痴）: 降低修为获取，增加突破难度
//   - doubt（疑）: 降低暴击率，增加技能冷却
//   - sloth（慢）: 降低修炼速度，降低移动速度
//
// 生成规则：
//   - 突破严重失败 50% 生成
//   - 渡劫失败 100% 生成
//   - 最多同时 3 个活跃心魔
//
// 压制方法：
//   - 心魔幻境挑战（每日1次/心魔）
//   - 清心丹（-1级）
//   - 镇魔符（-2级）
//   - 菩提心法（被动防护）
type HeartDemonService struct {
	logger *slog.Logger
	mu         sync.RWMutex
	playerRepo PlayerRepository
	// 幻境冷却记录 playerID_demonType -> lastChallengeTime
	illusionCD map[string]int64
	rng        *rand.Rand
}
// NewHeartDemonService 创建心魔服务
func NewHeartDemonService(logger *slog.Logger, repo PlayerRepository) *HeartDemonService {
	return &HeartDemonService{
		logger: logger,
		playerRepo: repo,
		illusionCD: make(map[string]int64),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
// ---------- 心魔生成 ----------
// GenerateDemon 生成一个心魔
//
//	player: 玩家对象
//	source: 来源（"breakthrough_failure" / "tribulation_failure" / "curse"）
//	force:  是否强制生成（渡劫失败 = 100%）
//
// 返回生成的心魔，nil 表示未生成
func (s *HeartDemonService) GenerateDemon(player *model.Player, source string, force bool) *model.PersistentHeartDemon {
	if !force && s.rng.Float64() > 0.5 {
		return nil
	}
	// 菩提心法防护 - 只有 level >= 5 的心魔才能突破
	if player.HasBodhiTechnique {
		// 如果已有心魔都低于5级，不生成
		allLow := true
		for _, d := range player.HeartDemons {
			if !d.Defeated && d.Level >= 5 {
				allLow = false
				break
			}
		}
		if allLow && len(player.HeartDemons) > 0 {
			// 菩提心法完全防护低级心魔
			s.logger.Info("菩提心法阻挡了新心魔生成", "player_id", player.ID)
			return nil
		}
	}
	// 最多3个活跃心魔
	activeCount := 0
	for _, d := range player.HeartDemons {
		if !d.Defeated {
			activeCount++
		}
	}
	if activeCount >= 3 {
		s.logger.Warn("已有3个活跃心魔，无法生成新心魔", "player_id", player.ID)
		return nil
	}
	// 根据玩家属性加权选择心魔类型
	demonType := s.weightedDemonType(player)
	demon := &model.PersistentHeartDemon{
		ID:          fmt.Sprintf("hd_%d_%d", player.ID, time.Now().UnixNano()),
		PlayerID:    player.ID,
		DemonType:   demonType,
		Level:       1,
		DebuffValue: 0.05, // 初始5%
		CreatedAt:   time.Now().Unix(),
		CreatedFrom: source,
		Defeated:    false,
	}
	player.HeartDemons = append(player.HeartDemons, *demon)
	s.logger.Info("玩家生成心魔", "player_id", player.ID, "demon_type", demon.DemonType, "level", demon.Level, "source", source)
	return demon
}
// GenerateDemonOnTribulationFail 渡劫失败生成心魔（强制100%）
func (s *HeartDemonService) GenerateDemonOnTribulationFail(player *model.Player) *model.PersistentHeartDemon {
	return s.GenerateDemon(player, "tribulation_failure", true)
}
// weightedDemonType 根据玩家属性加权选择心魔类型
func (s *HeartDemonService) weightedDemonType(player *model.Player) model.PersistentDemonType {
	types := []model.PersistentDemonType{
		model.DemonGreed,
		model.DemonWrath,
		model.DemonIgnorance,
		model.DemonDoubt,
		model.DemonSloth,
	}
	// 权重：默认均为1.0
	weights := []float64{1.0, 1.0, 1.0, 1.0, 1.0}
	// 低防御 → 更多嗔（wrath）
	if player.BaseDefense < 50 {
		weights[1] += 1.5
	}
	// 低攻击 → 更多贪（greed）
	if player.BaseAttack < 50 {
		weights[0] += 1.0
	}
	// 低气运 → 更多疑（doubt）
	if player.Luck < 30 {
		weights[3] += 2.0
	}
	// 低修为 → 更多痴（ignor）
	if player.Experience < 1000 {
		weights[2] += 1.5
	}
	// 高业力 → 全面增加
	if player.Karma > 50 {
		for i := range weights {
			weights[i] += 0.5
		}
	}
	return s.weightedSelect(types, weights)
}
// weightedSelect 加权随机选择
func (s *HeartDemonService) weightedSelect(items []model.PersistentDemonType, weights []float64) model.PersistentDemonType {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	roll := s.rng.Float64() * total
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			return items[i]
		}
	}
	return items[len(items)-1]
}
// ---------- Debuff 计算 ----------
// GetDebuffs 获取玩家所有心魔的debuff加成
//
// 返回 map[demon_type]debuff_value
// Level 1: -5%, Level 10: -50%
// 多个相同类型心魔叠加（最大值而非累加）
func (s *HeartDemonService) GetDebuffs(player *model.Player) map[model.PersistentDemonType]float64 {
	debuffs := make(map[model.PersistentDemonType]float64)
	for _, d := range player.HeartDemons {
		if d.Defeated {
			continue
		}
		val := 0.05 * float64(d.Level)
		if val > 0.50 {
			val = 0.50
		}
		// 相同类型取最高值
		if existing, ok := debuffs[d.DemonType]; !ok || val > existing {
			debuffs[d.DemonType] = val
		}
	}
	return debuffs
}
// GetMultipliers 获取心魔对各属性的倍率影响（1.0 = 无影响）
func (s *HeartDemonService) GetMultipliers(player *model.Player) map[string]float64 {
	debuffs := s.GetDebuffs(player)
	result := map[string]float64{
		"drop_rate":        1.0,
		"spirit_stone_cost": 1.0,
		"defense":           1.0,
		"damage_taken":      1.0,
		"exp_gain":          1.0,
		"breakthrough_diff": 1.0,
		"crit_rate":         1.0,
		"skill_cooldown":    1.0,
		"cultivation_speed": 1.0,
		"move_speed":        1.0,
	}
	for dtype, val := range debuffs {
		switch dtype {
		case model.DemonGreed:
			result["drop_rate"] = 1.0 - val
			result["spirit_stone_cost"] = 1.0 + val
		case model.DemonWrath:
			result["defense"] = 1.0 - val
			result["damage_taken"] = 1.0 + val
		case model.DemonIgnorance:
			result["exp_gain"] = 1.0 - val
			result["breakthrough_diff"] = 1.0 + val
		case model.DemonDoubt:
			result["crit_rate"] = 1.0 - val
			result["skill_cooldown"] = 1.0 + val
		case model.DemonSloth:
			result["cultivation_speed"] = 1.0 - val
			result["move_speed"] = 1.0 - val
		}
	}
	return result
}
// ---------- 心魔值 ----------
// CalculateHeartDemonValue 计算心魔值（0-100）
//
//	每个活跃心魔贡献 level * 10
//	>30: cultivation efficiency penalty
//	>60: unable to breakthrough
//	>90: risk of demon possession
func (s *HeartDemonService) CalculateHeartDemonValue(player *model.Player) int {
	total := 0
	for _, d := range player.HeartDemons {
		if !d.Defeated {
			total += d.Level * 10
		}
	}
	if total > 100 {
		total = 100
	}
	return total
}
// GetHeartDemonValueInfo 获取心魔值详情
func (s *HeartDemonService) GetHeartDemonValueInfo(player *model.Player) map[string]interface{} {
	value := s.CalculateHeartDemonValue(player)
	info := map[string]interface{}{
		"value":        value,
		"max":          100,
		"severity":     "safe",
		"penalties":    []string{},
		"debuffs":      s.GetDebuffs(player),
		"multipliers":  s.GetMultipliers(player),
	}
	if value > 90 {
		info["severity"] = "critical"
		info["penalties"] = append(info["penalties"].([]string), "心魔深重，有被心魔夺舍的风险")
	} else if value > 60 {
		info["severity"] = "severe"
		info["penalties"] = append(info["penalties"].([]string), "无法突破")
	} else if value > 30 {
		info["severity"] = "unstable"
		info["penalties"] = append(info["penalties"].([]string), "修炼效率降低")
	} else if value > 0 {
		info["severity"] = "mild"
	} else {
		info["severity"] = "safe"
	}
	return info
}
// ---------- 邪恶值事件 ----------
// TryPossessionEvent 尝试触发心魔夺舍事件（心魔值 > 90 时调用）
//
//	返回 true = 触发了夺舍事件
func (s *HeartDemonService) TryPossessionEvent(player *model.Player) (bool, string) {
	value := s.CalculateHeartDemonValue(player)
	if value <= 90 {
		return false, ""
	}
	// 心魔值越高越容易触发
	chance := float64(value-90) / 10.0
	if s.rng.Float64() < chance {
		events := []string{
			"修炼时真气逆流，心魔作祟，损失大量修为",
			"心神不宁，炼丹失败，材料全部报销",
			"心魔幻象丛生，战斗中不慎自伤",
			"道心动摇，突破时走火入魔，境界不稳",
		}
		evt := events[s.rng.Intn(len(events))]
		s.logger.Warn("玩家心魔夺舍事件触发", "player_id", player.ID, "event", evt)
		return true, evt
	}
	return false, ""
}
// ---------- 心魔幻境 ----------
// CanEnterIllusion 检查是否能进入心魔幻境
//
//	每心魔每日1次
//	消耗100灵石
//	菩提心法持有者免费
func (s *HeartDemonService) CanEnterIllusion(player *model.Player, demonID string) (bool, string) {
	// 查找心魔
	var demon *model.PersistentHeartDemon
	for i := range player.HeartDemons {
		if player.HeartDemons[i].ID == demonID {
			demon = &player.HeartDemons[i]
			break
		}
	}
	if demon == nil {
		return false, "心魔不存在"
	}
	if demon.Defeated {
		return false, "该心魔已被净化"
	}
	// 检查冷却
	cdKey := fmt.Sprintf("%d_%s", player.ID, string(demon.DemonType))
	s.mu.RLock()
	lastTime, exists := s.illusionCD[cdKey]
	s.mu.RUnlock()
	if exists {
		lastDate := time.Unix(lastTime, 0).Format("2006-01-02")
		today := time.Now().Format("2006-01-02")
		if lastDate == today {
			return false, "今日已挑战过该心魔的幻境"
		}
	}
	// 检查灵石（菩提心法免费）
	if !player.HasBodhiTechnique {
		// 这里简化处理，实际应从player的灵石余额检查
		// 假设 player 有 SpiritStones 字段
		return true, "需要消耗100灵石进入幻境"
	}
	return true, ""
}
// EnterIllusion 进入心魔幻境
func (s *HeartDemonService) EnterIllusion(player *model.Player, demonID string) (map[string]interface{}, string) {
	canEnter, msg := s.CanEnterIllusion(player, demonID)
	if !canEnter {
		return nil, msg
	}
	var demon *model.PersistentHeartDemon
	for i := range player.HeartDemons {
		if player.HeartDemons[i].ID == demonID {
			demon = &player.HeartDemons[i]
			break
		}
	}
	if demon == nil {
		return nil, "心魔不存在"
	}
	// 设置冷却
	cdKey := fmt.Sprintf("%d_%s", player.ID, string(demon.DemonType))
	s.mu.Lock()
	s.illusionCD[cdKey] = time.Now().Unix()
	s.mu.Unlock()
	// 构建幻境信息
	bossStats := s.calculateBossStats(demon)
	illusion := map[string]interface{}{
		"demon_id":     demon.ID,
		"demon_type":   demon.DemonType,
		"demon_level":  demon.Level,
		"boss_name":    s.GetDemonName(demon.DemonType),
		"boss_hp":      bossStats["hp"],
		"boss_atk":     bossStats["atk"],
		"special":      s.getIllusionSpecial(demon.DemonType),
		"description":  s.getIllusionDescription(demon.DemonType, demon.Level),
	}
	s.logger.Info("玩家进入心魔幻境", "player_id", player.ID, "demon_name", s.GetDemonName(demon.DemonType), "level", demon.Level)
	return illusion, ""
}
// FightIllusionBoss 挑战心魔幻境Boss
//
//	Win: demon level -1 (or defeated at level 1)
//	Lose: demon level +1 (max 10)
//
//	specialChoice: 特殊机制的玩家选择（嗔心魔的"fight"/"defend"，痴心魔的答案索引等）
func (s *HeartDemonService) FightIllusionBoss(
	player *model.Player,
	demonID string,
	specialChoice string,
	playerAtk int64,
	playerDef int64,
	playerHP int64,
) (map[string]interface{}, string) {
	var demonIdx int = -1
	var demon *model.PersistentHeartDemon
	for i := range player.HeartDemons {
		if player.HeartDemons[i].ID == demonID {
			demonIdx = i
			demon = &player.HeartDemons[i]
			break
		}
	}
	if demonIdx == -1 {
		return nil, "心魔不存在"
	}
	if demon.Defeated {
		return nil, "该心魔已被净化"
	}
	bossStats := s.calculateBossStats(demon)
	win := false
	var detail map[string]interface{}
	switch demon.DemonType {
	case model.DemonGreed:
		// 贪：Boss会偷取玩家物品，选择"sacrifice"放弃物品可降低Boss属性
		win = s.fightGreedDemon(demon, bossStats, specialChoice == "sacrifice", playerAtk, playerDef, playerHP)
		detail = map[string]interface{}{
			"mechanic": "Boss会偷取你的灵石，选择'放弃灵石'降低Boss攻击力",
		}
	case model.DemonWrath:
		// 嗔：狂暴计时器，必须快速击杀
		win = s.fightWrathDemon(demon, bossStats, specialChoice, playerAtk, playerDef, playerHP)
		detail = map[string]interface{}{
			"mechanic": "嗔怒心魔有狂暴计时，3回合内必须击杀",
		}
	case model.DemonIgnorance:
		// 痴：问答机制，答错回血
		win = s.fightIgnorDemon(demon, bossStats, specialChoice, playerAtk, playerDef, playerHP)
		detail = map[string]interface{}{
			"mechanic": "需要回答修行问题，答错则Boss恢复大量HP",
		}
	case model.DemonDoubt:
		// 疑：分身机制，找到真身
		win = s.fightDoubtDemon(demon, bossStats, specialChoice, playerAtk, playerDef, playerHP)
		detail = map[string]interface{}{
			"mechanic": "心魔会制造分身，选对真身才能造成伤害",
		}
	case model.DemonSloth:
		// 慢：行动减慢，需要耐心
		win = s.fightSlothDemon(demon, bossStats, specialChoice, playerAtk, playerDef, playerHP)
		detail = map[string]interface{}{
			"mechanic": "你的行动速度降低50%，需要耐心周旋",
		}
	default:
		// 默认普通战斗
		win = s.fightDefault(demon, bossStats, playerAtk, playerDef, playerHP)
	}
	levelBefore := demon.Level
	levelAfter := demon.Level
	if win {
		if demon.Level <= 1 {
			// 击败心魔
			player.HeartDemons[demonIdx].Defeated = true
			player.HeartDemons[demonIdx].DefeatedAt = time.Now().Unix()
			levelAfter = 0
			s.logger.Info("玩家成功净化心魔", "player_id", player.ID, "demon_type", demon.DemonType)
		} else {
			player.HeartDemons[demonIdx].Level--
			levelAfter = demon.Level - 1
			s.updateDebuffValue(&player.HeartDemons[demonIdx])
			s.logger.Info("玩家在幻境中击败心魔", "player_id", player.ID, "demon_type", demon.DemonType, "level_change", -1, "new_level", levelAfter)
		}
	} else {
		if demon.Level < 10 {
			player.HeartDemons[demonIdx].Level++
			levelAfter = demon.Level + 1
			s.updateDebuffValue(&player.HeartDemons[demonIdx])
		}
		s.logger.Info("玩家在心魔幻境中被击败", "player_id", player.ID, "demon_type", demon.DemonType, "level_change", 1, "new_level", levelAfter)
	}
	// 保存记录
	record := &model.DemonIllusionRecord{
		ID:           fmt.Sprintf("ir_%d_%d", player.ID, time.Now().UnixNano()),
		PlayerID:     player.ID,
		DemonType:    demon.DemonType,
		ChallengedAt: time.Now().Unix(),
		Won:          win,
		LevelBefore:  levelBefore,
		LevelAfter:   levelAfter,
	}
	result := map[string]interface{}{
		"win":          win,
		"demon_type":   demon.DemonType,
		"demon_name":   s.GetDemonName(demon.DemonType),
		"level_before": levelBefore,
		"level_after":  levelAfter,
		"defeated":     levelAfter == 0,
		"record":       record,
		"detail":       detail,
	}
	return result, ""
}
// calculateBossStats 计算幻境Boss属性
func (s *HeartDemonService) calculateBossStats(demon *model.PersistentHeartDemon) map[string]int64 {
	baseHP := int64(500 + demon.Level*200)
	baseAtk := int64(30 + demon.Level*15)
	baseDef := int64(10 + demon.Level*8)
	return map[string]int64{
		"hp":  baseHP,
		"atk": baseAtk,
		"def": baseDef,
	}
}
// 各心魔战斗逻辑
func (s *HeartDemonService) fightGreedDemon(demon *model.PersistentHeartDemon, boss map[string]int64, sacrifice bool, atk, def, hp int64) bool {
	effectiveAtk := boss["atk"]
	if sacrifice {
		effectiveAtk = boss["atk"] * 2 / 3 // 放弃灵石降低Boss攻击
	}
	// 模拟战斗：对比战力
	playerPower := atk + hp/10
	bossPower := boss["hp"]/10 + effectiveAtk
	return s.rng.Float64() < float64(playerPower)/float64(playerPower+bossPower)
}
func (s *HeartDemonService) fightWrathDemon(demon *model.PersistentHeartDemon, boss map[string]int64, action string, atk, def, hp int64) bool {
	// 嗔：狂暴计时，3回合
	// action: "fight" = 全力攻击, "defend" = 防御姿态
	playerDmg := atk - boss["def"]/2
	if playerDmg < 0 {
		playerDmg = 1
	}
	bossDmg := boss["atk"] - def/2
	if bossDmg < 0 {
		bossDmg = 1
	}
	if action == "defend" {
		bossDmg = bossDmg / 2
		playerDmg = playerDmg / 2
	}
	bossHP := boss["hp"]
	playerHP := hp
	// 3回合模拟
	for round := 0; round < 3; round++ {
		bossHP -= playerDmg
		if bossHP <= 0 {
			return true
		}
		// 狂暴递增伤害
		roundMultiplier := 1.0 + float64(round)*0.5
		playerHP -= int64(float64(bossDmg) * roundMultiplier)
		if playerHP <= 0 {
			return false
		}
	}
	return bossHP <= 0
}
func (s *HeartDemonService) fightIgnorDemon(demon *model.PersistentHeartDemon, boss map[string]int64, answer string, atk, def, hp int64) bool {
	// 痴：问答机制
	// 正确答案（简化版：随机决定正确答案）
	correctAnswer := fmt.Sprintf("ans%d", s.rng.Intn(4)+1)
	isCorrect := answer == correctAnswer
	playerPower := atk + hp/10
	bossPower := boss["hp"]/10 + boss["atk"]
	if isCorrect {
		// 答对：Boss被削弱
		bossPower = bossPower * 2 / 3
	} else {
		// 答错：Boss回血
		bossPower = bossPower * 4 / 3
	}
	return s.rng.Float64() < float64(playerPower)/float64(playerPower+bossPower)
}
func (s *HeartDemonService) fightDoubtDemon(demon *model.PersistentHeartDemon, boss map[string]int64, choice string, atk, def, hp int64) bool {
	// 疑：分身机制，选对真身才能造成伤害
	targetClone := fmt.Sprintf("clone%d", s.rng.Intn(3)+1)
	isReal := choice == targetClone
	if isReal {
		playerPower := atk + hp/10
		bossPower := boss["hp"]/10 + boss["atk"]
		return s.rng.Float64() < float64(playerPower)/float64(playerPower+bossPower)
	}
	return false
}
func (s *HeartDemonService) fightSlothDemon(demon *model.PersistentHeartDemon, boss map[string]int64, action string, atk, def, hp int64) bool {
	// 慢：行动减慢，需要耐心等待时机
	// action: "wait" = 等待破绽, "rush" = 强行攻击
	playerPower := (atk + hp/10) * 2 / 3 // 速度降低50%
	bossPower := boss["hp"]/10 + boss["atk"]
	winChance := float64(playerPower) / float64(playerPower+bossPower)
	if action == "wait" {
		// 等待破绽增加胜率
		winChance = winChance * 1.3
		if winChance > 0.9 {
			winChance = 0.9
		}
	} else {
		// 强攻降低胜率
		winChance = winChance * 0.7
	}
	return s.rng.Float64() < winChance
}
func (s *HeartDemonService) fightDefault(demon *model.PersistentHeartDemon, boss map[string]int64, atk, def, hp int64) bool {
	playerPower := atk + hp/10
	bossPower := boss["hp"]/10 + boss["atk"]
	return s.rng.Float64() < float64(playerPower)/float64(playerPower+bossPower)
}
// updateDebuffValue 根据等级更新debuff值
func (s *HeartDemonService) updateDebuffValue(demon *model.PersistentHeartDemon) {
	demon.DebuffValue = 0.05 * float64(demon.Level)
	if demon.DebuffValue > 0.50 {
		demon.DebuffValue = 0.50
	}
}
// ---------- 压制道具 ----------
// UseSuppressionItem 使用压制道具
//
//	镇魔符: 随机一个活跃心魔 -2 级
func (s *HeartDemonService) UseSuppressionItem(player *model.Player) (string, string) {
	// 检查是否有活跃心魔
	activeIndices := make([]int, 0)
	for i, d := range player.HeartDemons {
		if !d.Defeated {
			activeIndices = append(activeIndices, i)
		}
	}
	if len(activeIndices) == 0 {
		return "", "没有需要压制的心魔"
	}
	// 检查道具
	if player.SuppressionItems == nil {
		player.SuppressionItems = make(map[string]int)
	}
	if player.SuppressionItems[string(model.ItemSuppressionTalisman)] <= 0 {
		return "", "镇魔符不足"
	}
	// 消耗一个
	player.SuppressionItems[string(model.ItemSuppressionTalisman)]--
	// 随机选择一个活跃心魔
	idx := activeIndices[s.rng.Intn(len(activeIndices))]
	demon := &player.HeartDemons[idx]
	// 降低2级
	demon.Level -= 2
	if demon.Level <= 0 {
		demon.Level = 1 // 最低1级，不会直接消灭
	}
	s.updateDebuffValue(demon)
	s.logger.Info("玩家使用镇魔符", "player_id", player.ID, "demon_type", demon.DemonType, "level_reduction", 2, "new_level", demon.Level)
	return fmt.Sprintf("镇魔符生效，心魔[%s]等级降至Lv.%d", s.GetDemonName(demon.DemonType), demon.Level), ""
}
// UseCleansingPill 使用清心丹
//
//	清心丹: 随机一个活跃心魔 -1 级，若降至0则净化
func (s *HeartDemonService) UseCleansingPill(player *model.Player) (string, string) {
	activeIndices := make([]int, 0)
	for i, d := range player.HeartDemons {
		if !d.Defeated {
			activeIndices = append(activeIndices, i)
		}
	}
	if len(activeIndices) == 0 {
		return "", "没有需要净化的心魔"
	}
	if player.SuppressionItems == nil {
		player.SuppressionItems = make(map[string]int)
	}
	if player.SuppressionItems[string(model.ItemCleansingPill)] <= 0 {
		return "", "清心丹不足"
	}
	player.SuppressionItems[string(model.ItemCleansingPill)]--
	idx := activeIndices[s.rng.Intn(len(activeIndices))]
	demon := &player.HeartDemons[idx]
	demon.Level--
	if demon.Level <= 0 {
		demon.Defeated = true
		demon.DefeatedAt = time.Now().Unix()
		demon.DebuffValue = 0
		s.logger.Info("玩家使用清心丹成功净化心魔", "player_id", player.ID, "demon_type", demon.DemonType)
		return fmt.Sprintf("清心丹生效，心魔[%s]已被净化！", s.GetDemonName(demon.DemonType)), ""
	}
	s.updateDebuffValue(demon)
	s.logger.Info("玩家使用清心丹", "player_id", player.ID, "demon_type", demon.DemonType, "level_reduction", 1, "new_level", demon.Level)
	return fmt.Sprintf("清心丹生效，心魔[%s]等级降至Lv.%d", s.GetDemonName(demon.DemonType), demon.Level), ""
}
// LearnBodhiTechnique 学习菩提心法（被动防护）
func (s *HeartDemonService) LearnBodhiTechnique(player *model.Player) {
	player.HasBodhiTechnique = true
	s.logger.Info("玩家习得菩提心法", "player_id", player.ID)
}
// ---------- 查询 ----------
// GetActiveDemons 获取玩家活跃心魔列表
func (s *HeartDemonService) GetActiveDemons(player *model.Player) []model.PersistentHeartDemon {
	active := make([]model.PersistentHeartDemon, 0)
	for _, d := range player.HeartDemons {
		if !d.Defeated {
			active = append(active, d)
		}
	}
	// 按等级降序排列
	sort.Slice(active, func(i, j int) bool {
		return active[i].Level > active[j].Level
	})
	return active
}
// GetDemonHistory 获取心魔历史记录
func (s *HeartDemonService) GetDemonHistory(player *model.Player) []model.PersistentHeartDemon {
	// 返回全部心魔（包括已净化的）
	sorted := make([]model.PersistentHeartDemon, len(player.HeartDemons))
	copy(sorted, player.HeartDemons)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt > sorted[j].CreatedAt
	})
	return sorted
}
// ---------- 辅助 ----------
func (s *HeartDemonService) GetDemonName(demonType model.PersistentDemonType) string {
	names := map[model.PersistentDemonType]string{
		model.DemonGreed:     "贪欲之魔",
		model.DemonWrath:     "嗔怒之魔",
		model.DemonIgnorance: "痴念之魔",
		model.DemonDoubt:     "疑虑之魔",
		model.DemonSloth:     "傲慢之魔",
	}
	return names[demonType]
}
func (s *HeartDemonService) getDemonIcon(demonType model.PersistentDemonType) string {
	icons := map[model.PersistentDemonType]string{
		model.DemonGreed:     "💰",
		model.DemonWrath:     "🔥",
		model.DemonIgnorance: "🧠",
		model.DemonDoubt:     "👻",
		model.DemonSloth:     "🐌",
	}
	return icons[demonType]
}
func (s *HeartDemonService) getDemonColor(demonType model.PersistentDemonType) string {
	colors := map[model.PersistentDemonType]string{
		model.DemonGreed:     "#ffd700",
		model.DemonWrath:     "#ff5722",
		model.DemonIgnorance: "#6495ed",
		model.DemonDoubt:     "#9c27b0",
		model.DemonSloth:     "#4caf50",
	}
	return colors[demonType]
}
func (s *HeartDemonService) getDemonElement(demonType model.PersistentDemonType) string {
	elements := map[model.PersistentDemonType]string{
		model.DemonGreed:     "金",
		model.DemonWrath:     "火",
		model.DemonIgnorance: "水",
		model.DemonDoubt:     "木",
		model.DemonSloth:     "土",
	}
	return elements[demonType]
}
func (s *HeartDemonService) getIllusionDescription(demonType model.PersistentDemonType, level int) string {
	descriptions := map[model.PersistentDemonType]string{
		model.DemonGreed:     "幻境中堆满天材地宝，贪欲之魔诱惑你放弃道心换取宝物。",
		model.DemonWrath:     "幻境中怒火焚天，嗔怒之魔以狂暴之势向你袭来。",
		model.DemonIgnorance: "幻境中迷雾重重，痴念之魔设下修行谜题困住你的心神。",
		model.DemonDoubt:     "幻境中光影交错，疑虑之魔化出无数分身让你无法分辨。",
		model.DemonSloth:     "幻境中时间凝滞，傲慢之魔以逸待劳等待你露出破绽。",
	}
	return descriptions[demonType]
}
func (s *HeartDemonService) getIllusionSpecial(demonType model.PersistentDemonType) map[string]interface{} {
	specials := map[model.PersistentDemonType]map[string]interface{}{
		model.DemonGreed: {
			"mechanic":  "sacrifice",
			"desc":      "心魔会偷取你的灵石，选择'放弃灵石'可降低其攻击力",
			"options":   []string{"fight", "sacrifice"},
		},
		model.DemonWrath: {
			"mechanic":  "enrage",
			"desc":      "心魔进入狂暴状态，每回合攻击递增50%，必须在3回合内击败",
			"options":   []string{"fight", "defend"},
		},
		model.DemonIgnorance: {
			"mechanic":  "quiz",
			"desc":      "需回答修行问题，答错心魔恢复大量HP",
			"options":   []string{"ans1", "ans2", "ans3", "ans4"},
		},
		model.DemonDoubt: {
			"mechanic":  "clone",
			"desc":      "心魔制造3个分身，只有真身受到伤害",
			"options":   []string{"clone1", "clone2", "clone3"},
		},
		model.DemonSloth: {
			"mechanic":  "patience",
			"desc":      "行动速度降低50%，等待破绽可提高命中率",
			"options":   []string{"wait", "rush"},
		},
	}
	return specials[demonType]
}
// GetIllusionQuestions 获取痴心魔的问答题目
func (s *HeartDemonService) GetIllusionQuestions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"question": "修炼的最终目的是什么？",
			"choices":  []string{"追求长生", "超脱轮回", "护佑苍生", "追求力量巅峰"},
			"hint":     "道法自然，超脱轮回方为正道",
		},
		{
			"question": "何为道心？",
			"choices":  []string{"坚定不移的信念", "强大的修为", "无上的法宝", "众多的追随者"},
			"hint":     "道心即本心，坚定不移的求道之心",
		},
		{
			"question": "渡劫时最重要的什么？",
			"choices":  []string{"强大的法宝", "深厚的修为", "坚定的意志", "道友的护法"},
			"hint":     "外物终是虚妄，唯有意志坚定才能渡过天劫",
		},
	}
}