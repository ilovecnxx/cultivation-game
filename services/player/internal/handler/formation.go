package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FormationHandler 阵法 HTTP 处理器
type FormationHandler struct {
	formationService *service.FormationService
	log              *zap.Logger
}

// NewFormationHandler 创建 FormationHandler
func NewFormationHandler(formationService *service.FormationService, log *zap.Logger) *FormationHandler {
	return &FormationHandler{
		formationService: formationService,
		log:              log,
	}
}

// extractPlayerID 提取 URL 中的玩家 ID
func extractPlayerID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

// ============================================================
// 基础操作（保留原有 + 增强响应）
// ============================================================

// Learn 学习阵法图谱
// POST /api/v1/player/:id/formation/learn
func (h *FormationHandler) Learn(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		TmplID int `json:"tmpl_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.LearnFormation(c.Request.Context(), playerID, req.TmplID)
	if err != nil {
		h.log.Warn("学习阵法失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "学习阵法成功",
		"data": formation,
	})
}

// List 查询玩家所有阵法
// GET /api/v1/player/:id/formation
func (h *FormationHandler) List(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formations, err := h.formationService.ListFormations(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询阵法列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": formations,
	})
}

// Deploy 部署阵法
// POST /api/v1/player/:id/formation/deploy
func (h *FormationHandler) Deploy(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		FormationID int64 `json:"formation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.DeployFormation(c.Request.Context(), playerID, req.FormationID)
	if err != nil {
		h.log.Warn("部署阵法失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "部署阵法成功",
		"data": formation,
	})
}

// Undeploy 撤销阵法部署
// POST /api/v1/player/:id/formation/undeploy
func (h *FormationHandler) Undeploy(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		FormationID int64 `json:"formation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.UndeployFormation(c.Request.Context(), playerID, req.FormationID)
	if err != nil {
		h.log.Warn("撤销部署失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "撤销部署成功",
		"data": formation,
	})
}

// Upgrade 升级阵法等级
// POST /api/v1/player/:id/formation/upgrade
func (h *FormationHandler) Upgrade(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		FormationID int64 `json:"formation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.UpgradeFormation(c.Request.Context(), playerID, req.FormationID)
	if err != nil {
		h.log.Warn("升级阵法失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "阵法升级成功",
		"data": gin.H{
			"formation": formation,
			"new_level": formation.Level,
		},
	})
}

// Guardian 设置护法阵法
// POST /api/v1/player/:id/formation/guardian
func (h *FormationHandler) Guardian(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		FormationID int64 `json:"formation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.SetGuardian(c.Request.Context(), playerID, req.FormationID)
	if err != nil {
		h.log.Warn("设置护法失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	bonus := service.GetGuardianBonus(formation)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "设置护法成功",
		"data": gin.H{
			"formation":  formation,
			"bonus_rate": bonus,
			"bonus_desc": "护法突破成功率 +" + strconv.Itoa(int(bonus*100)) + "%",
		},
	})
}

// UnsetGuardian 取消护法
// POST /api/v1/player/:id/formation/unguard
func (h *FormationHandler) UnsetGuardian(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	if err := h.formationService.UnsetGuardian(c.Request.Context(), playerID); err != nil {
		h.log.Warn("取消护法失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "取消护法成功"})
}

// GuardianHistory 查询护法记录
// GET /api/v1/player/:id/formation/guardian-history
func (h *FormationHandler) GuardianHistory(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	tasks, err := h.formationService.GetGuardianHistory(c.Request.Context(), playerID, limit)
	if err != nil {
		h.log.Warn("查询护法记录失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": tasks,
	})
}

// Templates 获取所有阵法图谱
// GET /api/v1/player/formation/templates
func (h *FormationHandler) Templates(c *gin.Context) {
	templates := h.formationService.GetAllTemplates()
	if templates == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": []any{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": templates,
	})
}

// DeployedBonuses 查询已部署阵法的战斗加成
// GET /api/v1/player/:id/formation/bonuses
func (h *FormationHandler) DeployedBonuses(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	effects, err := h.formationService.GetDeployedBonuses(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询阵法加成失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": effects,
	})
}

// ============================================================
// ========== 新增端点 ==========
// ============================================================

// ============================================================
// 1. 熟练度系统
// ============================================================

// MasteryInfo 查询阵法熟练度信息
// GET /api/v1/player/:id/formation/:fid/mastery
func (h *FormationHandler) MasteryInfo(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formationID, err := strconv.ParseInt(c.Param("fid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的阵法ID"})
		return
	}

	currentExp, needExp, level, levelName, err := h.formationService.GetFormationMasteryInfo(c.Request.Context(), playerID, formationID)
	if err != nil {
		h.log.Warn("查询熟练度失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"current_exp": currentExp,
			"need_exp":    needExp,
			"level":       level,
			"level_name":  levelName,
			"max_level":   level >= 10,
		},
	})
}

// ============================================================
// 2. 守护灵兽
// ============================================================

// AssignGuardian 指派守护灵兽
// POST /api/v1/player/:id/formation/:fid/guardian-pet
func (h *FormationHandler) AssignGuardian(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formationID, err := strconv.ParseInt(c.Param("fid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的阵法ID"})
		return
	}

	var req struct {
		PetID int64 `json:"pet_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.AssignGuardianPet(c.Request.Context(), playerID, formationID, req.PetID)
	if err != nil {
		h.log.Warn("指派守护灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "指派守护灵兽成功",
		"data": formation,
	})
}

