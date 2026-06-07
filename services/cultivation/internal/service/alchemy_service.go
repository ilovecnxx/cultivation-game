// Package service 炼丹系统核心业务逻辑
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cultivation-game/services/cultivation/internal/model"
)

// AlchemyService 炼丹服务
type AlchemyService struct {
	mu                sync.RWMutex
	recipes           []model.Recipe
	ingredients       []model.Ingredient
	rng               *rand.Rand
	rngMu             sync.Mutex
	playerServiceAddr string // Player 服务 HTTP 地址
	// 炼丹等级升级所需经验（索引=等级，值=升级所需累计经验）
	levelExpRequirements []int64
}

// NewAlchemyService 创建炼丹服务
// dataDir 为配置目录，包含 alchemy.json
func NewAlchemyService(dataDir string) *AlchemyService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	s := &AlchemyService{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		// 炼丹等级经验表：1级->2级需100，2->3需300，依此类推
		levelExpRequirements: []int64{0, 100, 300, 600, 1000, 1500, 2100, 2800, 3600, 4500},
		playerServiceAddr:    playerAddr,
	}
	if err := s.LoadAlchemyConfig(dataDir); err != nil {
		// 启动时不阻塞，使用空配置
		_ = err
	}
	return s
}

// LoadAlchemyConfig 从JSON文件加载炼丹配置
func (s *AlchemyService) LoadAlchemyConfig(dataDir string) error {
	filePath := filepath.Join(dataDir, "alchemy.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取炼丹配置失败: %w", err)
	}

	var config model.AlchemyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析炼丹配置失败: %w", err)
	}

	s.mu.Lock()
	s.recipes = config.Recipes
	s.ingredients = config.Ingredients
	s.mu.Unlock()

	return nil
}

// GetRecipes 获取玩家可炼制的丹方列表（按境界和炼丹等级筛选）
func (s *AlchemyService) GetRecipes(player *model.Player) []model.Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var available []model.Recipe
	for _, r := range s.recipes {
		// 境界要求：玩家当前境界ID >= 丹方需求境界
		if player.RealmID >= r.RealmRequired &&
			// 炼丹等级要求
			player.AlchemyLevel >= r.LevelRequired {
			available = append(available, r)
		}
	}
	return available
}

// GetRecipeByID 按ID获取丹方
func (s *AlchemyService) GetRecipeByID(id int) (*model.Recipe, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.recipes {
		if s.recipes[i].ID == id {
			return &s.recipes[i], true
		}
	}
	return nil, false
}

// GetIngredientByID 按ID获取材料信息
func (s *AlchemyService) GetIngredientByID(id int) (*model.Ingredient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.ingredients {
		if s.ingredients[i].ID == id {
			return &s.ingredients[i], true
		}
	}
	return nil, false
}

// GetAllIngredients 获取所有灵药材料定义
func (s *AlchemyService) GetAllIngredients() []model.Ingredient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.Ingredient, len(s.ingredients))
	copy(result, s.ingredients)
	return result
}

// Craft 炼制丹药
// 返回炼制结果，若成功则丹药加入玩家背包并更新炼丹经验
func (s *AlchemyService) Craft(player *model.Player, recipeID int) *model.CraftResult {
	// 1. 获取丹方
	recipe, ok := s.GetRecipeByID(recipeID)
	if !ok {
		return &model.CraftResult{
			Quality:     0,
			QualityName: "",
			ExpGained:  0,
			AlchemyExp: 0,
		}
	}

	// 2. 验证玩家境界和炼丹等级要求
	if player.RealmID < recipe.RealmRequired {
		return &model.CraftResult{
			Quality:     0,
			QualityName: "",
			ExpGained:  0,
			AlchemyExp: 0,
		}
	}
	if player.AlchemyLevel < recipe.LevelRequired {
		return &model.CraftResult{
			Quality:     0,
			QualityName: "",
			ExpGained:  0,
			AlchemyExp: 0,
		}
	}

	// 3. 检查材料是否足够
	if player.Ingredients == nil {
		player.Ingredients = make(map[int]int)
	}
	for _, ing := range recipe.Ingredients {
		if player.Ingredients[ing.ItemID] < ing.Count {
			return &model.CraftResult{
				Quality:     0,
				QualityName: "",
				ExpGained:  0,
				AlchemyExp: 0,
			}
		}
	}

	// 4. 扣除材料
	for _, ing := range recipe.Ingredients {
		player.Ingredients[ing.ItemID] -= ing.Count
		if player.Ingredients[ing.ItemID] <= 0 {
			delete(player.Ingredients, ing.ItemID)
		}
	}

	// 5. 判定炼制成功率
	// 基础成功率70% + 炼丹等级*2% + 配方难度修正
	successRate := 0.70 + float64(player.AlchemyLevel)*0.02
	if successRate > 0.95 {
		successRate = 0.95
	}
	// 高等级丹方降低基础成功率
	successRate -= float64(recipe.LevelRequired) * 0.01
	if successRate < 0.30 {
		successRate = 0.30
	}

	roll := s.safeRandFloat()
	success := roll < successRate

	if !success {
		// 失败：获得少量炼丹经验
		alchemyExp := int64(recipe.LevelRequired * 5)
		s.addAlchemyExp(player, alchemyExp)
		return &model.CraftResult{
			Success:    false,
			Quality:    0,
			QualityName: "",
			ExpGained:  0,
			AlchemyExp: alchemyExp,
		}
	}

	// 6. 品质随机
	quality := s.rollQuality(player.AlchemyLevel)

	// 7. 品质对效果的倍率
	qualityMultiplier := s.qualityMultiplier(quality)

	// 8. 创建丹药
	createdAt := time.Now().Unix()
	pill := &model.Pill{
		ID:          model.NewPillID(recipe.ID, quality, createdAt),
		RecipeID:    recipe.ID,
		Name:        recipe.Name,
		Quality:     quality,
		QualityName: quality.String(),
		Effects:     s.scaleEffects(recipe.Effects, qualityMultiplier),
		Count:       1,
		CreatedAt:   createdAt,
		Expiry:      s.calcExpiry(recipe.Effects.Duration),
	}

	// 9. 加入背包
	player.Pills = append(player.Pills, *pill)

	// 通知 Player 服务将丹药加入玩家背包（异步，不阻塞）
	go s.notifyCraftSuccess(player.ID, recipe.ID, recipe.Name, qualityMultiplier)

	// 10. 即时效果：修为增加
	expGained := int64(0)
	if recipe.Effects.ExpBonus > 0 {
		expGained = int64(float64(recipe.Effects.ExpBonus) * qualityMultiplier)
		player.Experience += expGained
	}

	// 11. 炼丹经验 = 配方需求等级 * 10 * 品质倍率
	alchemyExp := int64(float64(recipe.LevelRequired*10) * qualityMultiplier)
	s.addAlchemyExp(player, alchemyExp)

	return &model.CraftResult{
		Success:     true,
		Quality:     quality,
		QualityName: quality.String(),
		Pill:        pill,
		ExpGained:   expGained,
		AlchemyExp:  alchemyExp,
	}
}

