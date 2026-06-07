// Package service 世界Boss系统
//
// 包含42个Boss配置(14个区域 x 3个难度等级)
// 每日20:00在随机区域刷新Boss, 持续2小时, 所有玩家共享Boss血量
// Boss阶段: 75%/50%/25%血量时进入狂暴(受到伤害增加)
// 击杀奖励: 最后一击奖励 + 全服Buff持续1小时
package service

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// ============================================================
// 常量
// ============================================================

const (
	// BossSpawnHour Boss每日刷新小时
	BossSpawnHour = 20
	// BossSpawnMinute Boss刷新分钟
	BossSpawnMinute = 0
	// BossDurationMinutes Boss存在时间(分钟)
	BossDurationMinutes = 120
	// MaxAttacksPerPlayer 每玩家每Boss每日最大攻击次数
	MaxAttacksPerPlayer = 3
	// CritChance 暴击概率
	CritChance = 0.10
	// CritMultiplier 暴击倍率
	CritMultiplier = 2.0
	// DamageVariance 伤害浮动范围(±%)
	DamageVariance = 0.20
	// GlobalBuffDurationMinutes 全服Buff持续时间(分钟)
	GlobalBuffDurationMinutes = 60

	// Phase thresholds (HP百分比)
	phaseThreshold1 = 0.75
	phaseThreshold2 = 0.50
	phaseThreshold3 = 0.25
)

// phaseDamageBonus phase->伤害加成
var phaseDamageBonuses = []float64{1.0, 1.10, 1.20, 1.35}

// Boss phase labels
var phaseLabels = []string{
	"normal",   // phase 0: 100%-75%
	"enraged",  // phase 1: 75%-50%
	"angry",    // phase 2: 50%-25%
	"furious",  // phase 3: 25%-0%
}

// ============================================================
// Boss模板 14个区域 x 3个难度等级 = 42个配置
// ============================================================

// bossTemplate Boss静态配置
type bossTemplate struct {
	ID          string // 唯一ID, 如 "wb_01_1" (区域01 难度1)
	RegionID    string // 区域ID
	RegionName  string // 区域名
	Name        string // Boss名称
	Tier        int    // 难度等级 1/2/3
	Level       int    // Boss等级
	MaxHP       int64  // 最大血量
	Attack      float64
	Defense     float64
	GoldReward  int64  // 基础参与灵石奖励
	ExpReward   int64  // 基础参与修为奖励
	Description string // Boss描述
}

// bossTemplates 所有42个Boss配置
var bossTemplates = func() []*bossTemplate {
	type regionDef struct {
		id   string
		name string
	}
	regions := []regionDef{
		{"region_01", "青竹村"},
		{"region_02", "青云山脉"},
		{"region_03", "星辉城"},
		{"region_04", "秘境森林"},
		{"region_05", "落霞谷"},
		{"region_06", "玄冰洞"},
		{"region_07", "天机阁"},
		{"region_08", "蛮荒之地"},
		{"region_09", "万妖窟"},
		{"region_10", "无尽深渊"},
		{"region_11", "新月湖"},
		{"region_12", "飞升台"},
		{"region_13", "上古遗迹"},
		{"region_14", "隐藏灵脉"},
	}

	// 各区域Boss名字(14个)
	names := []string{
		"青竹妖王", "石窟巨兽", "星辰古魔", "密林妖皇",
		"落霞火凤", "玄冰龙", "天机傀儡", "蛮荒巨兽",
		"万妖之王", "深渊之主", "新月狼神", "飞升守护者",
		"上古凶兽", "灵脉巨龙",
	}

	// 各区域基础血量
	baseHPs := []int64{
		50000, 150000, 300000, 500000,
		800000, 1200000, 1800000, 2500000,
		4000000, 6000000, 9000000, 15000000,
		25000000, 40000000,
	}

	// 各区域基础等级
	baseLevels := []int{1, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8}

	// 各区域奖励
	baseGoldRewards := []int64{200, 250, 300, 350, 400, 450, 500, 550, 600, 650, 700, 750, 800, 850}
	baseExpRewards := []int64{5000, 6000, 7000, 8000, 9000, 10000, 12000, 14000, 16000, 18000, 20000, 22000, 25000, 30000}

	// 描述
	descriptions := []string{
		"青竹村深处的千年竹妖，吸收天地灵气化为妖王。",
		"盘踞在青云山脉深处的巨石怪物，拥有无与伦比的防御力。",
		"来自星辰之外的远古恶魔，拥有操控星辰之力的能力。",
		"秘境森林的统治者，掌握着自然之力。",
		"落霞谷中诞生的火凤凰，周身燃烧着不灭的火焰。",
		"玄冰洞中的远古冰龙，寒气足以冻结一切。",
		"天机阁最强大的机关傀儡，蕴含着上古机关术的精髓。",
		"蛮荒之地的霸主，拥有毁灭一切的原始力量。",
		"万妖窟的统治者，统领着千万妖兽。",
		"无尽深渊中最恐怖的存在，吞噬一切光明。",
		"新月湖的守护神狼，在月光下拥有无穷力量。",
		"飞升台的守护者，考验着每一位修行者的实力。",
		"上古遗迹中沉睡的凶兽，拥有毁天灭地的力量。",
		"隐藏灵脉中沉睡的巨龙，掌控着天地灵脉。",
	}

	// Tier系数: HP倍数, 等级加成, 奖励倍数
	type tierCoeff struct {
		hpMul      float64
		lvAdd      int
		rewardMul  float64
	}
	tiers := []tierCoeff{
		{0.6, 0, 0.8},   // Tier1 简单
		{1.0, 2, 1.0},   // Tier2 普通
		{1.8, 5, 1.5},   // Tier3 困难
	}

	var templates []*bossTemplate
	for ri, reg := range regions {
		for ti, tc := range tiers {
			id := fmt.Sprintf("wb_%02d_%d", ri+1, ti+1)
			hp := int64(float64(baseHPs[ri]) * tc.hpMul)
			lv := baseLevels[ri] + tc.lvAdd
			gold := int64(float64(baseGoldRewards[ri]) * tc.rewardMul)
			exp := int64(float64(baseExpRewards[ri]) * tc.rewardMul)

			templates = append(templates, &bossTemplate{
				ID:          id,
				RegionID:    reg.id,
				RegionName:  reg.name,
				Name:        names[ri],
				Tier:        ti + 1,
				Level:       lv,
				MaxHP:       hp,
				Attack:      float64(lv * 15),
				Defense:     float64(lv * 8),
				GoldReward:  gold,
				ExpReward:   exp,
				Description: descriptions[ri] + fmt.Sprintf(" [难度%d]", ti+1),
			})
		}
	}
	return templates
}()

