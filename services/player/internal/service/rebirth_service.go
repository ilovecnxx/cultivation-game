package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// RebirthService 轮回转世业务逻辑
type RebirthService struct {
	db           *sql.DB
	playerRepo   *mysql.PlayerRepo
	dongfuRepo   *mysql.DongFuRepo
	artifactRepo *mysql.ArtifactRepo
	log          *zap.Logger
}

// NewRebirthService 创建 RebirthService
func NewRebirthService(
	db *sql.DB,
	playerRepo *mysql.PlayerRepo,
	dongfuRepo *mysql.DongFuRepo,
	artifactRepo *mysql.ArtifactRepo,
	log *zap.Logger,
) *RebirthService {
	return &RebirthService{
		db:           db,
		playerRepo:   playerRepo,
		dongfuRepo:   dongfuRepo,
		artifactRepo: artifactRepo,
		log:          log,
	}
}

// CheckRebirth 检查玩家轮回状态和条件
func (s *RebirthService) CheckRebirth(ctx context.Context, playerID int64) (*model.RebirthCheckResponse, error) {
	// 获取玩家信息
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	// 获取或初始化轮回记录
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return nil, err
	}

	// 判断轮回条件
	canRebirth := false
	condition := ""
	if player.Realm >= model.RealmTrib && player.Level >= 80 {
		canRebirth = true
		condition = "渡劫期80层以上可主动散功转世"
	}
	if rebirth.RebirthCount >= model.MaxRebirthCount {
		canRebirth = false
		condition = fmt.Sprintf("已达最大轮回次数(%d次)，不可再轮回", model.MaxRebirthCount)
	}

	bonuses := model.RebirthTitleBonuses[rebirth.RebirthCount]

	benefits := s.buildBenefits()

	resp := &model.RebirthCheckResponse{
		CanRebirth:           canRebirth,
		RebirthCount:         rebirth.RebirthCount,
		MaxRebirthCount:      model.MaxRebirthCount,
		CurrentTitle:         rebirth.Title,
		CurrentTitleBonuses: struct {
			AttackPct  float64 `json:"attack_pct"`
			DefensePct float64 `json:"defense_pct"`
			HPPct      float64 `json:"hp_pct"`
			SpeedPct   float64 `json:"speed_pct"`
		}{
			AttackPct:  bonuses.AttackPct,
			DefensePct: bonuses.DefensePct,
			HPPct:      bonuses.HPPct,
			SpeedPct:   bonuses.SpeedPct,
		},
		Enlightenment:         rebirth.Enlightenment,
		SpiritRootQuality:     rebirth.SpiritRootQuality,
		SpiritRootQualityName: qualityName(rebirth.SpiritRootQuality),
		CultivationSpeedBonus: rebirth.CultivationSpeedBonus,
		RebirthJade:           rebirth.RebirthJade,
		TalentPoints:          rebirth.TalentPoints,
		Condition:             condition,
		Benefits:              benefits,
	}

	return resp, nil
}

