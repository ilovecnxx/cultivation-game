package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BodyHandler 炼体系统 HTTP 处理器
type BodyHandler struct {
	bodyService   *service.BodyService
	playerService *service.PlayerService
	log           *zap.Logger
}

// NewBodyHandler 创建 BodyHandler
func NewBodyHandler(bodyService *service.BodyService, playerService *service.PlayerService, log *zap.Logger) *BodyHandler {
	return &BodyHandler{
		bodyService:   bodyService,
		playerService: playerService,
		log:           log,
	}
}

// Train 炼体训练
// POST /api/v1/body/train
func (h *BodyHandler) Train(c *gin.Context) {
	playerID, err := h.getPlayerID(c)
	if err != nil {
		return
	}

	var req model.BodyTrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	// 检查玩家境界是否达到筑基期（炼体解锁条件）
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}
	if player.Realm < model.RealmBase {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "筑基期方可开启炼体"})
		return
	}

	// 首次训练时自动开启炼体
	info := h.bodyService.GetOrCreateBodyInfo(playerID)
	if info.Realm <= 0 {
		// 初始化：从铜皮1层开始
		h.bodyService.InitBodyCultivation(playerID)
	}

	expGained, err := h.bodyService.Train(playerID, &req)
	if err != nil {
		h.log.Warn("炼体训练失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 更新玩家炼体属性加成到总属性
	h.syncBodyBonuses(c, playerID, player)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "炼体训练成功",
		"data": gin.H{
			"exp_gained": expGained,
			"body_info":  h.bodyService.GetBodyInfo(playerID),
			"bonuses":    h.bodyService.GetStatus(playerID).Bonuses,
		},
	})
}

// Breakthrough 炼体突破
// POST /api/v1/body/breakthrough
func (h *BodyHandler) Breakthrough(c *gin.Context) {
	playerID, err := h.getPlayerID(c)
	if err != nil {
		return
	}

	var req model.BodyBreakthroughRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	success, err := h.bodyService.Breakthrough(playerID, &req)
	if err != nil {
		h.log.Warn("炼体突破失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 更新玩家属性
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err == nil {
		h.syncBodyBonuses(c, playerID, player)
	}

	status := h.bodyService.GetStatus(playerID)

	if success {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "炼体突破成功",
			"data": gin.H{
				"success":   true,
				"body_info": status.BodyInfo,
				"bonuses":   status.Bonuses,
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "炼体突破失败，HP上限永久降低（可通过休养恢复）",
			"data": gin.H{
				"success":   false,
				"body_info": status.BodyInfo,
				"bonuses":   status.Bonuses,
				"hp_lost":   status.BodyInfo.MaxHPLost,
			},
		})
	}
}

// GetStatus 查询炼体状态
// GET /api/v1/body/status
func (h *BodyHandler) GetStatus(c *gin.Context) {
	playerID, err := h.getPlayerID(c)
	if err != nil {
		return
	}

	// 检查玩家是否存在
	_, err = h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}

	status := h.bodyService.GetStatus(playerID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": status,
	})
}

// ---------- 内部方法 ----------

// getPlayerID 从请求中提取玩家ID（支持 query param 和 JSON body）
func (h *BodyHandler) getPlayerID(c *gin.Context) (int64, error) {
	// 优先从 query 参数取
	idStr := c.Query("player_id")
	if idStr == "" {
		// 其次从 JSON body 取
		var body struct {
			PlayerID int64 `json:"player_id"`
		}
		if err := c.ShouldBindJSON(&body); err == nil && body.PlayerID > 0 {
			return body.PlayerID, nil
		}
		// 最后从 URL param 取
		idStr = c.Param("player_id")
	}

	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少玩家ID"})
		return 0, fmt.Errorf("missing player_id")
	}

	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return 0, err
	}
	return playerID, nil
}

// syncBodyBonuses 将炼体属性加成同步到玩家总属性
func (h *BodyHandler) syncBodyBonuses(c *gin.Context, playerID int64, player *model.Player) {
	status := h.bodyService.GetStatus(playerID)
	if status.Bonuses == nil {
		return
	}

	// 炼体加成叠加到基础属性
	// 注意：这里假设玩家属性的 MaxHP 和 Defense 已包含其他加成
	// 炼体加成是额外增加的，需要服务端统一维护
	// 这里简化处理：只记录到日志，实际由 playerService 统一管理
	h.log.Debug("炼体属性加成",
		zap.Int64("玩家", playerID),
		zap.Int64("额外HP", status.Bonuses.HP),
		zap.Int64("额外防御", status.Bonuses.Defense),
		zap.Float64("伤害减免", status.Bonuses.DamageReduction),
	)
}