// ============================================================
// 活动Boss会话
// ============================================================

// activeSession 当前活动Boss会话
type activeSession struct {
	template      *bossTemplate
	currentHP     int64
	status        string   // "alive", "defeated"
	spawnedAt     time.Time
	killedAt      time.Time
	phase         int      // 0=normal, 1=enraged, 2=angry, 3=furious
	phaseNotified [3]bool  // 是否已通知各阶段切换
	lastHitPlayerID   string
	lastHitPlayerName string
	participantCount  int
}

// playerRecord 玩家伤害记录
type playerRecord struct {
	playerID    string
	playerName  string
	totalDamage int64
	attackCount int
	lastAttack  time.Time
}

// globalBuff 全服Buff
type globalBuff struct {
	active    bool
	buffType  string // "boss_kill"
	buffName  string // Buff名称
	effect    string // Buff效果描述
	multiplier float64
	expiresAt time.Time
}

// dailyAttackRecord 每日攻击次数记录
type dailyAttackRecord struct {
	date  string // "2006-01-02"
	playerID string
	bossSessionID string // 当天刷新的Boss的ID
	count int
}

// ============================================================
// WorldBossService
// ============================================================

// WorldBossService 世界Boss业务逻辑
type WorldBossService struct {
	mu              sync.RWMutex
	templates       []*bossTemplate                 // 所有42个Boss模板
	session         *activeSession                  // 当前活动Boss
	playerRecords   map[string][]*playerRecord      // bossSessionID -> []playerRecord
	killRecords     map[string]*model.BossKillRecord // bossID -> 最近一次击杀记录
	buff            *globalBuff                     // 当前全服Buff
	dailyAttacks    []dailyAttackRecord             // 每日攻击次数
	lastSpawnDate   string                          // 上次刷新的日期 "2006-01-02"
	spawnTimes      []struct{ hour, minute int }
	stopCh          chan struct{}
	running         bool
}

// NewWorldBossService 创建WorldBossService
func NewWorldBossService() *WorldBossService {
	s := &WorldBossService{
		templates:     bossTemplates,
		session:       nil,
		playerRecords: make(map[string][]*playerRecord),
		killRecords:   make(map[string]*model.BossKillRecord),
		buff:          &globalBuff{},
		dailyAttacks:  make([]dailyAttackRecord, 0),
		spawnTimes: []struct{ hour, minute int }{
			{BossSpawnHour, BossSpawnMinute},
		},
		stopCh: make(chan struct{}),
	}
	return s
}

// Start 启动后台goroutine, 每30秒检查一次刷新/过期
func (s *WorldBossService) Start() {
	if s.running {
		return
	}
	s.running = true
	log.Println("[世界Boss] 世界Boss系统启动，检查间隔30秒")

	// 启动时立刻检查一次
	go func() {
		s.tick()
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-s.stopCh:
				log.Println("[世界Boss] 世界Boss系统已停止")
				return
			}
		}
	}()
}

