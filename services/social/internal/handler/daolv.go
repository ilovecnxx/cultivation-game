// Package handler 提供道侣系统 HTTP 接口
package handler

import (
	"fmt"
	"net/http"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// DaoLvHandler 道侣系统 HTTP 处理器
type DaoLvHandler struct {
	svc *service.DaoLvService
}

// NewDaoLvHandler 创建道侣处理器
func NewDaoLvHandler(svc *service.DaoLvService) *DaoLvHandler {
	return &DaoLvHandler{svc: svc}
}

// ============================================================
// 求婚系统
// ============================================================

// Propose 发送求婚
// @Router POST /api/v1/daolv/propose [post]
func (h *DaoLvHandler) Propose(c *gin.Context) {
	var req struct {
		FromID       uint64 `json:"from_id" binding:"required"`
		FromName     string `json:"from_name"`
		ToID         uint64 `json:"to_id" binding:"required"`
		ToName       string `json:"to_name"`
		Message      string `json:"message"`
		GiftItemID   string `json:"gift_item_id"`
		GiftItemName string `json:"gift_item_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	err := h.svc.Propose(c.Request.Context(), &service.ProposeRequest{
		FromID:       req.FromID,
		FromName:     req.FromName,
		ToID:         req.ToID,
		ToName:       req.ToName,
		Message:      req.Message,
		GiftItemID:   req.GiftItemID,
		GiftItemName: req.GiftItemName,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "求婚申请已发送", "data": gin.H{"status": "pending"}})
}

// HandleProposal 处理求婚申请(接受/拒绝)
// @Router POST /api/v1/daolv/handle-proposal [post]
func (h *DaoLvHandler) HandleProposal(c *gin.Context) {
	var req struct {
		ProposalID string `json:"proposal_id" binding:"required"`
		Action     string `json:"action" binding:"required"` // accept / reject
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action 必须为 accept 或 reject"})
		return
	}

	err := h.svc.HandleProposal(c.Request.Context(), req.ProposalID, req.Action == "accept")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := "已拒绝求婚"
	if req.Action == "accept" {
		msg = "结为道侣成功"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// GetProposals 获取玩家的求婚申请列表
// @Router GET /api/v1/daolv/proposals/:playerID [get]
func (h *DaoLvHandler) GetProposals(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	if playerIDStr == "" {
		playerIDStr = c.Query("player_id")
	}
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 playerID"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerIDStr, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerID 格式错误"})
		return
	}

	proposals, err := h.svc.GetProposals(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if proposals == nil {
		proposals = make([]*model.DaolvProposal, 0)
	}
	c.JSON(http.StatusOK, gin.H{"data": proposals})
}

// GetPendingProposals 获取待处理的求婚申请
// @Router GET /api/v1/daolv/pending-proposals/:playerID [get]
func (h *DaoLvHandler) GetPendingProposals(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	if playerIDStr == "" {
		playerIDStr = c.Query("player_id")
	}
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 playerID"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerIDStr, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerID 格式错误"})
		return
	}

	proposals, err := h.svc.GetPendingProposals(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []struct{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": proposals})
}

// ============================================================
// 道侣状态
// ============================================================

// GetStatus 获取道侣状态
// @Router GET /api/v1/daolv/status/:playerID [get]
func (h *DaoLvHandler) GetStatus(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 playerID"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerIDStr, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerID 格式错误"})
		return
	}

	status, err := h.svc.GetStatus(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// ============================================================
// 双修系统
// ============================================================

// StartDualCultivate 开始双修
// @Router POST /api/v1/daolv/dual-cultivate [post]
func (h *DaoLvHandler) StartDualCultivate(c *gin.Context) {
	var req struct {
		PlayerID  uint64 `json:"player_id" binding:"required"`
		Duration  int    `json:"duration"`  // 修炼时长(秒)，默认1800(30分钟)
		Technique string `json:"technique"` // 双修功法(可选)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.Duration <= 0 {
		req.Duration = 1800 // 默认30分钟
	}

	result, err := h.svc.StartDualCultivate(c.Request.Context(), &service.DualCultivateRequest{
		PlayerID: req.PlayerID,
		Duration: time.Duration(req.Duration) * time.Second,
		Technique: req.Technique,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "双修完成",
		"data":    result,
	})
}

// StopDualCultivate 停止双修(预留接口)
// @Router POST /api/v1/daolv/stop-cultivate [post]
func (h *DaoLvHandler) StopDualCultivate(c *gin.Context) {
	var req struct {
		PlayerID uint64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "双修已停止"})
}

// ============================================================
// 道侣技能
// ============================================================

// UseSkill 使用道侣技能
// @Router POST /api/v1/daolv/skill/:skillName [post]
func (h *DaoLvHandler) UseSkill(c *gin.Context) {
	skillName := c.Param("skillName")
	if skillName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 skillName"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	result, err := h.svc.UseSkill(c.Request.Context(), req.PlayerID, skillName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "技能使用成功",
		"data":    result,
	})
}

// ============================================================
// 道侣任务
// ============================================================

// GetTasks 获取道侣任务
// @Router GET /api/v1/daolv/tasks/:playerID [get]
func (h *DaoLvHandler) GetTasks(c *gin.Context) {
	playerIDStr := c.Param("playerID")
	if playerIDStr == "" {
		playerIDStr = c.Query("player_id")
	}
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 playerID"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerIDStr, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerID 格式错误"})
		return
	}

	tasks, err := h.svc.GetTasks(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []struct{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// ClaimTask 领取任务奖励
// @Router POST /api/v1/daolv/task/claim [post]
func (h *DaoLvHandler) ClaimTask(c *gin.Context) {
	var req struct {
		TaskID     string `json:"task_id" binding:"required"`
		RelationID string `json:"relation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	reward, err := h.svc.ClaimTask(c.Request.Context(), req.TaskID, req.RelationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "奖励已领取",
		"data":    reward,
	})
}

