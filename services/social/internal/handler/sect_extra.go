package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// SectExtraHandler 宗门扩展功能 HTTP 处理器
type SectExtraHandler struct {
	skillSvc   *service.SectSkillService
	missionSvc *service.SectMissionService
	warSvc     *service.SectWarService
}

// NewSectExtraHandler 创建宗门扩展处理器
func NewSectExtraHandler(
	skillSvc *service.SectSkillService,
	missionSvc *service.SectMissionService,
	warSvc *service.SectWarService,
) *SectExtraHandler {
	return &SectExtraHandler{
		skillSvc:   skillSvc,
		missionSvc: missionSvc,
		warSvc:     warSvc,
	}
}

// ============================================================
// 宗门技能
// ============================================================

// GetSkillTree 查看宗门技能树
// @Router POST /api/v1/sect/skill/list [post]
func (h *SectExtraHandler) GetSkillTree(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	skills, err := h.skillSvc.GetSkillTree(c.Request.Context(), req.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": skills})
}

// LearnSkill 学习宗门技能
// @Router POST /api/v1/sect/skill/learn [post]
func (h *SectExtraHandler) LearnSkill(c *gin.Context) {
	var req struct {
		SectID  string `json:"sect_id"`
		UserID  string `json:"user_id"`
		SkillID string `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	result, err := h.skillSvc.LearnMemberSkill(c.Request.Context(), req.SectID, req.UserID, req.SkillID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "message": "学习成功"})
}

// UpgradeSectSkill 升级宗门技能（仅长老以上）
// @Router POST /api/v1/sect/skill/upgrade [post]
func (h *SectExtraHandler) UpgradeSectSkill(c *gin.Context) {
	var req struct {
		SectID  string `json:"sect_id"`
		UserID  string `json:"user_id"`
		SkillID string `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	result, err := h.skillSvc.UpgradeSectSkill(c.Request.Context(), req.SectID, req.UserID, req.SkillID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "message": "升级成功"})
}

// ============================================================
// 宗门任务
// ============================================================

// GetDailyMissions 查看当日任务
// @Router POST /api/v1/sect/mission/list [post]
func (h *SectExtraHandler) GetDailyMissions(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	missions, err := h.missionSvc.GetDailyMissions(c.Request.Context(), req.SectID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": missions})
}

// ClaimMission 领取任务奖励
// @Router POST /api/v1/sect/mission/claim [post]
func (h *SectExtraHandler) ClaimMission(c *gin.Context) {
	var req struct {
		MemberMissionID string `json:"member_mission_id"`
		SectID          string `json:"sect_id"`
		UserID          string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	err := h.missionSvc.ClaimMissionReward(c.Request.Context(), req.MemberMissionID, req.SectID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "领取成功"})
}

// ============================================================
// 宗门战 - 赛季管理
// ============================================================

// GetWarSeason 获取当前赛季信息
// @Router GET /api/v1/sect/war/season [get]
func (h *SectExtraHandler) GetWarSeason(c *gin.Context) {
	season, err := h.warSvc.GetSeasonInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取报名宗门数量
	registeredCount := len(season.RegisteredSects)

	// 获取赛季奖励配置
	rewards := h.warSvc.GetSeasonRewards()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"season":           season,
			"registered_count": registeredCount,
			"rewards":          rewards,
		},
	})
}

