package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// PlayerService 玩家业务逻辑
type PlayerService struct {
	playerRepo PlayerRepository
	cache      Cache
	log        *zap.Logger
}

// NewPlayerService 创建 PlayerService
func NewPlayerService(playerRepo PlayerRepository, cache Cache, log *zap.Logger) *PlayerService {
	return &PlayerService{
		playerRepo: playerRepo,
		cache:      cache,
		log:        log,
	}
}

// CreatePlayer 创建角色
func (s *PlayerService) CreatePlayer(ctx context.Context, req *model.CreatePlayerRequest) (*model.Player, error) {
	existing, err := s.playerRepo.GetByName(req.Name)
	if err != nil {
		return nil, fmt.Errorf("检查重名失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("角色名 %s 已被使用", req.Name)
	}

	maxSpirit := model.CalcMaxSpirit(model.RealmForge, 1)
	atk, def, hp, maxMp, spd, critRate, critDmg, dodge, hit, cultBonus, breakBonus, mpRegen, lifespan := model.CalcRealmAttributes(model.RealmForge, 1)

	player := &model.Player{
		UserID:      req.UserID,
		Name:        req.Name,
		Gender:      req.Gender,
		Level:       1,
		Realm:       model.RealmForge,
		RealmStage:  1,
		SpiritRoot:  0,
		RootQuality: 0,
		HP:          hp,
		MaxHP:       hp,
		MP:          maxMp,
		MaxMP:       maxMp,
		Attack:      atk,
		Defense:     def,
		Speed:       spd,
		CritRate:    critRate,
		CritDmg:     critDmg,
		Dodge:       dodge,
		Hit:         hit,
		CultBonus:   cultBonus,
		BreakBonus:  breakBonus,
		MPRegen:     mpRegen,
		Lifespan:       lifespan,
		Comprehension:  model.RandomComprehension(0),
		Luck:           model.RollDailyLuck(0, 0),
		SpiritSense:    model.CalcSpiritSense(model.RealmForge, 0),
		LastLuckDate:   "",
		SpiritPower:    0,
		MaxSpirit:   maxSpirit,
		Gold:        100,
		Jade:        0,
	}

	if err := s.playerRepo.Create(player); err != nil {
		return nil, err
	}

	if err := s.cache.SetPlayer(ctx, player.ToCache()); err != nil {
		s.log.Warn("创建角色时写入缓存失败", zap.Error(err))
	}

	return player, nil
}

// GetPlayer 获取玩家信息（优先走缓存）
func (s *PlayerService) GetPlayer(ctx context.Context, playerID int64) (*model.Player, error) {
	cached, err := s.cache.GetPlayer(ctx, playerID)
	if err != nil {
		s.log.Warn("读取玩家缓存失败，回查 MySQL", zap.Error(err))
	}

	if cached != nil {
		_ = s.cache.RefreshTTL(ctx, playerID)
		player := &model.Player{}
		player.FromCache(cached)
		return player, nil
	}

	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	if err := s.cache.SetPlayer(ctx, player.ToCache()); err != nil {
		s.log.Warn("回写玩家缓存失败", zap.Error(err))
	}

	return player, nil
}

// GetPlayerByUserID 根据用户ID获取玩家
func (s *PlayerService) GetPlayerByUserID(ctx context.Context, userID string) (*model.Player, error) {
	player, err := s.playerRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, fmt.Errorf("玩家不存在")
	}

	if err := s.cache.SetPlayer(ctx, player.ToCache()); err != nil {
		s.log.Warn("回写玩家缓存失败", zap.Error(err))
	}

	return player, nil
}

// UpdatePlayer 更新玩家属性
func (s *PlayerService) UpdatePlayer(ctx context.Context, player *model.Player) error {
	player.UpdatedAt = time.Now()
	if err := s.playerRepo.Update(player); err != nil {
		return err
	}
	if err := s.cache.SetPlayer(ctx, player.ToCache()); err != nil {
		s.log.Warn("更新玩家缓存失败", zap.Error(err))
	}
	return nil
}