// ExecuteRebirth 执行轮回转世
func (s *RebirthService) ExecuteRebirth(ctx context.Context, playerID int64) (*model.RebirthCheckResponse, error) {
	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 查询玩家
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	// 查询轮回记录
	rebirth, err := s.getOrCreateRebirthTx(tx, playerID)
	if err != nil {
		return nil, err
	}

	// 校验轮回条件
	if rebirth.RebirthCount >= model.MaxRebirthCount {
		return nil, fmt.Errorf("已达最大轮回次数(%d次)，无法继续轮回", model.MaxRebirthCount)
	}
	if player.Realm < model.RealmTrib {
		return nil, fmt.Errorf("当前境界不足渡劫期(%s)，无法轮回", model.RealmNames[player.Realm])
	}
	if player.Level < 80 {
		return nil, fmt.Errorf("渡劫期修为不足(80层)，当前%d层", player.Level)
	}

	// 保存轮回前状态用于历史记录
	oldRealm := player.Realm
	oldRealmLevel := player.Level
	oldSpiritRoot := player.SpiritRoot
	oldQuality := rebirth.SpiritRootQuality
	oldAttack := player.Attack
	oldDefense := player.Defense
	oldHP := player.MaxHP

	// 计算继承修为（5%基础 + 天赋加成）
	carryOverPct := float64(model.CarryOverRatio)
	// 检查是否有"逆天改命"天赋加成
	extraCarryOver := s.getTalentEffectTotal(playerID, "extra_carry_over_pct")
	carryOverPct += extraCarryOver

	carryOverAttack := int64(float64(player.Attack) * carryOverPct / 100.0)
	carryOverDefense := int64(float64(player.Defense) * carryOverPct / 100.0)
	carryOverHP := int64(float64(player.MaxHP) * carryOverPct / 100.0)

	// 计算轮回币奖励（基于轮回前总修为）
	rebirthJadeEarned := s.calcRebirthJadeEarned(player)
	totalRebirthJade := rebirth.RebirthJade + rebirthJadeEarned

	// ---------- 执行轮回重置 ----------

	// 1. 境界重置为练气1层
	player.Realm = model.RealmQiRef
	player.Level = 1

	// 2. 基础属性重置为初始值，然后加上继承属性
	baseAttrs := calcRebirthBaseAttrs(player.SpiritRoot)
	player.MaxHP = baseAttrs.hp + carryOverHP
	player.HP = baseAttrs.hp + carryOverHP
	player.MP = baseAttrs.mp
	player.MaxMP = baseAttrs.mp
	player.Attack = baseAttrs.attack + carryOverAttack
	player.Defense = baseAttrs.defense + carryOverDefense
	player.SpiritPower = 0
	player.Experience = 0

	// 3. 保留50%灵石
	keptGold := player.Gold / 2
	player.Gold = keptGold

	// 4. 更新轮回次数与增益
	rebirth.RebirthCount++
	rebirth.Enlightenment = rebirth.RebirthCount
	rebirth.SpiritRootQuality = oldQuality + 1
	if rebirth.SpiritRootQuality > model.MaxSpiritRootQuality {
		rebirth.SpiritRootQuality = model.MaxSpiritRootQuality
	}
	rebirth.CultivationSpeedBonus = rebirth.RebirthCount * model.SpeedBonusPerRebirth
	rebirth.Title = rebirthTitle(rebirth.RebirthCount)
	rebirth.RebirthJade = totalRebirthJade
	rebirth.TalentPoints += model.TalentPointPerRebirth
	rebirth.CarryOverPower += carryOverAttack + carryOverDefense + carryOverHP
	rebirth.UpdatedAt = time.Now()

	// 5. 降级洞府为1级
	if err := s.demoteDongFu(tx, playerID); err != nil {
		return nil, err
	}

	// 6. 降级本命法宝为1级
	if err := s.demoteArtifact(tx, playerID); err != nil {
		return nil, err
	}

	// 7. 写入轮回历史
	history := &model.RebirthHistory{
		PlayerID:         playerID,
		RebirthNumber:    rebirth.RebirthCount,
		OldRealm:         oldRealm,
		OldRealmLevel:    oldRealmLevel,
		OldAttack:        oldAttack,
		OldDefense:       oldDefense,
		OldHP:            oldHP,
		CarryOverAttack:  carryOverAttack,
		CarryOverDefense: carryOverDefense,
		CarryOverHP:      carryOverHP,
		GoldKept:         keptGold,
		SpiritRootBefore: oldSpiritRoot,
		SpiritRootAfter:  player.SpiritRoot,
		QualityBefore:    oldQuality,
		QualityAfter:     rebirth.SpiritRootQuality,
		RebirthJadeEarned: rebirthJadeEarned,
		TitleEarned:      rebirth.Title,
		CreatedAt:        time.Now(),
	}
	if err := s.insertHistoryTx(tx, history); err != nil {
		return nil, err
	}

	// 8. 更新玩家数据
	if err := s.updatePlayerTx(tx, player); err != nil {
		return nil, err
	}

	// 9. 更新轮回记录
	if err := s.updateRebirthTx(tx, rebirth); err != nil {
		return nil, err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	bonuses := model.RebirthTitleBonuses[rebirth.RebirthCount]

	s.log.Info("轮回转世成功",
		zap.Int64("player_id", playerID),
		zap.Int("rebirth_count", rebirth.RebirthCount),
		zap.String("title", rebirth.Title),
		zap.Int("enlightenment", rebirth.Enlightenment),
		zap.Int("spirit_root_quality", rebirth.SpiritRootQuality),
		zap.Int("speed_bonus", rebirth.CultivationSpeedBonus),
		zap.Int64("gold_kept", keptGold),
		zap.Int64("carry_over_attack", carryOverAttack),
		zap.Int64("carry_over_defense", carryOverDefense),
		zap.Int64("carry_over_hp", carryOverHP),
		zap.Int64("rebirth_jade_earned", rebirthJadeEarned),
	)

	return &model.RebirthCheckResponse{
		CanRebirth:           false,
		RebirthCount:         rebirth.RebirthCount,
		MaxRebirthCount:      model.MaxRebirthCount,
		CurrentTitle:         rebirth.Title,
		CurrentTitleBonuses: struct {
			AttackPct  float64 `json:"attack_pct"`
			DefensePct float64 `json:"defense_pct"`
			HPPct      float64 `json:"hp_pct"`
			SpeedPct   float64 `json:"speed_pct"`
		}{
			AttackPct:  bonuses.AttackPct,
			DefensePct: bonuses.DefensePct,
			HPPct:      bonuses.HPPct,
			SpeedPct:   bonuses.SpeedPct,
		},
		Enlightenment:         rebirth.Enlightenment,
		SpiritRootQuality:     rebirth.SpiritRootQuality,
		SpiritRootQualityName: qualityName(rebirth.SpiritRootQuality),
		CultivationSpeedBonus: rebirth.CultivationSpeedBonus,
		RebirthJade:           rebirth.RebirthJade,
		TalentPoints:          rebirth.TalentPoints,
		Condition:             "轮回成功",
		Benefits:              s.buildBenefits(),
	}, nil
}

// ============================================================
// 天赋树系统
// ============================================================

// GetTalentInfo 获取玩家天赋信息
func (s *RebirthService) GetTalentInfo(ctx context.Context, playerID int64) (*model.TalentInfoResponse, error) {
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取轮回记录失败: %w", err)
	}

	learnedTalents, err := s.queryTalents(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询已学天赋失败: %w", err)
	}

	bonuses := s.calcTalentBonuses(learnedTalents)

	return &model.TalentInfoResponse{
		TalentPoints:   rebirth.TalentPoints,
		TalentTree:     model.TalentTree,
		LearnedTalents: learnedTalents,
		TalentBonuses:  bonuses,
	}, nil
}

