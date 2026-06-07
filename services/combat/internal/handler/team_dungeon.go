// Package handler 组队副本 HTTP 处理器
//
// 路由:
//   POST /api/v1/team-dungeon/create   - 创建队伍
//   POST /api/v1/team-dungeon/join     - 加入队伍
//   POST /api/v1/team-dungeon/leave    - 离开队伍
//   POST /api/v1/team-dungeon/invite   - 邀请玩家
//   POST /api/v1/team-dungeon/ready    - 设置就绪
//   POST /api/v1/team-dungeon/start    - 开始副本
//   GET  /api/v1/team-dungeon/status/:teamId - 队伍状态
//   GET  /api/v1/team-dungeon/list     - 招募中的队伍列表
//   GET  /api/v1/team-dungeon/configs  - 副本配置列表
//   POST /api/v1/team-dungeon/wave     - 提交波次行动
//   POST /api/v1/team-dungeon/complete - 完成副本结算
//   POST /api/v1/team-dungeon/claim    - 领取奖励
package handler

import (
	"net/http"

	"cultivation-game/services/combat/internal/service"

	"github.com/gin-gonic/gin"
)

// TeamDungeonHandler 组队副本 HTTP 处理器
type TeamDungeonHandler struct {
	svc *service.TeamDungeonService
}

// NewTeamDungeonHandler 创建 TeamDungeonHandler
func NewTeamDungeonHandler(svc *service.TeamDungeonService) *TeamDungeonHandler {
	return &TeamDungeonHandler{svc: svc}
}

// RegisterRoutes 注册组队副本路由
func (h *TeamDungeonHandler) RegisterRoutes(r *gin.Engine) {
	tdGroup := r.Group("/api/v1/team-dungeon")
	{
		tdGroup.GET("/configs", h.HandleGetConfigs)
		tdGroup.GET("/list", h.HandleList)
		tdGroup.GET("/status/:teamId", h.HandleStatus)
		tdGroup.POST("/create", h.HandleCreate)
		tdGroup.POST("/join", h.HandleJoin)
		tdGroup.POST("/leave", h.HandleLeave)
		tdGroup.POST("/invite", h.HandleInvite)
		tdGroup.POST("/accept-invite", h.HandleAcceptInvite)
		tdGroup.POST("/decline-invite", h.HandleDeclineInvite)
		tdGroup.POST("/ready", h.HandleReady)
		tdGroup.POST("/start", h.HandleStart)
		tdGroup.POST("/wave", h.HandleWave)
		tdGroup.POST("/complete", h.HandleComplete)
		tdGroup.POST("/claim", h.HandleClaim)
	}
}

// HandleGetConfigs 获取副本配置列表
// GET /api/v1/team-dungeon/configs
func (h *TeamDungeonHandler) HandleGetConfigs(c *gin.Context) {
	configs := h.svc.GetConfigs()
	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
	})
}

// HandleList 获取招募中的队伍列表
// GET /api/v1/team-dungeon/list?dungeon_id=1
func (h *TeamDungeonHandler) HandleList(c *gin.Context) {
	dungeonID := 0
	if idStr := c.Query("dungeon_id"); idStr != "" {
		if id, err := parseInt64(idStr); err == nil {
			dungeonID = int(id)
		}
	}

	teams := h.svc.GetActiveTeams(dungeonID)
	c.JSON(http.StatusOK, gin.H{
		"teams": teams,
	})
}

// HandleStatus 获取队伍状态
// GET /api/v1/team-dungeon/status/:teamId
func (h *TeamDungeonHandler) HandleStatus(c *gin.Context) {
	teamID := c.Param("teamId")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 teamId"})
		return
	}

	status := h.svc.GetTeamInfo(teamID)
	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "队伍不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"team_status": status,
	})
}

// HandleCreate 创建队伍
// POST /api/v1/team-dungeon/create
// Body: { player_id, player_name, dungeon_config_id, position }
func (h *TeamDungeonHandler) HandleCreate(c *gin.Context) {
	var req service.TeamDungeonCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.PlayerID == "" || req.DungeonConfigID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: player_id, dungeon_config_id"})
		return
	}

	team, err := h.svc.CreateTeam(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "队伍已创建",
		"team":    team,
	})
}