// UpdateCurrency 更新货币
func (s *PlayerService) UpdateCurrency(ctx context.Context, playerID int64, req *model.CurrencyChangeRequest) (*model.Player, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	if req.Gold < 0 && player.Gold+req.Gold < 0 {
		return nil, fmt.Errorf("灵石不足")
	}
	if req.Jade < 0 && player.Jade+req.Jade < 0 {
		return nil, fmt.Errorf("仙玉不足")
	}

	player.Gold += req.Gold
	player.BoundGold += req.BoundGold
	player.Jade += req.Jade

	if err := s.playerRepo.UpdateCurrency(playerID, player.Gold, player.BoundGold, player.Jade); err != nil {
		return nil, err
	}

	if err := s.cache.SetPlayer(ctx, player.ToCache()); err != nil {
		s.log.Warn("更新货币缓存失败", zap.Error(err))
	}

	return player, nil
}

// UpdateRealm 突破成功后更新境界
func (s *PlayerService) UpdateRealm(ctx context.Context, playerID int64, realmID, stage int32, attack, defense, maxHP int64) error {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return err
	}

	player.Realm = realmID
	player.RealmStage = stage
	player.Attack = attack
	player.Defense = defense
	player.MaxHP = maxHP
	player.MaxMP = maxHP / 2
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}

	return s.UpdatePlayer(ctx, player)
}

// AddSpiritPower 增加修为（到上限停止）
func (s *PlayerService) AddSpiritPower(ctx context.Context, playerID int64, amount int64) (*model.Player, int64, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, 0, err
	}

	if player.SpiritPower >= player.MaxSpirit {
		return player, 0, nil // 已达上限
	}

	before := player.SpiritPower
	player.SpiritPower += amount
	if player.SpiritPower > player.MaxSpirit {
		player.SpiritPower = player.MaxSpirit
	}
	added := player.SpiritPower - before

	if err := s.UpdatePlayer(ctx, player); err != nil {
		return nil, 0, err
	}
	return player, added, nil
}

// Breakthrough 手动突破
// 返回: 是否成功, 扣除的修为, 新境界名, error
func (s *PlayerService) Breakthrough(ctx context.Context, playerID int64) (bool, int64, int32, string, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return false, 0, 0, "", err
	}

	// 必须满修为才能突破
	if player.SpiritPower < player.MaxSpirit {
		return false, 0, 0, "", fmt.Errorf("修为不足，无法突破")
	}

	// 小境界突破成功率：根据灵根品质 50%-99%
	smallStageRates := map[int32]int32{
		model.RootQualityNone: 50, model.RootQualityLow: 65, model.RootQualityMedium: 80, model.RootQualityHigh: 90, model.RootQualityPerfect: 99,
	}
	// 满10期才能突破大境界
	if player.RealmStage < 10 {
		if player.SpiritPower >= player.MaxSpirit {
			// 升小期 — 随机判定（失败扣20%修为）
			smallRate := smallStageRates[player.RootQuality]
			if smallRate < 1 { smallRate = 50 }
			if rand.Intn(100) >= int(smallRate) {
				cost := player.MaxSpirit * 20 / 100
				player.SpiritPower -= cost
				if player.SpiritPower < 0 { player.SpiritPower = 0 }
				_ = s.UpdatePlayer(ctx, player)
				return false, cost, player.Realm, model.RealmNames[player.Realm], nil
			}
			player.RealmStage++
			oldReq := player.MaxSpirit
			player.SpiritPower = 0
			player.MaxSpirit = model.CalcMaxSpirit(player.Realm, player.RealmStage)
			// 更新属性
			player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.Speed, player.CritRate, player.CritDmg, player.Dodge, player.Hit, player.CultBonus, player.BreakBonus, player.MPRegen, player.Lifespan = model.CalcRealmAttributes(player.Realm, player.RealmStage)
			player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.CritRate, player.CritDmg, player.Dodge, player.MPRegen = model.ApplySpiritRootBonuses(player.SpiritRoot, player.RootQuality, player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.CritRate, player.CritDmg, player.Dodge, player.MPRegen)
			player.Comprehension = model.CalcComprehension(player.RootQuality, player.Realm, 0)
			player.HP = player.MaxHP // 突破回满血
			if err := s.UpdatePlayer(ctx, player); err != nil {
				return false, 0, 0, "", err
			}
			return true, oldReq, player.Realm, model.RealmNames[player.Realm], nil
		}
		return false, 0, 0, "", fmt.Errorf("修为不足")
	}

	// 大境界突破（10期→下一境1期）
	// 练气期触发灵根分配

	rate := model.CalcBreakthroughRate(player.Realm, player.RootQuality, int64(time.Since(player.CreatedAt).Hours()/24), player.Lifespan, player.Luck, player.Comprehension)
	r := rand.Intn(100)
	success := r < int(rate)

	cost := player.MaxSpirit * 20 / 100 // 失败扣20%
	if success {
		cost = player.MaxSpirit // 成功消耗全部修为
		nextRealm := player.Realm + 1
		if nextRealm > model.RealmTrib {
			// 渡劫→飞升
			return true, cost, player.Realm, "飞升成功！", nil
		}
		player.Realm = nextRealm
		player.RealmStage = 1
		player.SpiritPower = 0
		player.MaxSpirit = model.CalcMaxSpirit(player.Realm, 1)
		// 更新属性到新境界
		player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.Speed, player.CritRate, player.CritDmg, player.Dodge, player.Hit, player.CultBonus, player.BreakBonus, player.MPRegen, player.Lifespan = model.CalcRealmAttributes(player.Realm, 1)
		player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.CritRate, player.CritDmg, player.Dodge, player.MPRegen = model.ApplySpiritRootBonuses(player.SpiritRoot, player.RootQuality, player.Attack, player.Defense, player.MaxHP, player.MaxMP, player.CritRate, player.CritDmg, player.Dodge, player.MPRegen)
		player.Comprehension = model.CalcComprehension(player.RootQuality, player.Realm, 0)
		player.SpiritSense = model.CalcSpiritSense(player.Realm, player.RootQuality)
		player.HP = player.MaxHP // 突破回满血

		// 第一次到练气期，随机分配灵根
		if player.Realm == model.RealmQiRef && player.SpiritRoot == 0 {
			s.assignRandomSpiritRoot(player)
			player.Comprehension = model.RandomComprehension(player.RootQuality)
			player.SpiritSense = model.CalcSpiritSense(player.Realm, player.RootQuality)
		}


		if err := s.UpdatePlayer(ctx, player); err != nil {
			return false, 0, 0, "", err
		}
		return true, cost, player.Realm, model.RealmNames[player.Realm], nil
	}
		// 失败
	player.SpiritPower -= cost
	if player.SpiritPower < 0 {
		player.SpiritPower = 0
	}
	if err := s.UpdatePlayer(ctx, player); err != nil {
		return false, 0, 0, "", err
	}
	return false, cost, player.Realm, model.RealmNames[player.Realm], nil
}

