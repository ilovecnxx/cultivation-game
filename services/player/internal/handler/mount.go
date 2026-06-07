// Package handler 坐骑系统HTTP处理器
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MountHandler 坐骑 HTTP 处理器
type MountHandler struct {
	mountService *service.MountService
	log          *zap.Logger
}

// NewMountHandler 创建 MountHandler
func NewMountHandler(mountService *service.MountService, log *zap.Logger) *MountHandler {
	return &MountHandler{
		mountService: mountService,
		log:          log,
	}
}

// parsePlayerID 从查询参数解析玩家ID
func (h *MountHandler) parsePlayerID(c *gin.Context) (int64, error) {
	pid := c.Query("player_id")
	if pid == "" {
		pid = c.Param("player_id")
	}
	return strconv.ParseInt(pid, 10, 64)
}

// ListMounts 获取坐骑列表
// GET /api/v1/mount/list?player_id=xxx
func (h *MountHandler) ListMounts(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	mounts, err := h.mountService.GetMountList(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询坐骑列表失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": mounts,
	})
}

// EquipMount 装备坐骑
// POST /api/v1/mount/equip
func (h *MountHandler) EquipMount(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
		MountID  int64 `json:"mount_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	mount, err := h.mountService.EquipMount(c.Request.Context(), req.PlayerID, req.MountID)
	if err != nil {
		h.log.Warn("装备坐骑失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "装备坐骑成功",
		"data": mount,
	})
}

// UpgradeMount 坐骑升级
// POST /api/v1/mount/upgrade
func (h *MountHandler) UpgradeMount(c *gin.Context) {
	var req struct {
		PlayerID int64 `json:"player_id" binding:"required"`
		MountID  int64 `json:"mount_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	mount, err := h.mountService.UpgradeMount(c.Request.Context(), req.PlayerID, req.MountID)
	if err != nil {
		h.log.Warn("坐骑升级失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "坐骑升级成功",
		"data": mount,
	})
}