// HandleJoin 加入队伍
// POST /api/v1/team-dungeon/join
// Body: { team_id, player_id, player_name, position }
func (h *TeamDungeonHandler) HandleJoin(c *gin.Context) {
	var req service.TeamDungeonJoinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, player_id"})
		return
	}

	team, err := h.svc.JoinTeam(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已加入队伍",
		"team":    team,
	})
}

// HandleLeave 离开队伍
// POST /api/v1/team-dungeon/leave
// Body: { team_id, player_id }
func (h *TeamDungeonHandler) HandleLeave(c *gin.Context) {
	var req service.TeamDungeonLeaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, player_id"})
		return
	}

	if err := h.svc.LeaveTeam(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已离开队伍",
	})
}

// HandleInvite 邀请玩家
// POST /api/v1/team-dungeon/invite
// Body: { team_id, inviter_id, invitee_id }
func (h *TeamDungeonHandler) HandleInvite(c *gin.Context) {
	var req service.TeamDungeonInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || req.InviterID == "" || req.InviteeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, inviter_id, invitee_id"})
		return
	}

	invite, err := h.svc.InviteToTeam(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "邀请已发送",
		"invite":  invite,
	})
}

// HandleAcceptInvite 接受邀请
// POST /api/v1/team-dungeon/accept-invite
// Body: { invite_id, player_id }
func (h *TeamDungeonHandler) HandleAcceptInvite(c *gin.Context) {
	var req struct {
		InviteID string `json:"invite_id"`
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.InviteID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: invite_id, player_id"})
		return
	}

	team, err := h.svc.AcceptInvite(req.InviteID, req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已接受邀请",
		"team":    team,
	})
}

// HandleDeclineInvite 拒绝邀请
// POST /api/v1/team-dungeon/decline-invite
// Body: { invite_id, player_id }
func (h *TeamDungeonHandler) HandleDeclineInvite(c *gin.Context) {
	var req struct {
		InviteID string `json:"invite_id"`
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.InviteID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: invite_id, player_id"})
		return
	}

	if err := h.svc.DeclineInvite(req.InviteID, req.PlayerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已拒绝邀请",
	})
}

// HandleReady 设置就绪状态
// POST /api/v1/team-dungeon/ready
// Body: { team_id, player_id, ready }
func (h *TeamDungeonHandler) HandleReady(c *gin.Context) {
	var req service.TeamDungeonReadyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, player_id"})
		return
	}

	team, err := h.svc.SetReady(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "就绪状态已更新",
		"team":    team,
	})
}

// HandleStart 开始副本
// POST /api/v1/team-dungeon/start
// Body: { team_id, player_id }
func (h *TeamDungeonHandler) HandleStart(c *gin.Context) {
	var req service.TeamDungeonStartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, player_id"})
		return
	}

	team, err := h.svc.StartDungeon(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "副本开始!",
		"team_id":      team.ID,
		"current_wave": team.CurrentWave,
		"time_limit_sec": team.TimeLimitSec,
	})
}

// HandleWave 提交波次行动
// POST /api/v1/team-dungeon/wave
// Body: { team_id, actions: [{ player_id, element, target_id, is_heal, is_support }] }
func (h *TeamDungeonHandler) HandleWave(c *gin.Context) {
	var req struct {
		TeamID  string                         `json:"team_id"`
		Actions []*service.TeamDungeonAttackReq `json:"actions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.TeamID == "" || len(req.Actions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数: team_id, actions"})
		return
	}

	result, err := h.svc.ProcessWave(req.TeamID, req.Actions)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "波次行动完成",
		"result":  result,
	})
}

// HandleComplete 完成副本结算
// POST /api/v1/team-dungeon/complete
// Body: { team_id }
func (h *TeamDungeonHandler) HandleComplete(c *gin.Context) {
	var req struct {
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TeamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: team_id"})
		return
	}

	completion, err := h.svc.CompleteDungeon(req.TeamID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := "副本已结算"
	if completion.Completed && completion.BossDefeated {
		msg = "副本通关!"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    msg,
		"completion": completion,
	})
}

// HandleClaim 领取奖励
// POST /api/v1/team-dungeon/claim
// Body: { team_id, player_id }
func (h *TeamDungeonHandler) HandleClaim(c *gin.Context) {
	var req struct {
		TeamID   string `json:"team_id"`
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TeamID == "" || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数: team_id, player_id"})
		return
	}

	reward, err := h.svc.ClaimMemberRewards(req.TeamID, req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "奖励已领取",
		"reward":  reward,
	})
}

// parseInt64 辅助: 字符串转int64
func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
