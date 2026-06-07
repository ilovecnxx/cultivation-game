package handler

import (
	"database/sql"
	"math/rand"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PillHandler struct {
	db            *sql.DB
	playerService *service.PlayerService
	log           *zap.Logger
}

func NewPillHandler(db *sql.DB, playerService *service.PlayerService, log *zap.Logger) *PillHandler {
	return &PillHandler{db: db, playerService: playerService, log: log}
}

// PillRecipe 丹方
type PillRecipe struct {
	PillKey      string `json:"pill_key"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Tier         int    `json:"tier"`
	Category     string `json:"category"`
	EffectType   string `json:"effect_type"`
	EffectValue  int    `json:"effect_value"`
	Duration     int    `json:"duration"`
	ReqRealm     int    `json:"req_realm"`
	BaseSuccess  int    `json:"base_success"`
	SSCost       int    `json:"ss_cost"`
	Description  string `json:"description"`
}

var qualityNames = []string{"劣质", "普通", "优良", "精良", "完美"}
var qualityMults = []float64{0.6, 1.0, 1.4, 1.8, 2.5}
var qualityColors = []string{"#888", "#aaa", "#6bcb77", "#4d96ff", "#ff6b9e"}

// ListRecipes 获取所有丹方
func (h *PillHandler) ListRecipes(c *gin.Context) {
	rows, err := h.db.Query("SELECT pill_key,name,icon,tier,category,effect_type,effect_value,duration,req_realm,base_success,ss_cost,description FROM pill_recipes ORDER BY tier")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}
	defer rows.Close()
	var recipes []PillRecipe
	for rows.Next() {
		var r PillRecipe
		rows.Scan(&r.PillKey, &r.Name, &r.Icon, &r.Tier, &r.Category, &r.EffectType, &r.EffectValue, &r.Duration, &r.ReqRealm, &r.BaseSuccess, &r.SSCost, &r.Description)
		recipes = append(recipes, r)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": recipes})
}

// CraftPill 炼制丹药
func (h *PillHandler) CraftPill(c *gin.Context) {
	idStr := c.Param("id")
	playerID, _ := strconv.ParseInt(idStr, 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct {
		PillKey string `json:"pill_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 查丹方
	var recipe PillRecipe
	err := h.db.QueryRow("SELECT pill_key,name,icon,tier,category,effect_type,effect_value,duration,req_realm,base_success,ss_cost,description FROM pill_recipes WHERE pill_key=?", req.PillKey).
		Scan(&recipe.PillKey, &recipe.Name, &recipe.Icon, &recipe.Tier, &recipe.Category, &recipe.EffectType, &recipe.EffectValue, &recipe.Duration, &recipe.ReqRealm, &recipe.BaseSuccess, &recipe.SSCost, &recipe.Description)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "丹方不存在"})
		return
	}
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil || player.Realm < int32(recipe.ReqRealm) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "境界不足"})
		return
	}
	if player.SpiritSense < int64(recipe.SSCost) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "神识不足"})
		return
	}
	player.SpiritSense -= int64(recipe.SSCost)
	// 成功率 = 基础 + 神识/50%
	successRate := recipe.BaseSuccess + int(player.SpiritSense/50)
	if successRate > 95 { successRate = 95 }
	success := rand.Intn(100) < successRate
	if !success {
		_ = h.playerService.UpdatePlayer(c.Request.Context(), player)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": false, "reason": "炼制失败，材料损耗"}})
		return
	}
	// 品质判定（炼丹师等级加成）
	alchemyLv := int(player.ProfessionLevel)
	if player.Profession != "dan" { alchemyLv = 0 }
	quality := rollQuality(player.SpiritSense, player.Luck, alchemyLv)
	// 存背包 + 职业经验
	h.db.Exec("INSERT INTO player_pills (player_id,pill_key,tier,quality,quantity) VALUES (?,?,?,?,1) ON DUPLICATE KEY UPDATE quantity=quantity+1", playerID, req.PillKey, recipe.Tier, quality)
	player.ProfessionExp += int64(recipe.Tier * 10)
	// 升级判定：每100经验升1级，最高10级
	if player.ProfessionExp >= int64((player.ProfessionLevel+1)*100) && player.ProfessionLevel < 10 {
		player.ProfessionLevel++
	}
	_ = h.playerService.UpdatePlayer(c.Request.Context(), player)
	effect := int(float64(recipe.EffectValue) * qualityMults[quality])
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"success": true, "quality": quality, "quality_name": qualityNames[quality],
		"name": recipe.Name, "effect": effect, "effect_type": recipe.EffectType, "duration": recipe.Duration,
		"level": player.ProfessionLevel, "exp": player.ProfessionExp,
	}})
}