// AssignSpiritRootOnReachQiRef 当玩家到达练气期时分配灵根
func (s *PlayerService) assignRandomSpiritRoot(player *model.Player) {
	if player.SpiritRoot != 0 {
		return // 已有灵根
	}

	// 7种灵根等概率
	roots := []int32{
		model.SpiritRootMetal,
		model.SpiritRootWood,
		model.SpiritRootWater,
		model.SpiritRootFire,
		model.SpiritRootEarth,
		model.SpiritRootDi,
		model.SpiritRootTian,
	}
	player.SpiritRoot = roots[rand.Intn(len(roots))]

	// 品质概率：极品8% 上品22% 中品40% 下品20% 无品10%
	r := rand.Intn(100)
	switch {
	case r < 8:
		player.RootQuality = model.RootQualityPerfect
	case r < 30:
		player.RootQuality = model.RootQualityHigh
	case r < 70:
		player.RootQuality = model.RootQualityMedium
	case r < 90:
		player.RootQuality = model.RootQualityLow
	default:
		player.RootQuality = model.RootQualityNone
	}
}

// AddExp 增加经验（战斗/世界等服务调用）
func (s *PlayerService) AddExp(ctx context.Context, playerID int64, exp int64) (*model.Player, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	player.Experience += exp
	if player.Experience < 0 {
		player.Experience = 0
	}
	if err := s.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}
	return player, nil
}