// Stop 停止后台goroutine
func (s *WorldBossService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// tick 定时检查逻辑
func (s *WorldBossService) tick() {
	now := time.Now()
	today := now.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 检查是否需要刷新Boss(到达刷新时间, 当天尚未刷新)
	if s.session == nil || s.session.status == "defeated" {
		if s.lastSpawnDate != today && s.isSpawnTime(now) {
			s.spawnBoss(now)
		}
	}

	// 2. 检查Boss是否过期(超过存在时间)
	if s.session != nil && s.session.status == "alive" {
		elapsed := now.Sub(s.session.spawnedAt)
		if elapsed >= BossDurationMinutes*time.Minute {
			s.session.status = "defeated"
			s.session.killedAt = now
			log.Printf("[世界Boss] Boss '%s' 存在时间结束，已消失", s.session.template.Name)
		}
	}

	// 3. 检查Buff是否过期
	if s.buff.active && now.After(s.buff.expiresAt) {
		s.buff.active = false
		log.Println("[世界Boss] 全服Buff已过期")
	}

	// 4. 清理今日之前的每日攻击记录
	s.cleanDailyAttacks(today)
}

// isSpawnTime 检查当前时间是否到达刷新时间
func (s *WorldBossService) isSpawnTime(now time.Time) bool {
	for _, st := range s.spawnTimes {
		if now.Hour() == st.hour && now.Minute() == st.minute {
			return true
		}
	}
	return false
}

// spawnBoss 刷新Boss (调用方持有锁)
func (s *WorldBossService) spawnBoss(now time.Time) {
	// 随机选择一个Boss模板
	tpl := s.templates[rand.Intn(len(s.templates))]

	s.session = &activeSession{
		template:  tpl,
		currentHP: tpl.MaxHP,
		status:    "alive",
		spawnedAt: now,
		phase:     0,
	}
	s.lastSpawnDate = now.Format("2006-01-02")
	s.playerRecords[tpl.ID] = make([]*playerRecord, 0)

	log.Printf("[世界Boss] === Boss '%s' (Lv.%d) 在 '%s' 刷新! HP=%d ===",
		tpl.Name, tpl.Level, tpl.RegionName, tpl.MaxHP)
}

// ============================================================
// 公开API
// ============================================================

// ListBosses 获取所有Boss的简要状态(供前端展示)
func (s *WorldBossService) ListBosses() []*model.BossStatusBrief {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bosses := make([]*model.BossStatusBrief, 0, len(s.templates))

	for _, tpl := range s.templates {
		status, hp, hpPct, spawnedAt := s.getBossStatus(tpl.ID)

		spawnLeft := ""
		if status == "dormant" {
			// 计算下一次刷新倒计时
			now := time.Now()
			nextSpawn := time.Date(now.Year(), now.Month(), now.Day(), BossSpawnHour, BossSpawnMinute, 0, 0, now.Location())
			if now.After(nextSpawn) {
				nextSpawn = nextSpawn.Add(24 * time.Hour)
			}
			rem := nextSpawn.Sub(now)
			if rem > 0 {
				hours := int(rem.Hours())
				mins := int(rem.Minutes()) % 60
				if hours > 0 {
					spawnLeft = fmt.Sprintf("%d小时%d分后", hours, mins)
				} else {
					spawnLeft = fmt.Sprintf("%d分后", mins)
				}
			}
		}

		bosses = append(bosses, &model.BossStatusBrief{
			BossConfig: model.BossConfig{
				BossID:      tpl.ID,
				RegionID:    tpl.RegionID,
				RegionName:  tpl.RegionName,
				Name:        tpl.Name,
				Level:       tpl.Level,
				MaxHP:       float64(tpl.MaxHP),
				Attack:      tpl.Attack,
				Defense:     tpl.Defense,
				GoldReward:  tpl.GoldReward,
				ExpReward:   tpl.ExpReward,
				Description: tpl.Description,
			},
			Status:    status,
			HP:        float64(hp),
			HPPct:     hpPct,
			SpawnedAt: spawnedAt,
			SpawnLeft: spawnLeft,
		})
	}

	return bosses
}

// getBossStatus 获取指定Boss模板的当前状态 (调用方持有读锁)
func (s *WorldBossService) getBossStatus(tplID string) (status string, hp int64, hpPct float64, spawnedAt time.Time) {
	if s.session != nil && s.session.template.ID == tplID {
		if s.session.status == "alive" {
			hp = s.session.currentHP
			hpPct = float64(hp) / float64(s.session.template.MaxHP) * 100
			return "alive", hp, math.Round(hpPct*10) / 10, s.session.spawnedAt
		}
		// 已击杀
		return "defeated", 0, 0, s.session.spawnedAt
	}
	// 检查是否有击杀记录
	if _, ok := s.killRecords[tplID]; ok {
		return "defeated", 0, 0, s.killRecords[tplID].KilledAt
	}
	return "dormant", 0, 0, time.Time{}
}

// GetBossDetail 获取Boss详情(含排行榜)
func (s *WorldBossService) GetBossDetail(bossID string) (*model.BossStatusDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 查找模板
	var tpl *bossTemplate
	for _, t := range s.templates {
		if t.ID == bossID {
			tpl = t
			break
		}
	}
	if tpl == nil {
		return nil, fmt.Errorf("Boss不存在: %s", bossID)
	}

	status, hp, hpPct, spawnedAt := s.getBossStatus(bossID)

	var damageRank []*model.WorldBossDamage
	var lastKill *model.BossKillRecord

	if records, ok := s.playerRecords[bossID]; ok && len(records) > 0 {
		// 聚合同一玩家的伤害
		agg := make(map[string]*model.WorldBossDamage)
		for _, r := range records {
			if entry, exists := agg[r.playerID]; exists {
				entry.Damage += r.totalDamage
			} else {
				agg[r.playerID] = &model.WorldBossDamage{
					PlayerID:   parseUint64(r.playerID),
					PlayerName: r.playerName,
					Damage:     r.totalDamage,
					Time:       r.lastAttack,
				}
			}
		}
		for _, entry := range agg {
			damageRank = append(damageRank, entry)
		}
		// 按伤害降序排列
		sort.Slice(damageRank, func(i, j int) bool {
			return damageRank[i].Damage > damageRank[j].Damage
		})
		// 只取前50
		if len(damageRank) > 50 {
			damageRank = damageRank[:50]
		}
	}

	// 获取击杀记录
	if kr, ok := s.killRecords[bossID]; ok {
		lastKill = kr
	}

	spawnLeft := ""
	if status == "dormant" {
		now := time.Now()
		nextSpawn := time.Date(now.Year(), now.Month(), now.Day(), BossSpawnHour, BossSpawnMinute, 0, 0, now.Location())
		if now.After(nextSpawn) {
			nextSpawn = nextSpawn.Add(24 * time.Hour)
		}
		rem := nextSpawn.Sub(now)
		if rem > 0 {
			hours := int(rem.Hours())
			mins := int(rem.Minutes()) % 60
			if hours > 0 {
				spawnLeft = fmt.Sprintf("%d小时%d分后", hours, mins)
			} else {
				spawnLeft = fmt.Sprintf("%d分后", mins)
			}
		}
	}

	detail := &model.BossStatusDetail{
		BossStatusBrief: model.BossStatusBrief{
			BossConfig: model.BossConfig{
				BossID:      tpl.ID,
				RegionID:    tpl.RegionID,
				RegionName:  tpl.RegionName,
				Name:        tpl.Name,
				Level:       tpl.Level,
				MaxHP:       float64(tpl.MaxHP),
				Attack:      tpl.Attack,
				Defense:     tpl.Defense,
				GoldReward:  tpl.GoldReward,
				ExpReward:   tpl.ExpReward,
				Description: tpl.Description,
			},
			Status:    status,
			HP:        float64(hp),
			HPPct:     hpPct,
			SpawnedAt: spawnedAt,
			SpawnLeft: spawnLeft,
		},
		DamageRank: damageRank,
		LastKill:   lastKill,
	}

	return detail, nil
}

// AttackBoss 玩家攻击Boss
func (s *WorldBossService) AttackBoss(req *model.BossAttackRequest) (*model.BossAttackResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否有活跃Boss
	if s.session == nil {
		return nil, fmt.Errorf("当前没有活跃的Boss")
	}
	if s.session.status != "alive" {
		return nil, fmt.Errorf("Boss已被击败或已消失")
	}
	if s.session.template.ID != req.BossID {
		return nil, fmt.Errorf("该Boss当前并未激活")
	}

	today := time.Now().Format("2006-01-02")

	// 检查每日攻击次数限制
	attackCount := 0
	for _, d := range s.dailyAttacks {
		if d.date == today && d.playerID == req.PlayerID && d.bossSessionID == req.BossID {
			attackCount = d.count
			break
		}
	}
	if attackCount >= MaxAttacksPerPlayer {
		return nil, fmt.Errorf("今日攻击次数已达上限(%d次)", MaxAttacksPerPlayer)
	}

	// 计算实际伤害
	baseDmg := int64(req.AttackVal)
	if baseDmg <= 0 {
		baseDmg = 100 // 默认攻击力
	}

	// 伤害浮动 ±20%
	variance := 1.0 + (rand.Float64()*2-1)*DamageVariance
	rawDmg := int64(float64(baseDmg) * variance)
	if rawDmg < 1 {
		rawDmg = 1
	}

	// 暴击判定
	critical := rand.Float64() < CritChance
	if critical {
		rawDmg = int64(float64(rawDmg) * CritMultiplier)
	}

	// 阶段伤害加成
	phaseBonus := phaseDamageBonuses[s.session.phase]
	finalDmg := int64(float64(rawDmg) * phaseBonus)
	if finalDmg < 1 {
		finalDmg = 1
	}

	// 检查最后一击
	lastHit := false
	if finalDmg >= s.session.currentHP {
		finalDmg = s.session.currentHP
		lastHit = true
	}

	// 扣除Boss血量
	s.session.currentHP -= finalDmg

	// 记录玩家伤害
	playerRec := s.findOrCreatePlayerRecord(req.BossID, req.PlayerID, req.PlayerName)
	playerRec.totalDamage += finalDmg
	playerRec.attackCount++
	playerRec.lastAttack = time.Now()

	// 记录每日攻击
	s.recordDailyAttack(today, req.PlayerID, req.BossID)

	// 检查Boss是否被击杀
	if s.session.currentHP <= 0 {
		s.session.status = "defeated"
		s.session.killedAt = time.Now()
		s.session.lastHitPlayerID = req.PlayerID
		s.session.lastHitPlayerName = req.PlayerName
		lastHit = true

		// 计算参与人数
		s.session.participantCount = len(s.playerRecords[req.BossID])

		// 创建击杀记录
		s.createKillRecord(req.BossID)

		// 激活全服Buff
		s.buff = &globalBuff{
			active:     true,
			buffType:   "boss_kill",
			buffName:   "勇者祝福",
			effect:     "全服修炼效率提升30%，持续1小时",
			multiplier: 1.3,
			expiresAt:  time.Now().Add(GlobalBuffDurationMinutes * time.Minute),
		}

		log.Printf("[世界Boss] Boss '%s' 被玩家 '%s' 击杀! 参与人数: %d",
			s.session.template.Name, req.PlayerName, s.session.participantCount)

	} else {
		// 检查阶段切换
		s.checkPhaseTransition()
	}

	// 计算本次奖励
	goldReward := s.calculateGoldReward(req.PlayerID, req.BossID, finalDmg)
	expReward := s.calculateExpReward(req.PlayerID, req.BossID, finalDmg)

	// 最后一击额外奖励
	if lastHit {
		goldReward += model.LastHitGoldBonus
		expReward += model.LastHitExpBonus
	}

	result := &model.BossAttackResult{
		Damage:       float64(finalDmg),
		BossRemainHP: float64(s.session.currentHP),
		BossMaxHP:    float64(s.session.template.MaxHP),
		BossStatus:   s.session.status,
		Critical:     critical,
		LastHit:      lastHit,
		GoldReward:   goldReward,
		ExpReward:    expReward,
	}

	return result, nil
}

// findOrCreatePlayerRecord 查找或创建玩家伤害记录
func (s *WorldBossService) findOrCreatePlayerRecord(bossID, playerID, playerName string) *playerRecord {
	records, ok := s.playerRecords[bossID]
	if !ok {
		records = make([]*playerRecord, 0)
		s.playerRecords[bossID] = records
	}

	for _, r := range records {
		if r.playerID == playerID {
			return r
		}
	}

	pr := &playerRecord{
		playerID:    playerID,
		playerName:  playerName,
		totalDamage: 0,
		attackCount: 0,
		lastAttack:  time.Time{},
	}
	s.playerRecords[bossID] = append(s.playerRecords[bossID], pr)
	return pr
}

// recordDailyAttack 记录每日攻击次数
func (s *WorldBossService) recordDailyAttack(date, playerID, bossID string) {
	for i, d := range s.dailyAttacks {
		if d.date == date && d.playerID == playerID && d.bossSessionID == bossID {
			s.dailyAttacks[i].count++
			return
		}
	}
	s.dailyAttacks = append(s.dailyAttacks, dailyAttackRecord{
		date:          date,
		playerID:      playerID,
		bossSessionID: bossID,
		count:         1,
	})
}

// checkPhaseTransition 检查Boss阶段切换
func (s *WorldBossService) checkPhaseTransition() {
	hpPct := float64(s.session.currentHP) / float64(s.session.template.MaxHP)
	oldPhase := s.session.phase

	if hpPct <= phaseThreshold3 {
		s.session.phase = 3
	} else if hpPct <= phaseThreshold2 {
		s.session.phase = 2
	} else if hpPct <= phaseThreshold1 {
		s.session.phase = 1
	} else {
		s.session.phase = 0
	}

	if s.session.phase > oldPhase && s.session.phase > 0 {
		idx := s.session.phase - 1
		if !s.session.phaseNotified[idx] {
			s.session.phaseNotified[idx] = true
			log.Printf("[世界Boss] Boss '%s' 进入 %s 阶段! 伤害加成 %.0f%%",
				s.session.template.Name,
				phaseLabels[s.session.phase],
				(phaseDamageBonuses[s.session.phase]-1)*100)
		}
	}
}

// createKillRecord 创建击杀记录
func (s *WorldBossService) createKillRecord(bossID string) {
	tpl := s.session.template

	// 构建Top伤害列表
	var topDamage []model.WorldBossDamage
	if records, ok := s.playerRecords[bossID]; ok {
		// 聚合
		agg := make(map[string]*model.WorldBossDamage)
		for _, r := range records {
			if entry, exists := agg[r.playerID]; exists {
				entry.Damage += r.totalDamage
			} else {
				agg[r.playerID] = &model.WorldBossDamage{
					PlayerID:   parseUint64(r.playerID),
					PlayerName: r.playerName,
					Damage:     r.totalDamage,
					Time:       r.lastAttack,
				}
			}
		}
		for _, entry := range agg {
			topDamage = append(topDamage, *entry)
		}
		sort.Slice(topDamage, func(i, j int) bool {
			return topDamage[i].Damage > topDamage[j].Damage
		})
		if len(topDamage) > 10 {
			topDamage = topDamage[:10]
		}
	}

	// 生成回放日志
	replayLog := []string{
		fmt.Sprintf("远古凶兽「%s」在「%s」区域出现！", tpl.Name, tpl.RegionName),
		fmt.Sprintf("无数修士前往讨伐，战斗持续 %s ...", time.Since(s.session.spawnedAt).Round(time.Second).String()),
		fmt.Sprintf("修士「%s」给予了致命一击！", s.session.lastHitPlayerName),
		fmt.Sprintf("Boss「%s」被成功击杀！共有 %d 名修士参与战斗。", tpl.Name, s.session.participantCount),
	}

	record := &model.BossKillRecord{
		BossID:       tpl.ID,
		BossName:     tpl.Name,
		RegionID:     tpl.RegionID,
		RegionName:   tpl.RegionName,
		KilledAt:     s.session.killedAt,
		KillerID:     s.session.lastHitPlayerID,
		KillerName:   s.session.lastHitPlayerName,
		Participants: s.session.participantCount,
		TopDamage:    topDamage,
		ReplayLog:    replayLog,
	}

	s.killRecords[bossID] = record
}

// calculateGoldReward 计算灵石奖励 (基于伤害占比)
func (s *WorldBossService) calculateGoldReward(playerID, bossID string, damage int64) int64 {
	tpl := s.session.template

	// 基础参与奖励
	baseReward := tpl.GoldReward

	// 伤害占比奖励 (额外奖励 = 基础奖励 * 伤害占比 * 2)
	totalDamage := int64(0)
	if records, ok := s.playerRecords[bossID]; ok {
		for _, r := range records {
			totalDamage += r.totalDamage
		}
	}
	// 如果只有自己攻击过, 用本次伤害
	if totalDamage < damage {
		totalDamage = damage
	}

	if totalDamage <= 0 {
		return baseReward
	}

	dmgPct := float64(damage) / float64(tpl.MaxHP)
	bonusReward := int64(float64(baseReward) * dmgPct * 5.0)

	// 排名奖励
	rankBonus := s.calculateRankBonus(playerID, bossID, "gold")

	return baseReward + bonusReward + rankBonus
}

// calculateExpReward 计算修为奖励
func (s *WorldBossService) calculateExpReward(playerID, bossID string, damage int64) int64 {
	tpl := s.session.template
	baseReward := tpl.ExpReward

	totalDamage := int64(0)
	if records, ok := s.playerRecords[bossID]; ok {
		for _, r := range records {
			totalDamage += r.totalDamage
		}
	}
	if totalDamage < damage {
		totalDamage = damage
	}

	if totalDamage <= 0 {
		return baseReward
	}

	dmgPct := float64(damage) / float64(tpl.MaxHP)
	bonusReward := int64(float64(baseReward) * dmgPct * 5.0)

	rankBonus := s.calculateRankBonus(playerID, bossID, "exp")

	return baseReward + bonusReward + rankBonus
}

// calculateRankBonus 计算排名奖励
func (s *WorldBossService) calculateRankBonus(playerID, bossID, typ string) int64 {
	// 聚合伤害
	agg := make([]struct {
		playerID string
		damage   int64
	}, 0)
	if records, ok := s.playerRecords[bossID]; ok {
		dmgMap := make(map[string]int64)
		for _, r := range records {
			dmgMap[r.playerID] += r.totalDamage
		}
		for pid, dmg := range dmgMap {
			agg = append(agg, struct {
				playerID string
				damage   int64
			}{pid, dmg})
		}
		sort.Slice(agg, func(i, j int) bool {
			return agg[i].damage > agg[j].damage
		})
	}

	if len(agg) == 0 {
		return 0
	}

	// 找到排名
	rank := -1
	for i, a := range agg {
		if a.playerID == playerID {
			rank = i + 1
			break
		}
	}
	if rank <= 0 {
		return 0
	}

	totalPlayers := len(agg)

	// 根据排名区间发放奖励
	for _, rr := range model.BossRewardRanks {
		// 检查排名是否在区间内
		inRank := false
		if rr.RankTo == 0 {
			if rank >= rr.RankFrom {
				inRank = true
			}
		} else if rank >= rr.RankFrom && rank <= rr.RankTo {
			inRank = true
		}

		// 检查百分比区间 (如果RankTo为0且RankFrom>100, 表示百分比)
		if rr.RankFrom > 100 && totalPlayers > 0 {
			pct := float64(rank) / float64(totalPlayers) * 100
			pctFloor := int(pct)
			pctRankFrom := rr.RankFrom - 100 // e.g., 151 -> 51%
			if pctFloor >= pctRankFrom {
				inRank = true
			}
		}

		if inRank {
			if typ == "gold" {
				return rr.GoldBonus
			}
			return rr.ExpBonus
		}
	}

	return 0
}

// GetDamageRanking 获取伤害排行
func (s *WorldBossService) GetDamageRanking(bossID string) ([]model.WorldBossDamage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查Boss是否存在
	found := false
	for _, t := range s.templates {
		if t.ID == bossID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Boss不存在: %s", bossID)
	}

	// 聚合伤害
	agg := make(map[string]*model.WorldBossDamage)
	if records, ok := s.playerRecords[bossID]; ok {
		for _, r := range records {
			if entry, exists := agg[r.playerID]; exists {
				entry.Damage += r.totalDamage
			} else {
				agg[r.playerID] = &model.WorldBossDamage{
					PlayerID:   parseUint64(r.playerID),
					PlayerName: r.playerName,
					Damage:     r.totalDamage,
					Time:       r.lastAttack,
				}
			}
		}
	}

	result := make([]model.WorldBossDamage, 0, len(agg))
	for _, entry := range agg {
		result = append(result, *entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Damage > result[j].Damage
	})

	return result, nil
}

// GetKillRecord 获取Boss击杀记录
func (s *WorldBossService) GetKillRecord(bossID string) *model.BossKillRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if kr, ok := s.killRecords[bossID]; ok {
		return kr
	}
	return nil
}

