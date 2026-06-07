// Package service 提供世界服务的业务逻辑
package service

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// DivinationService 天机阁推演业务逻辑
type DivinationService struct {
	mu    sync.RWMutex
	store map[int64]*model.DivinationRecord // playerID -> record
}

// NewDivinationService 创建 DivinationService
func NewDivinationService() *DivinationService {
	return &DivinationService{
		store: make(map[int64]*model.DivinationRecord),
	}
}

// GetLevel 获取玩家天机阁等级
func (s *DivinationService) GetLevel(playerID int64) *model.DivinationLevel {
	s.mu.RLock()
	record, ok := s.store[playerID]
	s.mu.RUnlock()

	if !ok {
		return &model.DivinationLevel{
			Level:  1,
			Exp:    0,
			ExpMax: model.DivineExpPerLevel,
		}
	}

	expMax := record.Level * model.DivineExpPerLevel
	if expMax <= 0 {
		expMax = model.DivineExpPerLevel
	}

	return &model.DivinationLevel{
		Level:     record.Level,
		Exp:       record.Exp,
		ExpMax:    expMax,
		TotalUsed: record.TotalUsed,
	}
}

// Divine 执行推演
// costGold/costJade: 额外消耗的灵石/仙玉（在基础消耗之上），用于提高准确度
func (s *DivinationService) Divine(playerID int64, divType model.DivinationType, extraGold int64, extraJade int64) (*model.DivinationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取或创建记录
	record, ok := s.store[playerID]
	if !ok {
		record = &model.DivinationRecord{
			PlayerID:  playerID,
			Level:     1,
			TotalUsed: 0,
		}
		s.store[playerID] = record
	}

	// 判断是否可免费推演
	isFree := false
	now := time.Now()
	if record.LastFreeAt == nil || !isSameDay(record.LastFreeAt, &now) {
		isFree = true
		record.LastFreeAt = &now
	}

	// 计算消耗
	costGold := int64(model.DivineCostGoldBase) + extraGold
	costJade := int64(model.DivineCostJadeBase) + extraJade

	if !isFree {
		if costGold <= 0 && costJade <= 0 {
			return nil, fmt.Errorf("非免费推演需要消耗灵石或仙玉")
		}
	} else {
		// 免费推演不消耗货币
		costGold = 0
		costJade = 0
	}

	// 计算准确度
	// 基础准确度 = 50% + 天机阁等级 * 3%
	// 额外消耗: 每 1000 灵石 +2%, 每 10 仙玉 +5%, 上限 95%
	level := record.Level
	baseAccuracy := 0.50 + float64(level)*0.03
	goldBonus := float64(extraGold) / 1000 * 0.02
	jadeBonus := float64(extraJade) / 10 * 0.05
	accuracy := math.Min(baseAccuracy+goldBonus+jadeBonus, model.DivineAccuracyMax)

	// 生成推演内容
	content, data := s.generateContent(divType, accuracy)

	result := &model.DivinationResult{
		ID:        fmt.Sprintf("div_%d_%d", playerID, now.UnixNano()),
		PlayerID:  playerID,
		Type:      divType,
		Content:   content,
		Accuracy:  math.Round(accuracy*100) / 100,
		CostGold:  costGold,
		CostJade:  costJade,
		IsFree:    isFree,
		Data:      data,
		CreatedAt: now,
	}

	// 更新记录
	record.TotalUsed++
	record.Exp += model.DivineExpPerUse

	// 检查升级
	expMax := record.Level * model.DivineExpPerLevel
	for record.Exp >= expMax && record.Level < model.DivineMaxLevel {
		record.Exp -= expMax
		record.Level++
		expMax = record.Level * model.DivineExpPerLevel
		log.Printf("[天机阁] 玩家 %d 天机阁升级至 %d", playerID, record.Level)
	}

	// 保留最近10条结果
	if len(record.Results) >= 10 {
		record.Results = record.Results[1:]
	}
	record.Results = append(record.Results, result)

	log.Printf("[天机阁] 玩家 %d 推演 %s 完成, 准确度 %.0f%%, 免费=%v",
		playerID, divType, result.Accuracy*100, isFree)

	return result, nil
}

// GetResult 获取推演结果
func (s *DivinationService) GetResult(playerID int64, resultID string) (*model.DivinationResult, error) {
	s.mu.RLock()
	record, ok := s.store[playerID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("玩家未进行过推演")
	}

	for _, r := range record.Results {
		if r.ID == resultID {
			return r, nil
		}
	}

	return nil, fmt.Errorf("推演结果不存在")
}

// CanFreeDivine 检查玩家今日是否还可免费推演
func (s *DivinationService) CanFreeDivine(playerID int64) bool {
	s.mu.RLock()
	record, ok := s.store[playerID]
	s.mu.RUnlock()

	if !ok {
		return true
	}

	if record.LastFreeAt == nil {
		return true
	}

	return !isSameDay(record.LastFreeAt, getNowPtr())
}

// GetConsumeOptions 获取推演消耗选项（供前端展示）
type ConsumeOption struct {
	ExtraGold int64   `json:"extra_gold"`
	ExtraJade int64   `json:"extra_jade"`
	Accuracy  float64 `json:"accuracy"`
	Label     string  `json:"label"`
}