// RegisterWar 报名宗门战争
// @Router POST /api/v1/sect/war/register [post]
func (h *SectExtraHandler) RegisterWar(c *gin.Context) {
	var req struct {
		SectID    string   `json:"sect_id"`
		UserID    string   `json:"user_id"`
		MemberIDs []string `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.SectID == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 和 user_id 不能为空"})
		return
	}
	if len(req.MemberIDs) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少选择3名参战成员"})
		return
	}

	if err := h.warSvc.RegisterSect(c.Request.Context(), req.SectID, req.UserID, req.MemberIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "报名成功"})
}

// GetWarBrackets 查看赛程分组
// @Router GET /api/v1/sect/war/brackets [get]
func (h *SectExtraHandler) GetWarBrackets(c *gin.Context) {
	season, err := h.warSvc.GetCurrentSeason(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(season.BracketIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	brackets, err := h.warSvc.GetBracketsBySeason(c.Request.Context(), season.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": brackets})
}

// GetSectMatches 获取宗门比赛记录
// @Router GET /api/v1/sect/war/matches/:sectID [get]
func (h *SectExtraHandler) GetSectMatches(c *gin.Context) {
	sectID := c.Param("sectID")
	if sectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 不能为空"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	if limit < 1 || limit > 50 {
		limit = 20
	}

	matches, err := h.warSvc.GetSectMatches(c.Request.Context(), sectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": matches})
}

// SubmitMatchResult 提交比赛结果
// @Router POST /api/v1/sect/war/result [post]
func (h *SectExtraHandler) SubmitMatchResult(c *gin.Context) {
	var req struct {
		MatchID string `json:"match_id"`
		ScoreA  int    `json:"score_a"`
		ScoreB  int    `json:"score_b"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.MatchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match_id 不能为空"})
		return
	}
	if req.ScoreA < 0 || req.ScoreB < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分数不能为负"})
		return
	}

	if err := h.warSvc.SubmitBattleResult(c.Request.Context(), req.MatchID, req.ScoreA, req.ScoreB); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "结果提交成功"})
}

// GetWarRankings 获取赛季排名
// @Router GET /api/v1/sect/war/rankings [get]
func (h *SectExtraHandler) GetWarRankings(c *gin.Context) {
	rankings, err := h.warSvc.GetSeasonRankings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rankings})
}

// GetSpiritVeins 获取灵脉列表
// @Router GET /api/v1/sect/war/veins [get]
func (h *SectExtraHandler) GetSpiritVeins(c *gin.Context) {
	sectID := c.Query("sect_id")

	var veins []*service.SpiritVein
	var err error

	if sectID != "" {
		veins, err = h.warSvc.GetSectVeins(c.Request.Context(), sectID)
	} else {
		veins, err = h.warSvc.GetSpiritVeins(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": veins})
}

// ContestVein 争夺灵脉
// @Router POST /api/v1/sect/war/contest [post]
func (h *SectExtraHandler) ContestVein(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		VeinID string `json:"vein_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.SectID == "" || req.VeinID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sect_id 和 vein_id 不能为空"})
		return
	}

	if err := h.warSvc.ContestVein(c.Request.Context(), req.SectID, req.VeinID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "争夺指令已发出"})
}

// ============================================================
// 原宗门战接口(保持兼容)
// ============================================================

// GetWarStatus 查看宗门战状态(兼容旧版)
// @Router POST /api/v1/sect/war/status [post]
func (h *SectExtraHandler) GetWarStatus(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	war, err := h.warSvc.GetActiveWarStatus(c.Request.Context(), req.SectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": war})
}

// EnrollWar 报名宗门战(兼容旧版 -> 注册到赛季)
// @Router POST /api/v1/sect/war/enroll [post]
func (h *SectExtraHandler) EnrollWar(c *gin.Context) {
	var req struct {
		SectID    string   `json:"sect_id"`
		UserID    string   `json:"user_id"`
		PlayerIDs []string `json:"player_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	err := h.warSvc.RegisterSect(c.Request.Context(), req.SectID, req.UserID, req.PlayerIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "报名成功"})
}

// ============================================================
// 宗门排名
// ============================================================

// GetSectRank 宗门排名（联赛排行）
// @Router POST /api/v1/sect/rank [post]
func (h *SectExtraHandler) GetSectRank(c *gin.Context) {
	var req struct {
		Page     int64 `json:"page"`
		PageSize int64 `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Page = 1
		req.PageSize = 20
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 50 {
		req.PageSize = 20
	}

		rankings, err := h.warSvc.GetSeasonRankings(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		total := int64(len(rankings))
		start := (req.Page - 1) * req.PageSize
		if start > total {
			start = total
		}
		end := start + req.PageSize
		if end > total {
			end = total
		}
		if start < end {
			rankings = rankings[start:end]
		} else {
			rankings = []*service.SectRanking{}
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  rankings,
			"total": total,
		})
}
