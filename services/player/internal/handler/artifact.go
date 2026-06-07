package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ArtifactHandler 法宝 HTTP 处理器
type ArtifactHandler struct {
	artifactService *service.ArtifactService
	log             *zap.Logger
}

// NewArtifactHandler 创建 ArtifactHandler
func NewArtifactHandler(artifactService *service.ArtifactService, log *zap.Logger) *ArtifactHandler {
	return &ArtifactHandler{
		artifactService: artifactService,
		log:             log,
	}
}

// ============================================================
// 基础操作
// ============================================================

// BindArtifact 绑定本命法宝
// POST /api/v1/player/:id/artifact/bind
func (h *ArtifactHandler) BindArtifact(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=32"`
		Type int    `json:"type" binding:"required,min=1,max=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	artifact, err := h.artifactService.BindArtifact(c.Request.Context(), playerID, req.Name, req.Type)
	if err != nil {
		h.log.Warn("绑定法宝失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "绑定法宝成功",
		"data": artifact,
	})
}

// ListArtifacts 获取玩家所有法宝
// GET /api/v1/player/:id/artifacts
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resp, err := h.artifactService.ListArtifacts(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询法宝列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

// UpgradeArtifact 升级法宝
// POST /api/v1/player/:id/artifact/:aid/upgrade
func (h *ArtifactHandler) UpgradeArtifact(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	artifact, err := h.artifactService.UpgradeArtifact(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("升级法宝失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "法宝升级成功",
		"data": gin.H{
			"artifact":      artifact,
			"new_level":     artifact.Level,
			"attack_bonus":  artifact.AttackBonus,
			"defense_bonus": artifact.DefenseBonus,
			"hp_bonus":      artifact.HPBonus,
			"power_bonus":   artifact.PowerBonus,
		},
	})
}

// ============================================================
// 进化
// ============================================================

// EvolveArtifact 进化法宝（消耗材料升级/升品）
// POST /api/v1/player/:id/artifact/:aid/evolve
func (h *ArtifactHandler) EvolveArtifact(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	result, err := h.artifactService.EvolveArtifact(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("进化法宝失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  result.Msg,
		"data": result,
	})
}

// ============================================================
// 觉醒
// ============================================================

// GetAwakenInfo 获取觉醒信息
// GET /api/v1/player/:id/artifact/:aid/awaken
func (h *ArtifactHandler) GetAwakenInfo(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	milestones, unlocked, err := h.artifactService.GetAwakenInfo(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("查询觉醒信息失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"milestones": milestones,
			"unlocked":   unlocked,
		},
	})
}

// AwakenArtifact 觉醒法宝技能
// POST /api/v1/player/:id/artifact/:aid/awaken
func (h *ArtifactHandler) AwakenArtifact(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	var req struct {
		SlotIndex int `json:"slot" binding:"required,min=0,max=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	result, err := h.artifactService.AwakenArtifact(c.Request.Context(), playerID, artifactID, req.SlotIndex)
	if err != nil {
		h.log.Warn("觉醒法宝失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  fmt.Sprintf("觉醒成功！获得技能[%s]", result.SkillName),
		"data": result,
	})
}

// ============================================================
// 器灵
// ============================================================

// ActivateSpirit 激活器灵
// POST /api/v1/player/:id/artifact/:aid/spirit
func (h *ArtifactHandler) ActivateSpirit(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=32"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	spirit, err := h.artifactService.ActivateSpirit(c.Request.Context(), playerID, artifactID, req.Name)
	if err != nil {
		h.log.Warn("激活器灵失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "器灵激活成功",
		"data": spirit,
	})
}

// GetSpirit 获取器灵信息
// GET /api/v1/player/:id/artifact/:aid/spirit
func (h *ArtifactHandler) GetSpirit(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, _ := strconv.ParseInt(c.Param("aid"), 10, 64)

	spirit, err := h.artifactService.GetSpirit(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("查询器灵失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if spirit == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "未激活器灵", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": spirit,
	})
}

// InteractSpirit 与器灵互动
// POST /api/v1/player/:id/artifact/spirit/:sid/interact
func (h *ArtifactHandler) InteractSpirit(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	spiritID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的器灵ID"})
		return
	}

	spirit, dialogue, err := h.artifactService.InteractSpirit(c.Request.Context(), playerID, spiritID)
	if err != nil {
		h.log.Warn("与器灵互动失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  dialogue,
		"data": spirit,
	})
}

// ============================================================
// 共鸣
// ============================================================

// GetResonance 获取共鸣信息
// GET /api/v1/player/:id/artifact/resonance
func (h *ArtifactHandler) GetResonance(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	resonance, err := h.artifactService.GetResonance(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询共鸣信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resonance,
	})
}

// ============================================================
// 试炼
// ============================================================

// GetTrialStages 获取试炼关卡
// GET /api/v1/player/:id/artifact/:aid/trials
func (h *ArtifactHandler) GetTrialStages(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}

	stages, progress, err := h.artifactService.GetTrialStages(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("查询试炼关卡失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"stages":   stages,
			"progress": progress,
		},
	})
}

// EnterTrial 进入试炼
// POST /api/v1/player/:id/artifact/:aid/trials/:stageId
func (h *ArtifactHandler) EnterTrial(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	artifactID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的法宝ID"})
		return
	}
	stageID, err := strconv.ParseInt(c.Param("stageId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的试炼ID"})
		return
	}

	result, err := h.artifactService.EnterTrial(c.Request.Context(), playerID, artifactID, int(stageID))
	if err != nil {
		h.log.Warn("进入试炼失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	msg := "试炼失败"
	if result.Victory {
		msg = "试炼通关！"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  msg,
		"data": result,
	})
}

// ============================================================
// 查询（兼容旧接口 + 增强）
// ============================================================

// GetArtifact 获取法宝信息
// GET /api/v1/player/:id/artifact/:aid
// GET /api/v1/player/:id/artifact (默认主法宝)
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	// 可选的法宝ID参数
	artifactIDStr := c.Param("aid")
	var artifactID int64
	if artifactIDStr != "" {
		artifactID, _ = strconv.ParseInt(artifactIDStr, 10, 64)
	}

	resp, err := h.artifactService.GetArtifactBonus(c.Request.Context(), playerID, artifactID)
	if err != nil {
		h.log.Warn("查询法宝失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}