func (s *DivinationService) GetConsumeOptions(playerID int64, divType model.DivinationType) []ConsumeOption {
	level := 1
	s.mu.RLock()
	if record, ok := s.store[playerID]; ok {
		level = record.Level
	}
	s.mu.RUnlock()

	baseAcc := 0.50 + float64(level)*0.03

	return []ConsumeOption{
		{
			ExtraGold: 0,
			ExtraJade: 0,
			Accuracy:  math.Min(baseAcc, model.DivineAccuracyMax),
			Label:     "基础推演",
		},
		{
			ExtraGold: 2000,
			ExtraJade: 0,
			Accuracy:  math.Min(baseAcc+0.04, model.DivineAccuracyMax),
			Label:     "精妙推演",
		},
		{
			ExtraGold: 5000,
			ExtraJade: 0,
			Accuracy:  math.Min(baseAcc+0.10, model.DivineAccuracyMax),
			Label:     "玄妙推演",
		},
		{
			ExtraGold: 0,
			ExtraJade: 50,
			Accuracy:  math.Min(baseAcc+0.25, model.DivineAccuracyMax),
			Label:     "天机推演",
		},
	}
}

// generateContent 根据类型和准确度生成推演内容
func (s *DivinationService) generateContent(divType model.DivinationType, accuracy float64) (string, interface{}) {
	switch divType {
	case model.DivinationBreakthrough:
		return s.genBreakthrough(accuracy)
	case model.DivinationTreasure:
		return s.genTreasure(accuracy)
	case model.DivinationWeather:
		return s.genWeather(accuracy)
	default:
		return "天机混沌，无法推演。", nil
	}
}

// genBreakthrough 生成突破时机提示
func (s *DivinationService) genBreakthrough(accuracy float64) (string, interface{}) {
	hourRand := rand.Intn(6) + 1 // 1-6小时后
	phaseAcc := accuracy * 100

	templates := []string{
		fmt.Sprintf("观天象，察气运，%.0f%%的把握断定 %.0f 刻钟后乃突破良机，灵气充盈，心魔不侵。", phaseAcc, accuracy*6),
		fmt.Sprintf("北斗移形，紫微耀世，%.0f%%概率可断定 %.0f 时辰后天时地利，突破可增五分胜算。", phaseAcc, accuracy*8),
		fmt.Sprintf("冥冥之中感应天地气机，%.0f%%可信：子夜时分运道最盛，届时突破可事半功倍。", phaseAcc),
	}

	content := templates[rand.Intn(len(templates))]
	data := map[string]interface{}{
		"best_hour":    hourRand,
		"bonus_rate":   0.05,
		"accuracy_pct": math.Round(phaseAcc*100) / 100,
	}
	return content, data
}

// genTreasure 生成寻宝方位
func (s *DivinationService) genTreasure(accuracy float64) (string, interface{}) {
	regions := []string{
		"幽冥谷", "陨星潭", "风雷崖", "碧波洞", "赤焰岭", "玄冰窟",
	}
	items := []struct {
		name   string
		rarity string
	}{
		{"千年灵芝", "稀有"},
		{"玄铁精矿", "精良"},
		{"天蚕丝", "稀有"},
		{"星辰石", "史诗"},
		{"龙鳞果", "传说"},
	}

	region := regions[rand.Intn(len(regions))]
	item := items[rand.Intn(len(items))]

	// 高准确度时给出更精确的位置
	x := rand.Intn(100)
	y := rand.Intn(100)
	var content string
	if accuracy > 0.8 {
		content = fmt.Sprintf("天机感应，%s藏有%s！位置在(%.0f, %.0f)，方圆十里必有所获。", region, item.name, accuracy*100, accuracy*80)
	} else if accuracy > 0.6 {
		content = fmt.Sprintf("推演得知%s可能有%s出世，但天机朦胧，仅知大致方位。", region, item.name)
	} else {
		content = fmt.Sprintf("偶得一丝灵机，似与%s有关，或有宝物，然天机晦涩难明。", region)
	}

	tmap := &model.TreasureMap{
		RegionID: region,
		X:        x,
		Y:        y,
		ItemName: item.name,
		Rarity:   1,
	}
	return content, tmap
}

// genWeather 生成天气/灵气潮汐预告
func (s *DivinationService) genWeather(accuracy float64) (string, interface{}) {
	weathers := []struct {
		weather string
		effect  string
		bonus   float64
	}{
		{"灵气潮汐", "修炼速度+30%，持续1时辰", 1.3},
		{"天降甘霖", "采集获得加倍，持续2时辰", 2.0},
		{"罡风凛冽", "战斗暴击率+15%，持续1时辰", 1.15},
		{"紫气东来", "炼丹炼器成功率+20%，持续1时辰", 1.2},
		{"流星陨落", "秘境开启概率大增", 0},
		{"雾霭沉沉", "探索遇到奇遇概率+25%", 1.25},
	}

	w := weathers[rand.Intn(len(weathers))]
	periodDesc := "今日"

	switch rand.Intn(3) {
	case 0:
		periodDesc = "今日午后"
	case 1:
		periodDesc = "今夜子时"
	case 2:
		periodDesc = "明日清晨"
	}

	content := fmt.Sprintf("推演天机：%s将出现「%s」，%s。预测可靠度%.0f%%。",
		periodDesc, w.weather, w.effect, accuracy*100)

	forecast := &model.WeatherForecast{
		Period:        periodDesc,
		WeatherType:   w.weather,
		SpiritDensity: w.bonus,
		Effect:        w.effect,
	}
	return content, forecast
}

// isSameDay 判断两个时间是否同一天
func isSameDay(a, b *time.Time) bool {
	if a == nil || b == nil {
		return false
	}
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func getNowPtr() *time.Time {
	now := time.Now()
	return &now
}