// Collect 采集灵药
// 尝试采集指定材料，消耗行动力（简化：每次必得，数量1-3随机）
func (s *AlchemyService) Collect(player *model.Player, ingredientID int) *model.CollectResult {
	// 验证材料是否存在
	ingredient, ok := s.GetIngredientByID(ingredientID)
	if !ok {
		return &model.CollectResult{
			Success: false,
			Message: "未知的灵药材料",
		}
	}

	// 随机数量 1-3，根据稀有度调整
	baseCount := 1
	if ingredient.Rarity <= 2 {
		// 低稀有度材料可得2-3个
		baseCount = 2 + s.safeRandIntn(2)
	} else if ingredient.Rarity <= 4 {
		// 中等稀有度可得1-2个
		baseCount = 1 + s.safeRandIntn(2)
	} else {
		// 高稀有度可得1个
		baseCount = 1
	}

	// 炼丹等级加成：每5级额外+1
	if player.AlchemyLevel >= 5 {
		baseCount += 1
	}
	if player.AlchemyLevel >= 10 {
		baseCount += 1
	}

	// 初始化材料背包
	if player.Ingredients == nil {
		player.Ingredients = make(map[int]int)
	}
	player.Ingredients[ingredientID] += baseCount

	msg := fmt.Sprintf("采集到 %s x%d", ingredient.Name, baseCount)
	return &model.CollectResult{
		Success:      true,
		IngredientID: ingredientID,
		Name:         ingredient.Name,
		Count:        baseCount,
		Message:      msg,
	}
}

// GetPlayerIngredients 获取玩家已采集材料列表（含材料定义信息）
func (s *AlchemyService) GetPlayerIngredients(player *model.Player) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []map[string]interface{}
	for id, count := range player.Ingredients {
		info := map[string]interface{}{
			"item_id": id,
			"count":   count,
		}
		// 补全材料信息
		for _, ing := range s.ingredients {
			if ing.ID == id {
				info["name"] = ing.Name
				info["rarity"] = ing.Rarity
				info["description"] = ing.Description
				break
			}
		}
		result = append(result, info)
	}
	return result
}

// addAlchemyExp 增加炼丹经验并检测升级
func (s *AlchemyService) addAlchemyExp(player *model.Player, exp int64) {
	player.AlchemyExp += exp
	// 检查是否升级
	maxLevel := len(s.levelExpRequirements)
	for player.AlchemyLevel < maxLevel &&
		player.AlchemyExp >= s.levelExpRequirements[player.AlchemyLevel] {
		player.AlchemyLevel++
	}
}

// rollQuality 根据炼丹等级进行品质随机
func (s *AlchemyService) rollQuality(alchemyLevel int) model.Quality {
	roll := s.safeRandFloat()

	// 炼丹等级调整：每级增加0.5%高品质概率，从凡品中扣除
	levelBonus := float64(alchemyLevel) * 0.005

	// 仙品 2% + 等级加成
	if roll < 0.02+levelBonus*0.1 {
		return model.QualityImmortal
	}
	// 绝品 8% + 等级加成
	if roll < 0.10+levelBonus*0.3 {
		return model.QualitySupreme
	}
	// 极品 20% + 等级加成
	if roll < 0.30+levelBonus*0.5 {
		return model.QualityExcellent
	}
	// 良品 30%
	if roll < 0.60+levelBonus {
		return model.QualityGood
	}
	return model.QualityMortal
}