// UsePill 使用丹药
func (h *PillHandler) UsePill(c *gin.Context) {
	idStr := c.Param("id")
	playerID, _ := strconv.ParseInt(idStr, 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct {
		PillID int64 `json:"pill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 查丹药
	var pillID, tier, quality, quantity int64
	var pillKey string
	err := h.db.QueryRow("SELECT id,pill_key,tier,quality,quantity FROM player_pills WHERE id=? AND player_id=?", req.PillID, playerID).Scan(&pillID, &pillKey, &tier, &quality, &quantity)
	if err != nil || quantity <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "丹药不存在"})
		return
	}
	// 查丹方效果
	var effectType string
	var effectValue, duration int
	err = h.db.QueryRow("SELECT effect_type,effect_value,duration FROM pill_recipes WHERE pill_key=?", pillKey).Scan(&effectType, &effectValue, &duration)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "丹方不存在"})
		return
	}
	actualEffect := int(float64(effectValue) * qualityMults[quality])
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}
	// 应用效果
	applyPillEffect(player, effectType, actualEffect, duration)
	_ = h.playerService.UpdatePlayer(c.Request.Context(), player)
	// 减少数量
	if quantity <= 1 {
		h.db.Exec("DELETE FROM player_pills WHERE id=?", pillID)
	} else {
		h.db.Exec("UPDATE player_pills SET quantity=quantity-1 WHERE id=?", pillID)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": true, "effect": actualEffect, "effect_type": effectType}})
}

// ListPlayerPills 列出玩家丹药
func (h *PillHandler) ListPlayerPills(c *gin.Context) {
	idStr := c.Param("id")
	playerID, _ := strconv.ParseInt(idStr, 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	rows, err := h.db.Query("SELECT pp.id,pp.pill_key,pr.name,pr.icon,pp.tier,pp.quality,pp.quantity,pr.effect_type,pr.effect_value,pr.duration FROM player_pills pp JOIN pill_recipes pr ON pp.pill_key=pr.pill_key WHERE pp.player_id=? AND pp.quantity>0", playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}
	defer rows.Close()
	type PlayerPill struct {
		ID          int64  `json:"id"`
		PillKey     string `json:"pill_key"`
		Name        string `json:"name"`
		Icon        string `json:"icon"`
		Tier        int    `json:"tier"`
		Quality     int    `json:"quality"`
		QualityName string `json:"quality_name"`
		Quantity    int    `json:"quantity"`
		EffectType  string `json:"effect_type"`
		EffectValue int    `json:"effect_value"`
		Duration    int    `json:"duration"`
	}
	var pills []PlayerPill
	for rows.Next() {
		var p PlayerPill
		rows.Scan(&p.ID, &p.PillKey, &p.Name, &p.Icon, &p.Tier, &p.Quality, &p.Quantity, &p.EffectType, &p.EffectValue, &p.Duration)
		p.QualityName = qualityNames[p.Quality]
		pills = append(pills, p)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pills})
}

func rollQuality(ss int64, luck int64, alchemyLevel int) int {
	r := rand.Intn(100)
	bonus := int(ss/100) + alchemyLevel*2 + int(luck/50)
	if r < 10+bonus { return 4 } // 完美
	if r < 25+bonus { return 3 }  // 精良
	if r < 50+bonus { return 2 }  // 优良
	if r < 85 { return 1 }        // 普通
	return 0                       // 劣质
}

func applyPillEffect(player *model.Player, effectType string, effectValue int, duration int) {
	switch effectType {
	case "heal_hp":
		player.HP += int64(player.MaxHP) * int64(effectValue) / 100
		if player.HP > player.MaxHP { player.HP = player.MaxHP }
	case "heal_mp":
		player.MP += int64(player.MaxMP) * int64(effectValue) / 100
		if player.MP > player.MaxMP { player.MP = player.MaxMP }
	case "heal_ss":
		player.SpiritSense += int64(effectValue)
	case "atk_bonus":
		player.Attack = player.Attack * int64(100+effectValue) / 100
	case "def_bonus":
		player.Defense = player.Defense * int64(100+effectValue) / 100
	case "spd_bonus":
		player.Speed = player.Speed * int64(100+effectValue) / 100
	case "cult_bonus":
		player.CultBonus += int64(effectValue)
	case "break_bonus":
		player.BreakBonus += int64(effectValue)
	case "luck_bonus":
		player.Luck += int64(effectValue)
		if player.Luck > 100 { player.Luck = 100 }
	case "ss_bonus":
		player.SpiritSense += int64(effectValue)
	case "lifespan":
		player.Lifespan += int64(effectValue)
	case "root_reroll":
		if rand.Intn(100) < effectValue {
			roots := []int32{model.SpiritRootMetal, model.SpiritRootWood, model.SpiritRootWater, model.SpiritRootFire, model.SpiritRootEarth, model.SpiritRootDi, model.SpiritRootTian}
			player.SpiritRoot = roots[rand.Intn(len(roots))]
		}
	}
}