// ============================================================
// 解除道侣关系
// ============================================================

// Dissolve 解除道侣关系
// @Router POST /api/v1/daolv/dissolve [post]
func (h *DaoLvHandler) Dissolve(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
		PlayerID   uint64 `json:"player_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	err := h.svc.Dissolve(c.Request.Context(), req.RelationID, req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "道侣关系已解除"})
}

// ============================================================
// 旧接口(保持兼容)
// ============================================================

// OldPropose 旧版求婚接口
func (h *DaoLvHandler) OldPropose(c *gin.Context) {
	var req struct {
		FromID uint64 `json:"from_id"`
		ToID   uint64 `json:"to_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "请使用新版求婚接口 /api/v1/daolv/propose"})
}

// OldAccept 旧版接受求婚接口
func (h *DaoLvHandler) OldAccept(c *gin.Context) {
	var req struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "请使用 /api/v1/daolv/handle-proposal 接口"})
}

// OldReject 旧版拒绝求婚接口
func (h *DaoLvHandler) OldReject(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "请使用 /api/v1/daolv/handle-proposal 接口"})
}

// OldDivorce 旧版解除道侣关系
func (h *DaoLvHandler) OldDivorce(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "请使用 /api/v1/daolv/dissolve 接口"})
}

// OldDualCultivate 旧版双修接口
func (h *DaoLvHandler) OldDualCultivate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "请使用新版双修接口 /api/v1/daolv/dual-cultivate"})
}

// OldSendGift 旧版送礼接口
func (h *DaoLvHandler) OldSendGift(c *gin.Context) {
	var req struct {
		FromID uint64 `json:"from_id"`
		ToID   uint64 `json:"to_id"`
		ItemID uint64 `json:"item_id"`
		Qty    int    `json:"qty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.svc.SendGift(c.Request.Context(), req.FromID, req.ToID, req.ItemID, req.Qty); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "礼物已赠送"})
}

// OldTeleport 旧版传送接口
func (h *DaoLvHandler) OldTeleport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "请使用技能接口 /api/v1/daolv/skill/传送"})
}

// OldGetInfo 旧版查询接口
func (h *DaoLvHandler) OldGetInfo(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerID, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_id 格式错误"})
		return
	}
	status, err := h.svc.GetStatus(c.Request.Context(), pid)
	if err != nil || !status.HasPartner {
		c.JSON(http.StatusOK, gin.H{"data": nil, "message": "暂无道侣"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// OldGetProposals 旧版查询申请
func (h *DaoLvHandler) OldGetProposals(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}
	var pid uint64
	if _, err := fmt.Sscanf(playerID, "%d", &pid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_id 格式错误"})
		return
	}
	proposals, err := h.svc.GetProposals(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": proposals})
}