// ListBossesBrief 获取区域Boss列表(前端专用)
func (s *WorldBossService) ListBossesBrief() *model.BossListResponse {
	bosses := s.ListBosses()
	return &model.BossListResponse{
		Bosses: bosses,
	}
}

// ============================================================
// 新接口
// ============================================================

// GetActiveBoss 获取当前活跃Boss
func (s *WorldBossService) GetActiveBoss() *model.BossStatusDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil
	}

	tpl := s.session.template
	hpPct := float64(0)
	if s.session.currentHP > 0 && tpl.MaxHP > 0 {
		hpPct = float64(s.session.currentHP) / float64(tpl.MaxHP) * 100
	}

	var damageRank []*model.WorldBossDamage
	if records, ok := s.playerRecords[tpl.ID]; ok {
		agg := make(map[string]*model.WorldBossDamage)
		for _, r := range records {
			if entry, exists := agg[r.playerID]; exists {
				entry.Damage += r.totalDamage
			} else {
				agg[r.playerID] = &model.WorldBossDamage{
					PlayerID:   parseUint64(r.playerID),
					PlayerName: r.playerName,
					Damage:     r.totalDamage,
					Time:       r.lastAttack,
				}
			}
		}
		for _, entry := range agg {
			damageRank = append(damageRank, entry)
		}
		sort.Slice(damageRank, func(i, j int) bool {
			return damageRank[i].Damage > damageRank[j].Damage
		})
		if len(damageRank) > 50 {
			damageRank = damageRank[:50]
		}
	}

	return &model.BossStatusDetail{
		BossStatusBrief: model.BossStatusBrief{
			BossConfig: model.BossConfig{
				BossID:      tpl.ID,
				RegionID:    tpl.RegionID,
				RegionName:  tpl.RegionName,
				Name:        tpl.Name,
				Level:       tpl.Level,
				MaxHP:       float64(tpl.MaxHP),
				Attack:      tpl.Attack,
				Defense:     tpl.Defense,
				GoldReward:  tpl.GoldReward,
				ExpReward:   tpl.ExpReward,
				Description: tpl.Description,
			},
			Status:    s.session.status,
			HP:        float64(s.session.currentHP),
			HPPct:     math.Round(hpPct*10) / 10,
			SpawnedAt: s.session.spawnedAt,
		},
		DamageRank: damageRank,
		LastKill:   s.killRecords[tpl.ID],
	}
}