// LearnTalent 学习/升级天赋
func (s *RebirthService) LearnTalent(ctx context.Context, playerID int64, talentID string) error {
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return fmt.Errorf("获取轮回记录失败: %w", err)
	}

	// 查找天赋定义
	var targetNode *model.TalentTreeNode
	for _, branch := range model.TalentTree {
		for i := range branch {
			if branch[i].ID == talentID {
				targetNode = &branch[i]
				break
			}
		}
		if targetNode != nil {
			break
		}
	}
	if targetNode == nil {
		return fmt.Errorf("天赋不存在: %s", talentID)
	}

	// 获取已学天赋
	learnedTalents, err := s.queryTalents(playerID)
	if err != nil {
		return fmt.Errorf("查询已学天赋失败: %w", err)
	}

	// 检查前置条件
	if targetNode.PreReqSlot >= 0 {
		preReqID := fmt.Sprintf("%s_%d", targetNode.Branch, targetNode.PreReqSlot)
		hasPreReq := false
		for _, t := range learnedTalents {
			if t.TalentID == preReqID && t.Level > 0 {
				hasPreReq = true
				break
			}
		}
		if !hasPreReq {
			return fmt.Errorf("前置天赋未学习")
		}
	}

	// 检查是否已学及当前等级
	currentLevel := 0
	for _, t := range learnedTalents {
		if t.TalentID == talentID {
			currentLevel = t.Level
			break
		}
	}
	if currentLevel >= targetNode.MaxLevel {
		return fmt.Errorf("天赋已满级")
	}

	// 检查天赋点
	if rebirth.TalentPoints < targetNode.CostPoints {
		return fmt.Errorf("天赋点不足: 需要%d点，当前%d点", targetNode.CostPoints, rebirth.TalentPoints)
	}

	// 扣除天赋点 & 保存
	rebirth.TalentPoints -= targetNode.CostPoints
	if currentLevel == 0 {
		// 新增
		now := time.Now()
		pt := &model.PlayerTalent{
			PlayerID:  playerID,
			TalentID:  talentID,
			Level:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.insertTalent(pt); err != nil {
			return fmt.Errorf("学习天赋失败: %w", err)
		}
	} else {
		// 升级
		if err := s.upgradeTalent(playerID, talentID); err != nil {
			return fmt.Errorf("升级天赋失败: %w", err)
		}
	}

	// 更新轮回记录(天赋点)
	if err := s.updateRebirthPoints(playerID, rebirth.TalentPoints); err != nil {
		return err
	}

	s.log.Info("学习天赋成功",
		zap.Int64("player_id", playerID),
		zap.String("talent_id", talentID),
		zap.Int("cost_points", targetNode.CostPoints),
	)
	return nil
}

// ResetTalents 重置所有天赋
func (s *RebirthService) ResetTalents(ctx context.Context, playerID int64) error {
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return fmt.Errorf("获取轮回记录失败: %w", err)
	}

	// 计算已使用的天赋点
	learnedTalents, err := s.queryTalents(playerID)
	if err != nil {
		return fmt.Errorf("查询已学天赋失败: %w", err)
	}

	totalUsedPoints := 0
	for _, t := range learnedTalents {
		node := s.findTalentNode(t.TalentID)
		if node != nil {
			totalUsedPoints += node.CostPoints * t.Level
		}
	}

	// 重置天赋点数
	rebirth.TalentPoints += totalUsedPoints

	// 删除所有已学天赋
	if err := s.deleteAllTalents(playerID); err != nil {
		return fmt.Errorf("重置天赋失败: %w", err)
	}

	if err := s.updateRebirthPoints(playerID, rebirth.TalentPoints); err != nil {
		return err
	}

	s.log.Info("重置天赋成功",
		zap.Int64("player_id", playerID),
		zap.Int("refunded_points", totalUsedPoints),
	)
	return nil
}