// qualityMultiplier 品质倍率（用于效果缩放和炼丹经验）
func (s *AlchemyService) qualityMultiplier(q model.Quality) float64 {
	switch q {
	case model.QualityMortal:
		return 1.0
	case model.QualityGood:
		return 1.5
	case model.QualityExcellent:
		return 2.5
	case model.QualitySupreme:
		return 4.0
	case model.QualityImmortal:
		return 8.0
	default:
		return 1.0
	}
}

// scaleEffects 根据品质倍率缩放丹药效果
// 突破类加成（时限/判定范围）和修炼速度类加成不缩放，仅数值类缩放
func (s *AlchemyService) scaleEffects(effects model.PillEffects, multiplier float64) model.PillEffects {
	return model.PillEffects{
		ExpBonus:             int64(float64(effects.ExpBonus) * multiplier),
		HpBonus:              int64(float64(effects.HpBonus) * multiplier),
		MpBonus:              int64(float64(effects.MpBonus) * multiplier),
		HealHp:               int64(float64(effects.HealHp) * multiplier),
		DefenseBonus:         int64(float64(effects.DefenseBonus) * multiplier),
		AttackBonus:          int64(float64(effects.AttackBonus) * multiplier),
		CultivationSpeed:     effects.CultivationSpeed,     // 倍率不缩放
		MeditationEfficiency: effects.MeditationEfficiency, // 倍率不缩放
		BreakthroughTimeBonus:  effects.BreakthroughTimeBonus,  // 时限不加成（品质不影响固定秒数）
		BreakthroughRangeBonus: effects.BreakthroughRangeBonus, // 判定范围不加成
		BreakthroughBonus:    effects.BreakthroughBonus,    // 旧版保留，不缩放
		Duration:             effects.Duration,
	}
}

// calcExpiry 计算过期时间（有duration则从当前时间计算，否则为0永不过期）
func (s *AlchemyService) calcExpiry(durationSec int64) int64 {
	if durationSec <= 0 {
		return 0
	}
	return time.Now().Unix() + durationSec
}

// notifyCraftSuccess 炼丹成功后通知 Player 服务将丹药加入背包
func (s *AlchemyService) notifyCraftSuccess(playerID uint64, recipeID int, pillName string, qualityMultiplier float64) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"item_id":  int64(1000 + recipeID), // 用 recipeID 映射为物品 ID
		"quantity": 1,
	})

	url := fmt.Sprintf("%s/api/v1/player/%d/inventory/add", s.playerServiceAddr, playerID)
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
}

// UsePill 使用丹药，返回突破效果（时限加成和判定范围加成）
//
// 新效果替代旧版概率加成：
//   - 凝神丹: 突破时限+30秒 (BreakthroughTimeBonus)
//   - 聚灵丹: 节点判定范围+20% (BreakthroughRangeBonus)
//   - 筑基丹: 解锁大境界突破 (标记位, 由前端逻辑处理)
//   - 护脉丹: 失败时修为只扣15% (标记位, 由前端逻辑处理)
//
// 返回:
//   - timeBonus: 时限加成（秒）
//   - rangeBonus: 判定范围加成（百分比, 如0.2表示+20%）
//   - err: 错误信息
func (s *AlchemyService) UsePill(player *model.Player, pillID string) (timeBonus int64, rangeBonus float64, err error) {
	// 查找玩家背包中的丹药
	var found *model.Pill
	for i := range player.Pills {
		if player.Pills[i].ID == pillID && player.Pills[i].Count > 0 {
			found = &player.Pills[i]
			break
		}
	}
	if found == nil {
		return 0, 0, fmt.Errorf("丹药 %s 不存在或数量不足", pillID)
	}

	// 消耗一个
	found.Count--
	if found.Count <= 0 {
		// 从背包移除
		newPills := make([]model.Pill, 0, len(player.Pills)-1)
		for _, p := range player.Pills {
			if p.ID != pillID {
				newPills = append(newPills, p)
			}
		}
		player.Pills = newPills
	}

	// 从丹药效果中提取时限和判定范围加成
	timeBonus = found.Effects.BreakthroughTimeBonus
	rangeBonus = found.Effects.BreakthroughRangeBonus

	return timeBonus, rangeBonus, nil
}

// GetLevelExpRequirements 获取炼丹等级经验表
func (s *AlchemyService) GetLevelExpRequirements() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]int64, len(s.levelExpRequirements))
	copy(result, s.levelExpRequirements)
	return result
}

// safeRandFloat 线程安全获取[0,1)随机浮点数
func (s *AlchemyService) safeRandFloat() float64 {
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.Float64()
}

// safeRandIntn 线程安全获取[0,n)随机整数
func (s *AlchemyService) safeRandIntn(n int) int {
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.Intn(n)
}