// GetUpcomingBosses 获取即将刷新的Boss列表
func (s *WorldBossService) GetUpcomingBosses() []*model.BossStatusBrief {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	nextSpawn := time.Date(now.Year(), now.Month(), now.Day(), BossSpawnHour, BossSpawnMinute, 0, 0, now.Location())
	if now.After(nextSpawn) {
		nextSpawn = nextSpawn.Add(24 * time.Hour)
	}
	spawnLeft := nextSpawn.Sub(now)

	spawnLeftStr := ""
	if spawnLeft > 0 {
		hours := int(spawnLeft.Hours())
		mins := int(spawnLeft.Minutes()) % 60
		if hours > 0 {
			spawnLeftStr = fmt.Sprintf("%d小时%d分后", hours, mins)
		} else {
			spawnLeftStr = fmt.Sprintf("%d分后", mins)
		}
	}

	// 如果当前有活跃Boss, 不返回即将刷新的
	if s.session != nil && s.session.status == "alive" {
		return nil
	}

	// 随机选择3个即将刷新的Boss
	count := 3
	if count > len(s.templates) {
		count = len(s.templates)
	}
	indices := rand.Perm(len(s.templates))[:count]

	result := make([]*model.BossStatusBrief, 0, count)
	for _, idx := range indices {
		tpl := s.templates[idx]
		result = append(result, &model.BossStatusBrief{
			BossConfig: model.BossConfig{
				BossID:      tpl.ID,
				RegionID:    tpl.RegionID,
				RegionName:  tpl.RegionName,
				Name:        tpl.Name,
				Level:       tpl.Level,
				MaxHP:       float64(tpl.MaxHP),
				Attack:      tpl.Attack,
				Defense:     tpl.Defense,
				GoldReward:  tpl.GoldReward,
				ExpReward:   tpl.ExpReward,
				Description: tpl.Description,
			},
			Status:    "dormant",
			HP:        0,
			HPPct:     0,
			SpawnLeft: spawnLeftStr,
		})
	}
	return result
}