// GetPlayerWithDetails 获取玩家完整信息（含每日气运判定）
func (s *PlayerService) GetPlayerWithDetails(ctx context.Context, playerID int64) (*model.PlayerResponse, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// 每日首次加载时重新随机气运
	today := time.Now().Format("2006-01-02")
	if player.LastLuckDate != today {
		player.Luck = model.RollDailyLuck(player.RootQuality, player.SpiritRoot)
		player.LastLuckDate = today
		_ = s.UpdatePlayer(ctx, player) // 非关键路径，忽略保存失败
	}

	resp := &model.PlayerResponse{
		Player:      player,
		RealmName:   realmName(player.Realm),
		RealmStage:  player.RealmStage,
		SpiritName:  spiritRootName(player.SpiritRoot),
		QualityName: rootQualityName(player.RootQuality),
		CultRate:    model.CalcCultivationRate(player.Realm, player.RealmStage, player.RootQuality),
		BreakRate:   model.CalcBreakthroughRate(player.Realm, player.RootQuality, int64(time.Since(player.CreatedAt).Hours()/24), player.Lifespan, player.Luck, player.Comprehension),
	}

	return resp, nil
}

// GetBreakthroughInfo 获取突破信息（供前端显示）
// RollDailyLuckIfNewDay 每日首次登录时重新随机气运
func (s *PlayerService) RollDailyLuckIfNewDay(ctx context.Context, playerID int64) (int64, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return 0, err
	}
	today := time.Now().Format("2006-01-02")
	if player.LastLuckDate == today {
		return player.Luck, nil // 今天已经随过了
	}
	player.Luck = model.RollDailyLuck(player.RootQuality, player.SpiritRoot)
	player.LastLuckDate = today
	if err := s.UpdatePlayer(ctx, player); err != nil {
		return 0, err
	}
	return player.Luck, nil
}

func (s *PlayerService) GetBreakthroughInfo(ctx context.Context, playerID int64) (int32, int32, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return 0, 0, err
	}
	rate := model.CalcBreakthroughRate(player.Realm, player.RootQuality, int64(time.Since(player.CreatedAt).Hours()/24), player.Lifespan, player.Luck, player.Comprehension)
	return rate, player.RealmStage, nil
}

// ============================================================
// 辅助函数
// ============================================================

func realmName(realm int32) string {
	if name, ok := model.RealmNames[realm]; ok {
		return name
	}
	return "未知"
}

func spiritRootName(spiritRoot int32) string {
	if name, ok := model.SpiritRootNames[spiritRoot]; ok {
		return name
	}
	return "无灵根"
}

func rootQualityName(quality int32) string {
	if name, ok := model.RootQualityNames[quality]; ok {
		return name
	}
	return "未知"
}

// ============================================================
// 道侣属性加成
// ============================================================

// PartnerStatBonus 道侣属性加成信息
type PartnerStatBonus struct {
	BonusAttack  int64 `json:"bonus_attack"`
	BonusDefense int64 `json:"bonus_defense"`
	BonusMaxHP   int64 `json:"bonus_max_hp"`
}

// GetDaolvPartnerStats 获取道侣的原始属性(供社交服务调用)
func (s *PlayerService) GetDaolvPartnerStats(ctx context.Context, partnerID int64) (*PartnerStatBonus, error) {
	partner, err := s.GetPlayer(ctx, partnerID)
	if err != nil {
		return nil, err
	}

	return &PartnerStatBonus{
		BonusAttack:  int64(float64(partner.Attack) * 0.05),
		BonusDefense: int64(float64(partner.Defense) * 0.05),
		BonusMaxHP:   int64(float64(partner.MaxHP) * 0.05),
	}, nil
}

// ApplyDaolvStatsBonus 应用道侣属性加成
func (s *PlayerService) ApplyDaolvStatsBonus(ctx context.Context, playerID int64, bonus *PartnerStatBonus) (*model.Player, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	player.Attack += bonus.BonusAttack
	player.Defense += bonus.BonusDefense
	player.MaxHP += bonus.BonusMaxHP
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}

	if err := s.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}
	return player, nil
}

// RemoveDaolvStatsBonus 移除道侣属性加成
func (s *PlayerService) RemoveDaolvStatsBonus(ctx context.Context, playerID int64, bonus *PartnerStatBonus) (*model.Player, error) {
	player, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	player.Attack -= bonus.BonusAttack
	player.Defense -= bonus.BonusDefense
	player.MaxHP -= bonus.BonusMaxHP
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}
	if player.HP < 1 {
		player.HP = 1
	}

	if err := s.UpdatePlayer(ctx, player); err != nil {
		return nil, err
	}
	return player, nil
}
