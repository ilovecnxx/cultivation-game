// Package handler 提供师徒系统的 HTTP API 处理器
package handler

import (
	"net/http"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// MasterHandler 师徒 HTTP 处理器
type MasterHandler struct {
	svc *service.MasterService
}

// NewMasterHandler 创建师徒处理器
func NewMasterHandler(svc *service.MasterService) *MasterHandler {
	return &MasterHandler{svc: svc}
}

// ============================================================
// 申请拜师/收徒
// POST /api/v1/master/apply
// ============================================================

// Apply 发起拜师或收徒申请
func (h *MasterHandler) Apply(c *gin.Context) {
	var req struct {
		FromID    string `json:"from_id" binding:"required"`
		FromName  string `json:"from_name" binding:"required"`
		ToID      string `json:"to_id" binding:"required"`
		ToName    string `json:"to_name" binding:"required"`
		ApplyType string `json:"apply_type" binding:"required"` // as_student / as_master
		Message   string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Apply(c.Request.Context(), req.FromID, req.FromName, req.ToID, req.ToName, model.MasterApplyType(req.ApplyType), req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "申请已发送"})
}

// ============================================================
// 同意申请
// POST /api/v1/master/accept
// ============================================================

// Accept 同意拜师/收徒申请
func (h *MasterHandler) Accept(c *gin.Context) {
	var req struct {
		ApplyID string `json:"apply_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Accept(c.Request.Context(), req.ApplyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已同意申请，师徒关系建立"})
}

// ============================================================
// 拒绝申请
// POST /api/v1/master/reject
// ============================================================

// Reject 拒绝拜师/收徒申请
func (h *MasterHandler) Reject(c *gin.Context) {
	var req struct {
		ApplyID string `json:"apply_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Reject(c.Request.Context(), req.ApplyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已拒绝申请"})
}

// ============================================================
// 待处理申请列表
// GET /api/v1/master/pending-applies?user_id=xxx
// ============================================================