// GetPlayerRewards 获取玩家在指定Boss中的奖励信息
func (s *WorldBossService) GetPlayerRewards(bossID, playerID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查Boss是否存在
	var tpl *bossTemplate
	for _, t := range s.templates {
		if t.ID == bossID {
			tpl = t
			break
		}
	}
	if tpl == nil {
		return nil
	}

	// 查找玩家伤害记录
	var playerTotal int64
	attackCount := 0
	if _, ok := s.playerRecords[bossID]; ok {
		for _, r := range s.playerRecords[bossID] {
			if r.playerID == playerID {
				playerTotal = r.totalDamage
				attackCount = r.attackCount
				break
			}
		}
	}

	// 计算总伤害
	totalDamage := int64(0)
	if records, ok := s.playerRecords[bossID]; ok {
		for _, r := range records {
			totalDamage += r.totalDamage
		}
	}

	// 计算排名
	rank := 0
	if totalDamage > 0 {
		agg := make([]struct {
			pid string
			dmg int64
		}, 0)
		dmgMap := make(map[string]int64)
		for _, r := range s.playerRecords[bossID] {
			dmgMap[r.playerID] += r.totalDamage
		}
		for pid, dmg := range dmgMap {
			agg = append(agg, struct {
				pid string
				dmg int64
			}{pid, dmg})
		}
		sort.Slice(agg, func(i, j int) bool {
			return agg[i].dmg > agg[j].dmg
		})
		for i, a := range agg {
			if a.pid == playerID {
				rank = i + 1
				break
			}
		}
	}

	// 计算应得奖励
	goldReward := model.BaseParticipateGold
	expReward := model.BaseParticipateExp

	if tpl.MaxHP > 0 && playerTotal > 0 {
		dmgPct := float64(playerTotal) / float64(tpl.MaxHP)
		goldReward += int64(float64(tpl.GoldReward) * dmgPct * 5.0)
		expReward += int64(float64(tpl.ExpReward) * dmgPct * 5.0)
	}

	// 排名奖励
	for _, rr := range model.BossRewardRanks {
		if rank >= rr.RankFrom && (rr.RankTo == 0 || rank <= rr.RankTo) {
			goldReward += rr.GoldBonus
			expReward += rr.ExpBonus
			break
		}
	}

	// 检查最后一击
	isLastHit := false
	if kr, ok := s.killRecords[bossID]; ok && kr.KillerID == playerID {
		isLastHit = true
		goldReward += model.LastHitGoldBonus
		expReward += model.LastHitExpBonus
	}

	result := map[string]interface{}{
		"boss_id":      bossID,
		"player_id":    playerID,
		"total_damage": playerTotal,
		"attack_count": attackCount,
		"rank":         rank,
		"total_players": len(s.playerRecords[bossID]),
		"gold_reward":  goldReward,
		"exp_reward":   expReward,
		"is_last_hit":  isLastHit,
	}

	return result
}

