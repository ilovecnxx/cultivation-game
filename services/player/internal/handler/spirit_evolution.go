// Package handler 灵根进化系统 HTTP 处理器
//
// 路由前缀: /api/v1/player/spirit-evolution
//   GET    /              - 获取当前灵根状态
//   POST   /evolve        - 尝试进化
//   GET    /stones        - 查看进化石需求
//   GET    /history       - 进化历史
package handler

import (
	"math"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SpiritEvolutionHandler 灵根进化 HTTP 处理器
type SpiritEvolutionHandler struct {
	evoService *service.SpiritEvolutionService
	log        *zap.Logger
}

// NewSpiritEvolutionHandler 创建 SpiritEvolutionHandler
func NewSpiritEvolutionHandler(evoService *service.SpiritEvolutionService, log *zap.Logger) *SpiritEvolutionHandler {
	return &SpiritEvolutionHandler{
		evoService: evoService,
		log:        log,
	}
}

// GetCurrentSpiritInfo 获取当前灵根状态
// GET /api/v1/player/spirit-evolution?player_id=xxx
func (h *SpiritEvolutionHandler) GetCurrentSpiritInfo(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Query("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	spirit, err := h.evoService.GetSpiritStatus(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询灵根状态失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	// 下一品质信息
	nextQuality := int(spirit.Quality)
	canEvolve := false
	if spirit.Quality < model.SpiritQualityHongMeng {
		nextQuality = int(spirit.Quality) + 1
		canEvolve = true
	}

	// 境界要求
	reqRealm := int32(0)
	reqRealmName := ""
	if canEvolve {
		if rr, ok := model.EvolutionRealmRequirement[spirit.Quality]; ok {
			reqRealm = rr
			reqRealmName = model.RealmNames[rr]
		}
	}

	// 成功率
	successRate := model.EvolutionBaseSuccessRate + float64(spirit.Reincarnations)*model.ReincarnationSuccessBonus
	successRate = math.Min(successRate, 1.0)

	// 元素觉醒信息
	elementAwakened := false
	elementName := ""
	elementDamageBonus := 0.0
	awakening, err := h.evoService.GetElementAwakening(c.Request.Context(), playerID)
	if err == nil && awakening != nil {
		elementAwakened = true
		elementName = model.SpiritRootNames[awakening.Element]
		elementDamageBonus = awakening.DamageBonus
	}

	resp := model.SpiritInfoResponse{
		PlayerID:           spirit.PlayerID,
		Quality:            int(spirit.Quality),
		QualityName:        model.SpiritQualityNames[spirit.Quality],
		QualityColor:       model.SpiritQualityColors[spirit.Quality],
		NextQuality:        nextQuality,
		NextQualityName:    model.SpiritQualityNames[model.SpiritQuality(nextQuality)],
		NextQualityColor:   model.SpiritQualityColors[model.SpiritQuality(nextQuality)],
		CanEvolve:          canEvolve,
		EvolutionStones:    model.SpiritEvolutionCost[spirit.Quality],
		CultBonus:          int(spirit.Quality) * model.SpiritSpeedBonusPerLevel,
		BreakBonus:         int(spirit.Quality) * model.SpiritBreakthroughBonusPerLevel,
		Reincarnations:     spirit.Reincarnations,
		ReincarnationChance: spirit.ReincarnationUpgradeChance(),
		RealmRequirement:   reqRealm,
		RealmName:          reqRealmName,
		SuccessRate:        successRate,
		ElementAwakened:    elementAwakened,
		ElementName:        elementName,
		ElementDamageBonus: elementDamageBonus,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// EvolveSpiritRoot 尝试灵根进化
// POST /api/v1/player/spirit-evolution/evolve
func (h *SpiritEvolutionHandler) EvolveSpiritRoot(c *gin.Context) {
	var req model.EvolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":   400,
			"msg":    "参数错误",
			"detail": err.Error(),
		})
		return
	}

	result, err := h.evoService.EvolveByStone(c.Request.Context(), req.PlayerID)
	if err != nil {
		h.log.Warn("灵根进化失败", zap.Error(err))
		// 业务错误（境界不足、石头不够等）返回 400
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 构造响应
	responseData := gin.H{
		"success":      result.Success,
		"degraded":     result.Degraded,
		"success_rate": result.SuccessRate,
		"stones_used":  result.StonesUsed,
		"from_quality": int(result.FromQuality),
		"from_name":    model.SpiritQualityNames[result.FromQuality],
		"to_quality":   int(result.ToQuality),
		"to_name":      model.SpiritQualityNames[result.ToQuality],
	}

	if result.ElementAwakening != nil {
		responseData["element_awakening"] = gin.H{
			"element":      result.ElementAwakening.Element,
			"element_name": model.SpiritRootNames[result.ElementAwakening.Element],
			"damage_bonus": result.ElementAwakening.DamageBonus,
		}
	}

	msg := "灵根进化失败"
	if result.Degraded {
		msg = "灵根进化失败，品质降级！"
	} else if result.Success {
		msg = "灵根进化成功！"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  msg,
		"data": responseData,
	})
}

// GetStoneInfo 获取各品质所需进化石信息
// GET /api/v1/player/spirit-evolution/stones
func (h *SpiritEvolutionHandler) GetStoneInfo(c *gin.Context) {
	info := h.evoService.GetStoneInfo()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": info,
	})
}

// GetHistory 获取进化历史（分页）
// GET /api/v1/player/spirit-evolution/history?player_id=xxx&page=1&page_size=20
func (h *SpiritEvolutionHandler) GetHistory(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Query("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, total, err := h.evoService.GetEvolutionHistory(c.Request.Context(), playerID, page, pageSize)
	if err != nil {
		h.log.Warn("查询进化历史失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	// 给历史记录补充名称字段
	type historyItem struct {
		ID          int64   `json:"id"`
		PlayerID    int64   `json:"player_id"`
		FromQuality int     `json:"from_quality"`
		FromName    string  `json:"from_name"`
		ToQuality   int     `json:"to_quality"`
		ToName      string  `json:"to_name"`
		Success     bool    `json:"success"`
		Degraded    bool    `json:"degraded"`
		StonesUsed  int     `json:"stones_used"`
		RealmAtTime int32   `json:"realm_at_time"`
		RealmName   string  `json:"realm_name"`
		SuccessRate float64 `json:"success_rate"`
		CreatedAt   int64   `json:"created_at"`
	}

	items := make([]historyItem, 0, len(records))
	for _, r := range records {
		realmName := model.RealmNames[r.RealmAtTime]
		items = append(items, historyItem{
			ID:          r.ID,
			PlayerID:    r.PlayerID,
			FromQuality: int(r.FromQuality),
			FromName:    model.SpiritQualityNames[r.FromQuality],
			ToQuality:   int(r.ToQuality),
			ToName:      model.SpiritQualityNames[r.ToQuality],
			Success:     r.Success,
			Degraded:    r.Degraded,
			StonesUsed:  r.StonesUsed,
			RealmAtTime: r.RealmAtTime,
			RealmName:   realmName,
			SuccessRate: r.SuccessRate,
			CreatedAt:   r.CreatedAt,
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"items":      items,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_pages": totalPages,
		},
	})
}