// RemoveGuardian 解除守护灵兽
// DELETE /api/v1/player/:id/formation/:fid/guardian-pet
func (h *FormationHandler) RemoveGuardian(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formationID, err := strconv.ParseInt(c.Param("fid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的阵法ID"})
		return
	}

	formation, err := h.formationService.RemoveGuardianPet(c.Request.Context(), playerID, formationID)
	if err != nil {
		h.log.Warn("解除守护灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "解除守护灵兽成功",
		"data": formation,
	})
}

// ============================================================
// 3. 阵法联动
// ============================================================

// SetLink 设置联动组
// POST /api/v1/player/:id/formation/:fid/link
func (h *FormationHandler) SetLink(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formationID, err := strconv.ParseInt(c.Param("fid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的阵法ID"})
		return
	}

	var req struct {
		Group int `json:"group" binding:"required,min=1,max=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	formation, err := h.formationService.SetFormationLink(c.Request.Context(), playerID, formationID, req.Group)
	if err != nil {
		h.log.Warn("设置联动失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "设置联动成功",
		"data": gin.H{
			"formation": formation,
			"group":     req.Group,
		},
	})
}

// ClearLink 清除阵法联动
// DELETE /api/v1/player/:id/formation/:fid/link
func (h *FormationHandler) ClearLink(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	formationID, err := strconv.ParseInt(c.Param("fid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的阵法ID"})
		return
	}

	formation, err := h.formationService.ClearFormationLink(c.Request.Context(), playerID, formationID)
	if err != nil {
		h.log.Warn("清除联动失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "清除联动成功",
		"data": formation,
	})
}

// ClearAllLinks 清除所有联动组
// DELETE /api/v1/player/:id/formation/links
func (h *FormationHandler) ClearAllLinks(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	if err := h.formationService.ClearAllLinks(c.Request.Context(), playerID); err != nil {
		h.log.Warn("清除所有联动失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "清除所有联动成功"})
}

// LinkBonuses 查询联动加成详情
// GET /api/v1/player/:id/formation/links/bonuses
func (h *FormationHandler) LinkBonuses(c *gin.Context) {
	playerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	result, err := h.formationService.GetLinkBonuses(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询联动加成失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": result,
	})
}

// ============================================================
// 4. 阵法相克（破阵）
// ============================================================

// CalcBreak 计算 PVP 破阵效果
// GET /api/v1/player/:id/formation/break?defender_id=xxx
func (h *FormationHandler) CalcBreak(c *gin.Context) {
	attackerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	defenderIDStr := c.Query("defender_id")
	defenderID, err := strconv.ParseInt(defenderIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的防守方玩家ID"})
		return
	}

	result, err := h.formationService.CalcFormationBreak(c.Request.Context(), attackerID, defenderID)
	if err != nil {
		h.log.Warn("计算破阵效果失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": result,
	})
}

// ApplyBreak 应用破阵效果（在 PVP 战斗中使用）
// POST /api/v1/player/:id/formation/break/apply
func (h *FormationHandler) ApplyBreak(c *gin.Context) {
	attackerID, err := extractPlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		DefenderID int64 `json:"defender_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	modifier, bonusActive, err := h.formationService.ApplyFormationBreak(c.Request.Context(), attackerID, req.DefenderID)
	if err != nil {
		h.log.Warn("应用破阵效果失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "破阵计算完成",
		"data": gin.H{
			"defender_modifier":  modifier,
			"bonus_active":       bonusActive,
			"reduction_pct":      (1 - modifier) * 100,
		},
	})
}