// IsGlobalBuffActive 检查全服Buff是否生效
func (s *WorldBossService) IsGlobalBuffActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buff != nil && s.buff.active
}

// GetGlobalBuffInfo 获取全服Buff信息
func (s *WorldBossService) GetGlobalBuffInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.buff == nil || !s.buff.active {
		return map[string]interface{}{
			"active": false,
		}
	}

	remaining := time.Until(s.buff.expiresAt)
	return map[string]interface{}{
		"active":     true,
		"buff_name":  s.buff.buffName,
		"effect":     s.buff.effect,
		"multiplier": s.buff.multiplier,
		"remaining":  remaining.String(),
		"expires_at": s.buff.expiresAt,
	}
}

// cleanDailyAttacks 清理非今日的每日攻击记录
func (s *WorldBossService) cleanDailyAttacks(today string) {
	var clean []dailyAttackRecord
	for _, d := range s.dailyAttacks {
		if d.date == today {
			clean = append(clean, d)
		}
	}
	s.dailyAttacks = clean
}

// parseUint64 将字符串转换为uint64，忽略错误
func parseUint64(s string) uint64 {
	n := uint64(0)
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint64(c-'0')
		}
	}
	return n
}

// GetAllBosses 获取所有Boss(兼容旧接口)
func (s *WorldBossService) GetAllBosses() []*model.WorldBoss {
	briefs := s.ListBosses()
	result := make([]*model.WorldBoss, len(briefs))
	for i, b := range briefs {
		result[i] = &model.WorldBoss{
			ID:     b.BossID,
			Name:   b.Name,
			Region: b.RegionName,
			MaxHP:  int64(b.MaxHP),
			CurrentHP: int64(b.HP),
			Level:  b.Level,
			Status: b.Status,
		}
	}
	return result
}