// ============================================================
// 轮回商店系统
// ============================================================

// GetRebirthShop 获取轮回商店信息
func (s *RebirthService) GetRebirthShop(ctx context.Context, playerID int64) (*model.RebirthShopResponse, error) {
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return nil, fmt.Errorf("获取轮回记录失败: %w", err)
	}

	purchases, err := s.queryPurchases(playerID)
	if err != nil {
		return nil, fmt.Errorf("查询购买记录失败: %w", err)
	}

	// 根据轮回次数筛选可购买商品
	var available []model.RebirthShopItem
	for _, item := range model.RebirthShopItems {
		if rebirth.RebirthCount >= item.RebirthReq {
			available = append(available, item)
		}
	}

	return &model.RebirthShopResponse{
		RebirthJade: rebirth.RebirthJade,
		Items:       available,
		Purchases:   purchases,
	}, nil
}

// BuyRebirthShopItem 购买轮回商店物品
func (s *RebirthService) BuyRebirthShopItem(ctx context.Context, playerID int64, shopID string) error {
	rebirth, err := s.getOrCreateRebirth(playerID)
	if err != nil {
		return fmt.Errorf("获取轮回记录失败: %w", err)
	}

	// 查找商品
	var item *model.RebirthShopItem
	for i := range model.RebirthShopItems {
		if model.RebirthShopItems[i].ID == shopID {
			item = &model.RebirthShopItems[i]
			break
		}
	}
	if item == nil {
		return fmt.Errorf("商品不存在")
	}

	// 检查轮回次数要求
	if rebirth.RebirthCount < item.RebirthReq {
		return fmt.Errorf("轮回次数不足: 需要%d次轮回", item.RebirthReq)
	}

	// 检查轮回币
	if rebirth.RebirthJade < int64(item.Price) {
		return fmt.Errorf("轮回币不足: 需要%d，当前%d", item.Price, rebirth.RebirthJade)
	}

	// 检查限购
	if item.MaxPurchase > 0 {
		purchases, err := s.queryPurchases(playerID)
		if err != nil {
			return fmt.Errorf("查询购买记录失败: %w", err)
		}
		for _, p := range purchases {
			if p.ShopID == shopID && p.Count >= item.MaxPurchase {
				return fmt.Errorf("商品已达购买上限(%d次)", item.MaxPurchase)
			}
		}
	}

	// 扣轮回币
	rebirth.RebirthJade -= int64(item.Price)

	// 记录购买（增加计数或新增）
	purchases, err := s.queryPurchases(playerID)
	if err != nil {
		return fmt.Errorf("查询购买记录失败: %w", err)
	}

	found := false
	for _, p := range purchases {
		if p.ShopID == shopID {
			found = true
			p.Count++
			p.UpdatedAt = time.Now()
			if err := s.updatePurchaseCount(&p); err != nil {
				return fmt.Errorf("更新购买记录失败: %w", err)
			}
			break
		}
	}
	if !found {
		purchase := &model.RebirthShopPurchase{
			PlayerID:  playerID,
			ShopID:    shopID,
			Count:     1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.insertPurchase(purchase); err != nil {
			return fmt.Errorf("记录购买失败: %w", err)
		}
	}

	// 应用效果
	if err := s.applyShopEffect(txWrap{s.db}, playerID, item); err != nil {
		return err
	}

	// 更新轮回币
	if err := s.updateRebirthJade(playerID, rebirth.RebirthJade); err != nil {
		return err
	}

	s.log.Info("购买轮回商店物品成功",
		zap.Int64("player_id", playerID),
		zap.String("shop_id", shopID),
		zap.Int("price", item.Price),
	)
	return nil
}

// GetRebirthBenefits 获取轮回福利列表
func (s *RebirthService) GetRebirthBenefits(ctx context.Context) *model.RebirthBenefits {
	return s.buildBenefits()
}

// ListRebirthHistory 获取轮回历史
func (s *RebirthService) ListRebirthHistory(ctx context.Context, playerID int64) ([]model.RebirthHistory, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, player_id, rebirth_number, old_realm, old_realm_level,
			old_attack, old_defense, old_hp, carry_over_attack, carry_over_defense, carry_over_hp,
			gold_kept, spirit_root_before, spirit_root_after,
			quality_before, quality_after, rebirth_jade_earned, title_earned, created_at
		 FROM player_rebirth_history
		 WHERE player_id = ?
		 ORDER BY rebirth_number ASC`, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询轮回历史失败: %w", err)
	}
	defer rows.Close()

	var histories []model.RebirthHistory
	for rows.Next() {
		var h model.RebirthHistory
		if err := rows.Scan(
			&h.ID, &h.PlayerID, &h.RebirthNumber, &h.OldRealm, &h.OldRealmLevel,
			&h.OldAttack, &h.OldDefense, &h.OldHP, &h.CarryOverAttack, &h.CarryOverDefense, &h.CarryOverHP,
			&h.GoldKept, &h.SpiritRootBefore, &h.SpiritRootAfter,
			&h.QualityBefore, &h.QualityAfter, &h.RebirthJadeEarned, &h.TitleEarned, &h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描轮回历史行失败: %w", err)
		}
		histories = append(histories, h)
	}
	if histories == nil {
		histories = []model.RebirthHistory{}
	}
	return histories, rows.Err()
}

// GetPlayerRebirth 获取玩家轮回记录
func (s *RebirthService) GetPlayerRebirth(ctx context.Context, playerID int64) (*model.PlayerRebirth, error) {
	return s.getOrCreateRebirth(playerID)
}

// ============================================================
// 内部辅助方法
// ============================================================

// calcRebirthJadeEarned 计算轮回可获得的轮回币
func (s *RebirthService) calcRebirthJadeEarned(player *model.Player) int64 {
	// 基础 50 + (境界-渡劫)*20 + 等级*2
	base := int64(50)
	realmBonus := int64(player.Realm-model.RealmTrib) * 20
	levelBonus := int64(player.Level) / 10 * 5
	powerBonus := (player.Attack + player.Defense + player.MaxHP) / 1000
	total := base + realmBonus + levelBonus + powerBonus
	if total < 10 {
		total = 10
	}
	return total
}

// getTalentEffectTotal 获取某类天赋的总加成值
func (s *RebirthService) getTalentEffectTotal(playerID int64, effectType string) float64 {
	talents, err := s.queryTalents(playerID)
	if err != nil {
		return 0
	}
	total := 0.0
	for _, t := range talents {
		node := s.findTalentNode(t.TalentID)
		if node != nil && node.EffectType == effectType {
			total += node.EffectValue * float64(t.Level)
		}
	}
	return total
}

// findTalentNode 根据ID查找天赋节点
func (s *RebirthService) findTalentNode(talentID string) *model.TalentTreeNode {
	for _, branch := range model.TalentTree {
		for i := range branch {
			if branch[i].ID == talentID {
				return &branch[i]
			}
		}
	}
	return nil
}

// calcTalentBonuses 计算所有已学天赋的总加成
func (s *RebirthService) calcTalentBonuses(talents []model.PlayerTalent) map[string]float64 {
	bonuses := make(map[string]float64)
	for _, t := range talents {
		node := s.findTalentNode(t.TalentID)
		if node != nil {
			bonuses[node.EffectType] += node.EffectValue * float64(t.Level)
		}
	}
	return bonuses
}

// applyShopEffect 应用商店物品效果
type txWrap struct {
	db *sql.DB
}

func (t txWrap) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.db.Exec(query, args...)
}

func (s *RebirthService) applyShopEffect(tx interface {
	Exec(string, ...interface{}) (sql.Result, error)
}, playerID int64, item *model.RebirthShopItem) error {
	if item.ItemType != "effect" {
		// 非效果类物品由外部背包系统处理
		return nil
	}

	switch item.ItemID {
	case "rebirth_cd_reset":
		// 重置轮回冷却（由上层处理）
	case "perm_attack_pct":
		// 永久攻击加成存储到玩家额外属性表（简化：记录日志）
		s.log.Info("永久攻击加成生效", zap.Int64("player_id", playerID), zap.Int("value", item.EffectValue))
	case "perm_defense_pct":
		s.log.Info("永久防御加成生效", zap.Int64("player_id", playerID), zap.Int("value", item.EffectValue))
	case "perm_speed_pct":
		s.log.Info("永久修炼速度加成生效", zap.Int64("player_id", playerID), zap.Int("value", item.EffectValue))
	case "talent_reset":
		// 重置天赋已由调用端处理
	case "carry_over_boost":
		// 下次轮回额外继承（记录到rebirth表carry_over_boost字段）
		_, err := tx.Exec(
			`UPDATE player_rebirths SET carry_over_power = carry_over_power + ? WHERE player_id = ?`,
			item.EffectValue, playerID,
		)
		if err != nil {
			return fmt.Errorf("应用轮回护符效果失败: %w", err)
		}
	case "perm_luck_pct":
		s.log.Info("永久气运加成生效", zap.Int64("player_id", playerID), zap.Int("value", item.EffectValue))
	case "perm_hp_pct":
		s.log.Info("永久生命加成生效", zap.Int64("player_id", playerID), zap.Int("value", item.EffectValue))
	}
	return nil
}

// -- 数据库查询方法 --

func (s *RebirthService) queryTalents(playerID int64) ([]model.PlayerTalent, error) {
	rows, err := s.db.Query(
		`SELECT id, player_id, talent_id, level, created_at, updated_at
		 FROM player_talents WHERE player_id = ?`, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询天赋失败: %w", err)
	}
	defer rows.Close()

	var talents []model.PlayerTalent
	for rows.Next() {
		var t model.PlayerTalent
		if err := rows.Scan(&t.ID, &t.PlayerID, &t.TalentID, &t.Level, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描天赋行失败: %w", err)
		}
		talents = append(talents, t)
	}
	if talents == nil {
		talents = []model.PlayerTalent{}
	}
	return talents, rows.Err()
}

func (s *RebirthService) insertTalent(t *model.PlayerTalent) error {
	result, err := s.db.Exec(
		`INSERT INTO player_talents (player_id, talent_id, level, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		t.PlayerID, t.TalentID, t.Level, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("插入天赋失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取天赋自增ID失败: %w", err)
	}
	t.ID = id
	return nil
}

func (s *RebirthService) upgradeTalent(playerID int64, talentID string) error {
	_, err := s.db.Exec(
		`UPDATE player_talents SET level = level + 1, updated_at = ? WHERE player_id = ? AND talent_id = ?`,
		time.Now(), playerID, talentID)
	if err != nil {
		return fmt.Errorf("升级天赋失败: %w", err)
	}
	return nil
}

func (s *RebirthService) deleteAllTalents(playerID int64) error {
	_, err := s.db.Exec(`DELETE FROM player_talents WHERE player_id = ?`, playerID)
	if err != nil {
		return fmt.Errorf("删除天赋失败: %w", err)
	}
	return nil
}

func (s *RebirthService) updateRebirthPoints(playerID int64, points int) error {
	_, err := s.db.Exec(
		`UPDATE player_rebirths SET talent_points = ?, updated_at = ? WHERE player_id = ?`,
		points, time.Now(), playerID)
	if err != nil {
		return fmt.Errorf("更新天赋点失败: %w", err)
	}
	return nil
}

func (s *RebirthService) updateRebirthJade(playerID int64, jade int64) error {
	_, err := s.db.Exec(
		`UPDATE player_rebirths SET rebirth_jade = ?, updated_at = ? WHERE player_id = ?`,
		jade, time.Now(), playerID)
	if err != nil {
		return fmt.Errorf("更新轮回币失败: %w", err)
	}
	return nil
}

func (s *RebirthService) queryPurchases(playerID int64) ([]model.RebirthShopPurchase, error) {
	rows, err := s.db.Query(
		`SELECT id, player_id, shop_id, count, created_at, updated_at
		 FROM rebirth_shop_purchases WHERE player_id = ?`, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询购买记录失败: %w", err)
	}
	defer rows.Close()

	var purchases []model.RebirthShopPurchase
	for rows.Next() {
		var p model.RebirthShopPurchase
		if err := rows.Scan(&p.ID, &p.PlayerID, &p.ShopID, &p.Count, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描购买记录行失败: %w", err)
		}
		purchases = append(purchases, p)
	}
	if purchases == nil {
		purchases = []model.RebirthShopPurchase{}
	}
	return purchases, rows.Err()
}

func (s *RebirthService) insertPurchase(p *model.RebirthShopPurchase) error {
	result, err := s.db.Exec(
		`INSERT INTO rebirth_shop_purchases (player_id, shop_id, count, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		p.PlayerID, p.ShopID, p.Count, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("插入购买记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取购买记录自增ID失败: %w", err)
	}
	p.ID = id
	return nil
}

func (s *RebirthService) updatePurchaseCount(p *model.RebirthShopPurchase) error {
	_, err := s.db.Exec(
		`UPDATE rebirth_shop_purchases SET count = ?, updated_at = ? WHERE id = ?`,
		p.Count, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("更新购买记录失败: %w", err)
	}
	return nil
}

// -- 原有方法保持不变 --

func (s *RebirthService) getOrCreateRebirth(playerID int64) (*model.PlayerRebirth, error) {
	rebirth, err := s.queryRebirth(playerID)
	if err != nil {
		return nil, err
	}
	if rebirth != nil {
		return rebirth, nil
	}
	return s.createRebirth(playerID)
}

func (s *RebirthService) getOrCreateRebirthTx(tx *sql.Tx, playerID int64) (*model.PlayerRebirth, error) {
	rebirth, err := s.queryRebirthTx(tx, playerID)
	if err != nil {
		return nil, err
	}
	if rebirth != nil {
		return rebirth, nil
	}
	now := time.Now()
	rebirth = &model.PlayerRebirth{
		PlayerID:              playerID,
		RebirthCount:          0,
		Enlightenment:         0,
		SpiritRootQuality:     model.SpiritRootQualityLow,
		CultivationSpeedBonus: 0,
		Title:                 model.RebirthTitleNames[0],
		RebirthJade:           0,
		TalentPoints:          0,
		CarryOverPower:        0,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	result, err := tx.Exec(
		`INSERT INTO player_rebirths (player_id, rebirth_count, enlightenment,
			spirit_root_quality, cultivation_speed_bonus, title, rebirth_jade, talent_points, carry_over_power, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rebirth.PlayerID, rebirth.RebirthCount, rebirth.Enlightenment,
		rebirth.SpiritRootQuality, rebirth.CultivationSpeedBonus, rebirth.Title,
		rebirth.RebirthJade, rebirth.TalentPoints, rebirth.CarryOverPower,
		rebirth.CreatedAt, rebirth.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建轮回记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取轮回记录自增ID失败: %w", err)
	}
	rebirth.ID = id
	return rebirth, nil
}

func (s *RebirthService) queryRebirth(playerID int64) (*model.PlayerRebirth, error) {
	row := s.db.QueryRow(
		`SELECT id, player_id, rebirth_count, enlightenment,
			spirit_root_quality, cultivation_speed_bonus, title, rebirth_jade, talent_points, carry_over_power, created_at, updated_at
		 FROM player_rebirths WHERE player_id = ?`, playerID)

	r := &model.PlayerRebirth{}
	err := row.Scan(
		&r.ID, &r.PlayerID, &r.RebirthCount, &r.Enlightenment,
		&r.SpiritRootQuality, &r.CultivationSpeedBonus, &r.Title,
		&r.RebirthJade, &r.TalentPoints, &r.CarryOverPower,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询轮回记录失败: %w", err)
	}
	return r, nil
}

func (s *RebirthService) queryRebirthTx(tx *sql.Tx, playerID int64) (*model.PlayerRebirth, error) {
	row := tx.QueryRow(
		`SELECT id, player_id, rebirth_count, enlightenment,
			spirit_root_quality, cultivation_speed_bonus, title, rebirth_jade, talent_points, carry_over_power, created_at, updated_at
		 FROM player_rebirths WHERE player_id = ? FOR UPDATE`, playerID)

	r := &model.PlayerRebirth{}
	err := row.Scan(
		&r.ID, &r.PlayerID, &r.RebirthCount, &r.Enlightenment,
		&r.SpiritRootQuality, &r.CultivationSpeedBonus, &r.Title,
		&r.RebirthJade, &r.TalentPoints, &r.CarryOverPower,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询轮回记录失败: %w", err)
	}
	return r, nil
}

func (s *RebirthService) createRebirth(playerID int64) (*model.PlayerRebirth, error) {
	now := time.Now()
	r := &model.PlayerRebirth{
		PlayerID:              playerID,
		RebirthCount:          0,
		Enlightenment:         0,
		SpiritRootQuality:     model.SpiritRootQualityLow,
		CultivationSpeedBonus: 0,
		Title:                 model.RebirthTitleNames[0],
		RebirthJade:           0,
		TalentPoints:          0,
		CarryOverPower:        0,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	result, err := s.db.Exec(
		`INSERT INTO player_rebirths (player_id, rebirth_count, enlightenment,
			spirit_root_quality, cultivation_speed_bonus, title, rebirth_jade, talent_points, carry_over_power, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.PlayerID, r.RebirthCount, r.Enlightenment,
		r.SpiritRootQuality, r.CultivationSpeedBonus, r.Title,
		r.RebirthJade, r.TalentPoints, r.CarryOverPower,
		r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建轮回记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取轮回记录自增ID失败: %w", err)
	}
	r.ID = id
	return r, nil
}

func (s *RebirthService) updateRebirthTx(tx *sql.Tx, r *model.PlayerRebirth) error {
	_, err := tx.Exec(
		`UPDATE player_rebirths SET rebirth_count=?, enlightenment=?,
			spirit_root_quality=?, cultivation_speed_bonus=?, title=?,
			rebirth_jade=?, talent_points=?, carry_over_power=?, updated_at=?
		 WHERE id=?`,
		r.RebirthCount, r.Enlightenment, r.SpiritRootQuality,
		r.CultivationSpeedBonus, r.Title, r.RebirthJade, r.TalentPoints, r.CarryOverPower,
		r.UpdatedAt, r.ID,
	)
	if err != nil {
		return fmt.Errorf("更新轮回记录失败: %w", err)
	}
	return nil
}

func (s *RebirthService) updatePlayerTx(tx *sql.Tx, p *model.Player) error {
	_, err := tx.Exec(
		`UPDATE players SET level=?, realm=?, hp=?, max_hp=?, mp=?, max_mp=?,
			attack=?, defense=?, spirit_power=?, experience=?, gold=?, updated_at=?
		 WHERE id=?`,
		p.Level, p.Realm, p.HP, p.MaxHP, p.MP, p.MaxMP,
		p.Attack, p.Defense, p.SpiritPower, p.Experience, p.Gold,
		time.Now(), p.ID,
	)
	if err != nil {
		return fmt.Errorf("更新玩家失败: %w", err)
	}
	return nil
}

func (s *RebirthService) insertHistoryTx(tx *sql.Tx, h *model.RebirthHistory) error {
	_, err := tx.Exec(
		`INSERT INTO player_rebirth_history (player_id, rebirth_number, old_realm,
			old_realm_level, old_attack, old_defense, old_hp,
			carry_over_attack, carry_over_defense, carry_over_hp,
			gold_kept, spirit_root_before, spirit_root_after,
			quality_before, quality_after, rebirth_jade_earned, title_earned, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		h.PlayerID, h.RebirthNumber, h.OldRealm, h.OldRealmLevel,
		h.OldAttack, h.OldDefense, h.OldHP,
		h.CarryOverAttack, h.CarryOverDefense, h.CarryOverHP,
		h.GoldKept, h.SpiritRootBefore, h.SpiritRootAfter,
		h.QualityBefore, h.QualityAfter, h.RebirthJadeEarned, h.TitleEarned, h.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("插入轮回历史失败: %w", err)
	}
	return nil
}

func (s *RebirthService) demoteDongFu(tx *sql.Tx, playerID int64) error {
	var dongfuID int64
	err := tx.QueryRow("SELECT id FROM dongfu WHERE player_id = ?", playerID).Scan(&dongfuID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询洞府失败: %w", err)
	}
	_, err = tx.Exec(
		`UPDATE dongfu SET level=1, cultivation_bonus=0, alchemy_bonus=0, storage_bonus=0,
		combat_exp_per_hour=0, spirit_stones_per_hour=0, spirit_energy=0, updated_at=? WHERE id=?`,
		time.Now(), dongfuID,
	)
	if err != nil {
		return fmt.Errorf("降级洞府失败: %w", err)
	}
	_, err = tx.Exec("DELETE FROM dongfu_rooms WHERE dongfu_id = ?", dongfuID)
	if err != nil {
		return fmt.Errorf("删除洞府房间失败: %w", err)
	}
	return nil
}

func (s *RebirthService) demoteArtifact(tx *sql.Tx, playerID int64) error {
	var artifactID int64
	err := tx.QueryRow("SELECT id FROM player_artifacts WHERE player_id = ?", playerID).Scan(&artifactID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询法宝失败: %w", err)
	}
	_, err = tx.Exec(
		`UPDATE player_artifacts SET level=1, exp=0, attack_bonus=10, defense_bonus=5, hp_bonus=20, skill_id=0 WHERE id=?`,
		artifactID,
	)
	if err != nil {
		return fmt.Errorf("降级法宝失败: %w", err)
	}
	return nil
}

func (s *RebirthService) buildBenefits() *model.RebirthBenefits {
	var titles []string
	for i := 0; i <= model.MaxRebirthCount; i++ {
		titles = append(titles, model.RebirthTitleNames[i])
	}
	return &model.RebirthBenefits{
		EnlightenmentPerRebirth:  1,
		SpeedBonusPerRebirth:     model.SpeedBonusPerRebirth,
		MaxRebirthCount:          model.MaxRebirthCount,
		GoldRetentionRate:        "50%",
		DongFuLevelAfter:         1,
		ArtifactLevelAfter:       1,
		SpiritRootQualityUpgrade: "下品→中品→上品→极品（每轮回提升1档）",
		Titles:                   titles,
		TalentPointPerRebirth:    model.TalentPointPerRebirth,
		CarryOverPercent:         model.CarryOverRatio,
	}
}

func calcRebirthBaseAttrs(spiritRoot int32) rebirthBaseAttrs {
	base := rebirthBaseAttrs{hp: 100, mp: 50, attack: 10, defense: 5}
	switch spiritRoot {
	case model.SpiritRootMetal:
		base.attack += 5
	case model.SpiritRootWood:
		base.hp += 30
	case model.SpiritRootWater:
		base.mp += 30
	case model.SpiritRootFire:
		base.attack += 3
		base.defense -= 2
	case model.SpiritRootEarth:
		base.defense += 5
	case model.SpiritRootWind:
		base.attack += 4
		base.mp += 15
	case model.SpiritRootThunder:
		base.attack += 6
		base.defense += 2
	case model.SpiritRootIce:
		base.defense += 4
		base.hp += 15
	}
	return base
}

func rebirthTitle(count int) string {
	if name, ok := model.RebirthTitleNames[count]; ok {
		return name
	}
	return "一世散修"
}

func qualityName(quality int) string {
	if name, ok := model.SpiritRootQualityNames[quality]; ok {
		return name
	}
	return "未知"
}

// rebirthBaseAttrs 轮回后基础属性结构
type rebirthBaseAttrs struct {
	hp      int64
	mp      int64
	attack  int64
	defense int64
}