// GetPendingApplies 获取发给自己的待处理申请
func (h *MasterHandler) GetPendingApplies(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	applies, err := h.svc.GetPendingApplies(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": applies})
}

// ============================================================
// 获取我的师父
// GET /api/v1/master/my-master?user_id=xxx
// ============================================================

// GetMyMaster 获取玩家的师父
func (h *MasterHandler) GetMyMaster(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	master, err := h.svc.GetMyMaster(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if master == nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": master})
}

// ============================================================
// 获取我的徒弟列表
// GET /api/v1/master/my-students?user_id=xxx
// ============================================================

// GetMyStudents 获取玩家的徒弟列表
func (h *MasterHandler) GetMyStudents(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	students, err := h.svc.GetMyStudents(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": students})
}

// ============================================================
// 传授功法
// POST /api/v1/master/teach
// ============================================================

// Teach 师父传授功法给徒弟
func (h *MasterHandler) Teach(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
		SkillID    string `json:"skill_id" binding:"required"`
		SkillName  string `json:"skill_name" binding:"required"`
		CostMV     int64  `json:"cost_mv" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Teach(c.Request.Context(), req.RelationID, req.SkillID, req.SkillName, req.CostMV); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "传授成功"})
}

// ============================================================
// 获取每日任务
// GET /api/v1/master/missions?relation_id=xxx
// ============================================================

// GetDailyMissions 获取师徒每日任务
func (h *MasterHandler) GetDailyMissions(c *gin.Context) {
	relationID := c.Query("relation_id")
	if relationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 relation_id"})
		return
	}

	missions, err := h.svc.GetDailyMissions(c.Request.Context(), relationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": missions})
}

// ============================================================
// 更新任务进度
// POST /api/v1/master/mission/progress
// ============================================================

// UpdateMissionProgress 更新师徒任务进度
func (h *MasterHandler) UpdateMissionProgress(c *gin.Context) {
	var req struct {
		MissionID   string `json:"mission_id" binding:"required"`
		AddProgress int32  `json:"add_progress" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.UpdateMissionProgress(c.Request.Context(), req.MissionID, req.AddProgress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "进度已更新"})
}

// ============================================================
// 领取任务奖励
// POST /api/v1/master/mission/claim
// ============================================================

// ClaimMission 领取师徒任务奖励
func (h *MasterHandler) ClaimMission(c *gin.Context) {
	var req struct {
		MissionID string `json:"mission_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.ClaimMission(c.Request.Context(), req.MissionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "奖励已领取"})
}

// ============================================================
// 出师
// POST /api/v1/master/graduate
// ============================================================

// Graduate 徒弟出师
func (h *MasterHandler) Graduate(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Graduate(c.Request.Context(), req.RelationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "出师成功，双方获得大量奖励"})
}

// ============================================================
// 逐出师门
// POST /api/v1/master/kick
// ============================================================

// Kick 师父将徒弟逐出师门
func (h *MasterHandler) Kick(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Kick(c.Request.Context(), req.RelationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已逐出师门"})
}

// ============================================================
// 师徒值查询
// GET /api/v1/master/master-value?relation_id=xxx
// ============================================================

// GetMasterValue 获取师徒关系的师徒值
func (h *MasterHandler) GetMasterValue(c *gin.Context) {
	relationID := c.Query("relation_id")
	if relationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 relation_id"})
		return
	}

	mv, err := h.svc.GetMasterValue(c.Request.Context(), relationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mv})
}

// ============================================================
// 师徒等级(新增)
// ============================================================

// GetMentorshipLevelInfo 获取师徒等级详情
// GET /api/v1/master/level?relation_id=xxx
func (h *MasterHandler) GetMentorshipLevelInfo(c *gin.Context) {
	relationID := c.Query("relation_id")
	if relationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 relation_id"})
		return
	}

	info, err := h.svc.GetMentorshipLevelInfo(c.Request.Context(), relationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

// UpgradeMentorshipLevel 提升师徒等级
// POST /api/v1/master/level/upgrade
func (h *MasterHandler) UpgradeMentorshipLevel(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.TryUpgradeMentorshipLevel(c.Request.Context(), req.RelationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "师徒等级提升成功"})
}

// ============================================================
// 每日训练(新增)
// ============================================================

// AssignDailyTraining 师父分配训练任务
// POST /api/v1/master/training/assign
func (h *MasterHandler) AssignDailyTraining(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
		TaskType   string `json:"task_type" binding:"required"`
		Target     int32  `json:"target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.AssignDailyTraining(c.Request.Context(), req.RelationID, req.TaskType, req.Target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "训练任务已分配"})
}

// GetDailyTraining 获取今日训练任务
// GET /api/v1/master/training?relation_id=xxx
func (h *MasterHandler) GetDailyTraining(c *gin.Context) {
	relationID := c.Query("relation_id")
	if relationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 relation_id"})
		return
	}

	training, err := h.svc.GetDailyTraining(c.Request.Context(), relationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": training})
}

// UpdateTrainingProgress 更新训练进度
// POST /api/v1/master/training/progress
func (h *MasterHandler) UpdateTrainingProgress(c *gin.Context) {
	var req struct {
		MissionID   string `json:"mission_id" binding:"required"`
		AddProgress int32  `json:"add_progress" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.UpdateTrainingProgress(c.Request.Context(), req.MissionID, req.AddProgress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "训练进度已更新"})
}

// ClaimTrainingReward 领取训练奖励
// POST /api/v1/master/training/claim
func (h *MasterHandler) ClaimTrainingReward(c *gin.Context) {
	var req struct {
		MissionID string `json:"mission_id" binding:"required"`
		UserID    string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.ClaimTrainingReward(c.Request.Context(), req.MissionID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "训练奖励已领取"})
}

// ============================================================
// 叛离师门(新增)
// ============================================================

// Betray 徒弟叛离师门
// POST /api/v1/master/betray
func (h *MasterHandler) Betray(c *gin.Context) {
	var req struct {
		RelationID string `json:"relation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.Betray(c.Request.Context(), req.RelationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已叛离师门，失去所有师徒加成"})
}

// GetBetrayalHistory 获取叛离历史
// GET /api/v1/master/betray-history?user_id=xxx
func (h *MasterHandler) GetBetrayalHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_id"})
		return
	}

	records, err := h.svc.GetBetrayalHistory(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": records})
}

// ============================================================
// 师徒副本(新增)
// ============================================================

// CreateDungeon 创建副本实例
// POST /api/v1/master/dungeon/create
func (h *MasterHandler) CreateDungeon(c *gin.Context) {
	var req struct {
		RelationID  string `json:"relation_id" binding:"required"`
		DungeonLevel int   `json:"dungeon_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	instance, err := h.svc.CreateDungeonInstance(c.Request.Context(), req.RelationID, req.DungeonLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instance})
}

// EnterDungeon 进入副本
// POST /api/v1/master/dungeon/enter
func (h *MasterHandler) EnterDungeon(c *gin.Context) {
	var req struct {
		InstanceID string `json:"instance_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.EnterDungeon(c.Request.Context(), req.InstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已进入副本"})
}

// DungeonWaveComplete 完成一波
// POST /api/v1/master/dungeon/wave-complete
func (h *MasterHandler) DungeonWaveComplete(c *gin.Context) {
	var req struct {
		InstanceID string `json:"instance_id" binding:"required"`
		MasterDmg  int64  `json:"master_dmg"`
		StudentDmg int64  `json:"student_dmg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	instance, err := h.svc.DungeonWaveComplete(c.Request.Context(), req.InstanceID, req.MasterDmg, req.StudentDmg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instance})
}

// ClaimDungeonReward 领取副本奖励
// POST /api/v1/master/dungeon/claim
func (h *MasterHandler) ClaimDungeonReward(c *gin.Context) {
	var req struct {
		InstanceID string `json:"instance_id" binding:"required"`
		UserID     string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	reward, err := h.svc.ClaimDungeonReward(c.Request.Context(), req.InstanceID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "奖励已领取",
		"data":    reward,
	})
}

// GetDungeonInstance 获取副本状态
// GET /api/v1/master/dungeon/status?instance_id=xxx
func (h *MasterHandler) GetDungeonInstance(c *gin.Context) {
	instanceID := c.Query("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 instance_id"})
		return
	}

	instance, err := h.svc.GetDungeonInstance(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instance})
}