// GetBossStatus 获取Boss状态(兼容旧接口)
func (s *WorldBossService) GetBossStatus(bid string) *model.WorldBoss {
	detail, err := s.GetBossDetail(bid)
	if err != nil {
		return nil
	}
	return &model.WorldBoss{
		ID:     detail.BossID,
		Name:   detail.Name,
		Region: detail.RegionName,
		MaxHP:  int64(detail.MaxHP),
		CurrentHP: int64(detail.HP),
		Level:  detail.Level,
		Status: detail.Status,
	}
}

// ShouldSpawnNow 检查是否应该刷新Boss(兼容旧接口)
func (s *WorldBossService) ShouldSpawnNow() bool {
	now := time.Now()
	return s.isSpawnTime(now)
}

// RandomBoss 随机选择一个Boss模板(兼容旧接口)
func (s *WorldBossService) RandomBoss() *model.WorldBoss {
	tpl := s.templates[rand.Intn(len(s.templates))]
	return &model.WorldBoss{
		ID:     tpl.ID,
		Name:   tpl.Name,
		Region: tpl.RegionName,
		MaxHP:  tpl.MaxHP,
		Level:  tpl.Level,
	}
}

// SpawnBosses 刷新所有Boss(兼容旧接口)
func (s *WorldBossService) SpawnBosses() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	today := now.Format("2006-01-02")

	if s.lastSpawnDate == today {
		return nil
	}

	s.spawnBoss(now)
	return []string{s.session.template.ID}
}
