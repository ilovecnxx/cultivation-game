package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CultivationHandler 修炼相关 HTTP 处理器
type CultivationHandler struct {
	playerService *service.PlayerService
	log           *zap.Logger
}

// NewCultivationHandler 创建 CultivationHandler
func NewCultivationHandler(playerService *service.PlayerService, log *zap.Logger) *CultivationHandler {
	return &CultivationHandler{playerService: playerService, log: log}
}

// CultivateTick 修炼 tick（前端每 N 秒调用一次）
// POST /api/v1/player/:id/cultivate/tick
func (h *CultivationHandler) CultivateTick(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	jwtPlayerID, exists := c.Get("player_id")
	if !exists || jwtPlayerID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作此角色"})
		return
	}

	var req struct {
		Seconds int   `json:"seconds" binding:"required,min=1,max=28800"`
		HP      int64 `json:"hp"`
		MP      int64 `json:"mp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}

	rate := model.CalcCultivationRate(player.Realm, player.RealmStage, player.RootQuality)
	gain := int64(rate * float64(req.Seconds))

	player, added, err := h.playerService.AddSpiritPower(c.Request.Context(), playerID, gain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "修炼失败"})
		return
	}

	// 打坐恢复：HP 每10秒回1%，MP 按 mp_regen 速率（×100，如500=5%/秒）
	hpHeal := int64(float64(player.MaxHP) * 0.001 * float64(req.Seconds))
	mpHeal := int64(float64(player.MaxMP) * float64(player.MPRegen) / 10000.0 * float64(req.Seconds))
	needsHeal := false
	if hpHeal > 0 && player.HP < player.MaxHP {
		player.HP += hpHeal
		if player.HP > player.MaxHP {
			player.HP = player.MaxHP
		}
		needsHeal = true
	}
	if mpHeal > 0 && player.MP < player.MaxMP {
		player.MP += mpHeal
		if player.MP > player.MaxMP {
			player.MP = player.MaxMP
		}
		needsHeal = true
	}
	if needsHeal {
		_ = h.playerService.UpdatePlayer(c.Request.Context(), player)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"spirit_power": player.SpiritPower,
			"max_spirit":   player.MaxSpirit,
			"gained":       added,
			"rate":         rate,
			"is_full":      player.SpiritPower >= player.MaxSpirit,
			"hp":           player.HP,
			"mp":           player.MP,
		},
	})
}

// Breakthrough 手动突破
// POST /api/v1/player/:id/breakthrough
func (h *CultivationHandler) Breakthrough(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	jwtPlayerID, exists := c.Get("player_id")
	if !exists || jwtPlayerID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作此角色"})
		return
	}

	success, cost, newRealm, realmName, err := h.playerService.Breakthrough(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("突破失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"success":    success,
			"cost":       cost,
			"new_realm":  newRealm,
			"realm_name": realmName,
		},
	})
}
