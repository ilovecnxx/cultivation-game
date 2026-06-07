package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DongFuHandler 洞府 HTTP 处理器
type DongFuHandler struct {
	dongfuService *service.DongFuService
	log           *zap.Logger
}

// NewDongFuHandler 创建 DongFuHandler
func NewDongFuHandler(dongfuService *service.DongFuService, log *zap.Logger) *DongFuHandler {
	return &DongFuHandler{
		dongfuService: dongfuService,
		log:           log,
	}
}

// ---------- 洞府基础 ----------

// Build 建造洞府
// POST /api/v1/player/:id/dongfu/build
func (h *DongFuHandler) Build(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.BuildDongFuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	dongfu, err := h.dongfuService.Build(c.Request.Context(), playerID, req.Name)
	if err != nil {
		h.log.Warn("建造洞府失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "建造洞府成功",
		"data": dongfu,
	})
}

// GetDongFu 查看洞府
// GET /api/v1/player/:id/dongfu
func (h *DongFuHandler) GetDongFu(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.dongfuService.GetDongFu(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询洞府失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// ---------- 房间操作 ----------

// BuildRoom 建造房间
// POST /api/v1/player/:id/dongfu/room/build
func (h *DongFuHandler) BuildRoom(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.BuildRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	room, err := h.dongfuService.BuildRoom(c.Request.Context(), playerID, req.RoomType)
	if err != nil {
		h.log.Warn("建造房间失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "建造房间成功",
		"data": room,
	})
}

// UpgradeRoom 升级房间
// POST /api/v1/player/:id/dongfu/room/upgrade
func (h *DongFuHandler) UpgradeRoom(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.UpgradeRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	room, err := h.dongfuService.UpgradeRoom(c.Request.Context(), playerID, req.RoomID)
	if err != nil {
		h.log.Warn("升级房间失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "升级房间成功",
		"data": gin.H{
			"room":  room,
			"level": room.Level,
			"bonus": room.Bonus,
		},
	})
}

// GetRoomDetail 获取房间详情
// GET /api/v1/player/:id/dongfu/room/:room_id
func (h *DongFuHandler) GetRoomDetail(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	roomID, err := strconv.ParseInt(c.Param("room_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的房间ID"})
		return
	}

	// 通过 GetDongFu 查找房间详情
	resp, err := h.dongfuService.GetDongFu(c.Request.Context(), playerID)
	if err != nil || resp.DongFu == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "洞府不存在"})
		return
	}

	for _, room := range resp.DongFu.Rooms {
		if room.ID == roomID {
			detail := h.dongfuService.ToRoomDetail(&room)
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": detail,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "房间不存在"})
}

// ---------- 灵气汇聚 ----------

// StartGathering 开始灵气汇聚
// POST /api/v1/player/:id/dongfu/gathering/start
func (h *DongFuHandler) StartGathering(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.StartGatheringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	gathering, err := h.dongfuService.StartGathering(c.Request.Context(), playerID, req.Duration)
	if err != nil {
		h.log.Warn("开始灵气汇聚失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "开始灵气汇聚",
		"data": gathering,
	})
}

// CollectGathering 领取灵气汇聚收益
// POST /api/v1/player/:id/dongfu/gathering/collect
func (h *DongFuHandler) CollectGathering(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	gathering, err := h.dongfuService.CollectGathering(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("领取灵气汇聚失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": gathering,
	})
}

// GetGatheringStatus 获取灵气汇聚状态
// GET /api/v1/player/:id/dongfu/gathering/status
func (h *DongFuHandler) GetGatheringStatus(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	gathering, err := h.dongfuService.GetGatheringStatus(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询灵气汇聚失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gathering,
	})
}

// ---------- 装饰系统 ----------

// PlaceDecoration 摆放装饰
// POST /api/v1/player/:id/dongfu/decorate
func (h *DongFuHandler) PlaceDecoration(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.PlaceDecorationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	decoration, err := h.dongfuService.PlaceDecoration(c.Request.Context(), playerID, req)
	if err != nil {
		h.log.Warn("摆放装饰失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "摆放装饰成功",
		"data": decoration,
	})
}

// RemoveDecoration 移除装饰
// DELETE /api/v1/player/:id/dongfu/decorate/:decoration_id
func (h *DongFuHandler) RemoveDecoration(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	decorationID, err := strconv.ParseInt(c.Param("decoration_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的装饰ID"})
		return
	}

	if err := h.dongfuService.RemoveDecoration(c.Request.Context(), playerID, decorationID); err != nil {
		h.log.Warn("移除装饰失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "移除装饰成功",
	})
}

// ListDecorations 获取装饰列表
// GET /api/v1/player/:id/dongfu/decorations
func (h *DongFuHandler) ListDecorations(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	decorations, err := h.dongfuService.ListDecorations(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询装饰列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": decorations,
	})
}

// ---------- 访客系统 ----------

// InviteGuest 邀请访客
// POST /api/v1/player/:id/dongfu/guest/invite
func (h *DongFuHandler) InviteGuest(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.InviteGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	guest, err := h.dongfuService.InviteGuest(c.Request.Context(), playerID, req.GuestPlayerID)
	if err != nil {
		h.log.Warn("邀请访客失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "邀请发送成功",
		"data": guest,
	})
}

// GuestAction 处理访客邀请
// POST /api/v1/player/:id/dongfu/guest/action
func (h *DongFuHandler) GuestAction(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req model.GuestActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	guest, err := h.dongfuService.GuestAction(c.Request.Context(), playerID, req)
	if err != nil {
		h.log.Warn("处理访客操作失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "操作成功",
		"data": guest,
	})
}

// GetGuests 获取访客列表
// GET /api/v1/player/:id/dongfu/guests
func (h *DongFuHandler) GetGuests(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	guests, err := h.dongfuService.GetGuests(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询访客列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": guests,
	})
}

// GetInvitations 获取邀请列表
// GET /api/v1/player/:id/dongfu/invitations
func (h *DongFuHandler) GetInvitations(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	invitations, err := h.dongfuService.GetInvitations(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询邀请列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": invitations,
	})
}

// ---------- 被动收益 ----------

// GetPassiveRewards 获取洞府被动收益信息
// GET /api/v1/player/:id/dongfu/passive
func (h *DongFuHandler) GetPassiveRewards(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	combatExp, stones, err := h.dongfuService.GetPassiveRewards(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询被动收益失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"combat_exp_per_hour": combatExp,
			"spirit_stones_per_hour": stones,
		},
	})
}

// CollectPassiveRewards 领取被动收益
// POST /api/v1/player/:id/dongfu/passive/collect
func (h *DongFuHandler) CollectPassiveRewards(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	combatExp, stones, err := h.dongfuService.CollectPassiveRewards(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("领取被动收益失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "领取成功",
		"data": gin.H{
			"combat_exp":    combatExp,
			"spirit_stones": stones,
		},
	})
}

// GetDongFuLevelThresholds 获取洞府等级解锁阈值
// GET /api/v1/player/dongfu/thresholds
func (h *DongFuHandler) GetDongFuLevelThresholds(c *gin.Context) {
	thresholds := h.dongfuService.GetDongFuLevelThresholds()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"required_level":       h.dongfuService.GetRequiredLevel(),
			"build_cost":           h.dongfuService.GetBuildCost(),
			"room_build_cost":      h.dongfuService.GetRoomBuildCost(),
			"room_upgrade_cost_base": h.dongfuService.GetRoomUpgradeCostBase(),
			"feature_thresholds":   thresholds,
		},
	})
}
