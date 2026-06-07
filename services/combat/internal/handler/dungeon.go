package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/model"
	"cultivation-game/services/combat/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// DungeonLoader 秘境数据加载器(保持与旧数据格式兼容)
type DungeonLoader struct {
	Dungeons map[int]*model.Dungeon `json:"dungeons"`
}

// DungeonHandler 秘境副本 HTTP 处理器
type DungeonHandler struct {
	cfg                *config.Config
	svc                *service.DungeonService
	playerServiceAddr  string
	cultivationSvcAddr string

	// 旧版兼容: 保留原有数据加载器
	loader *DungeonLoader

	// 运行时状态(旧版兼容)
	mu        sync.Mutex
	sessions  map[string]*model.DungeonSession
	dailies   map[string]*model.DungeonDailyRecord
	enemyTeam map[string][]*model.Fighter
}

// NewDungeonHandler 创建秘境处理器
func NewDungeonHandler(cfg *config.Config) *DungeonHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	cultivationAddr := os.Getenv("CULTIVATION_SERVICE_ADDR")
	if cultivationAddr == "" {
		cultivationAddr = "http://127.0.0.1:8080"
	}

	return &DungeonHandler{
		cfg:                cfg,
		svc:                service.NewDungeonService(),
		playerServiceAddr:  playerAddr,
		cultivationSvcAddr: cultivationAddr,
		loader:             &DungeonLoader{Dungeons: make(map[int]*model.Dungeon)},
		sessions:           make(map[string]*model.DungeonSession),
		dailies:            make(map[string]*model.DungeonDailyRecord),
		enemyTeam:          make(map[string][]*model.Fighter),
	}
}

// ---------- 数据加载(兼容旧版) ----------

// LoadDungeonData 从 JSON 文件加载秘境数据(兼容旧格式)
func (h *DungeonHandler) LoadDungeonData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取秘境数据文件失败: %w", err)
	}

	var raw struct {
		Dungeons []*model.Dungeon `json:"dungeons"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析秘境数据失败: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, d := range raw.Dungeons {
		h.loader.Dungeons[d.ID] = d
	}
	log.Info().Int("count", len(raw.Dungeons)).Msg("秘境数据(旧版)加载完成")
	return nil
}

// ========================================================================
// 新版 API 路由: /api/v1/dungeon/*
// ========================================================================

// ---------- 秘境列表 ----------

// HandleListDungeonsV2 获取秘境列表(新版)
//
// GET /api/v1/dungeon/list
func (h *DungeonHandler) HandleListDungeonsV2(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		// 尝试从请求体获取
		playerID = c.DefaultQuery("player_id", "0")
	}

	list := h.svc.GetDungeonList(playerID)
	c.JSON(http.StatusOK, gin.H{
		"dungeons": list,
	})
}

// ---------- 秘境详情 ----------

// HandleDungeonDetail 获取秘境详情+进度
//
// GET /api/v1/dungeon/:dungeonID
func (h *DungeonHandler) HandleDungeonDetail(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	dungeonID, err := strconv.Atoi(c.Param("dungeonID"))
	if err != nil || dungeonID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 dungeonID"})
		return
	}

	detail := h.svc.GetDungeonDetail(playerID, dungeonID)
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "秘境不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dungeon": detail,
	})
}

// ---------- 进入秘境 ----------

// HandleEnterDungeonV2 进入秘境
//
// POST /api/v1/dungeon/:dungeonID/enter
func (h *DungeonHandler) HandleEnterDungeonV2(c *gin.Context) {
	dungeonID, err := strconv.Atoi(c.Param("dungeonID"))
	if err != nil || dungeonID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 dungeonID"})
		return
	}

	var req struct {
		PlayerID string   `json:"player_id"`
		Team     []string `json:"team,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	// 检查是否持有进行中的会话
	h.mu.Lock()
	if _, exists := h.sessions[req.PlayerID]; exists {
		h.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "已有进行中的秘境(旧版), 请先完成或放弃"})
		return
	}
	h.mu.Unlock()

	// 检查是否可以进入
	canEnter, msg := h.svc.CanEnter(req.PlayerID, dungeonID)
	if !canEnter {
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	// 扣除入场费(调用 Player 服务)
	dConfig := h.svc.GetDungeonConfig(dungeonID)
	if dConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "秘境不存在"})
		return
	}
	if dConfig.EntryCost.SpiritStones > 0 {
		if err := h.deductCurrency(req.PlayerID, dConfig.EntryCost.SpiritStones); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": fmt.Sprintf("灵石不足: %s", err.Error())})
			return
		}
	}

	session, err := h.svc.EnterDungeon(req.PlayerID, dungeonID, req.Team)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("player_id", req.PlayerID).Int("dungeon_id", dungeonID).
		Strs("team", session.Team).Msg("进入秘境(新版)")

	c.JSON(http.StatusOK, gin.H{
		"message":       "已进入秘境",
		"dungeon_name":  dConfig.Name,
		"current_floor": session.CurrentFloor,
		"total_floors":  dConfig.TotalFloors,
		"time_limit_sec": dConfig.TimeLimitSec,
	})
}

// ---------- 战斗 ----------

// HandleFightV2 挑战当前层
//
// POST /api/v1/dungeon/:dungeonID/fight
func (h *DungeonHandler) HandleFightV2(c *gin.Context) {
	dungeonID, err := strconv.Atoi(c.Param("dungeonID"))
	if err != nil || dungeonID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 dungeonID"})
		return
	}

	var req struct {
		PlayerID string              `json:"player_id"`
		Heroes   []PlayerFighterInfo `json:"heroes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	// 验证会话
	session := h.svc.GetSession(req.PlayerID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未进入秘境, 请先进入"})
		return
	}
	if session.DungeonID != dungeonID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "秘境ID不匹配"})
		return
	}
	if session.Completed || session.Failed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "秘境已结束"})
		return
	}

	result, err := h.svc.FightFloor(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dConfig := h.svc.GetDungeonConfig(dungeonID)

	// 通知奖励发放(异步)
	if result.Win && result.Rewards != nil {
		go h.grantFloorRewards(req.PlayerID, result, dConfig)
	}

	// 如果秘境通关, 发放通关奖励
	if result.Completed && result.Win {
		go h.grantCompletionRewards(req.PlayerID, dungeonID, result.Rating, dConfig)
	}

	log.Info().Str("player_id", req.PlayerID).Int("dungeon_id", dungeonID).
		Int("floor", result.Floor).Bool("win", result.Win).
		Bool("completed", result.Completed).Int("rating", result.Rating).
		Msg("秘境战斗(新版)")

	resp := gin.H{
		"result":        map[string]interface{}{
			"win":       result.Win,
			"floor":     result.Floor,
			"is_boss":   result.IsBoss,
			"completed": result.Completed,
		},
		"dungeon_name":  dConfig.Name,
		"current_floor": session.CurrentFloor,
		"total_floors":  dConfig.TotalFloors,
		"logs":          result.Logs,
	}
	if result.Win && result.Rewards != nil {
		resp["floor_reward"] = gin.H{
			"exp":   result.Rewards.Exp,
			"money": result.Rewards.Money,
			"items": result.Rewards.Items,
		}
	}
	if result.Completed {
		resp["rating"] = result.Rating
		resp["message"] = "秘境通关!"
	}

	c.JSON(http.StatusOK, resp)
}

// ---------- 退出秘境 ----------

// HandleExitDungeon 退出秘境
//
// POST /api/v1/dungeon/:dungeonID/exit
func (h *DungeonHandler) HandleExitDungeon(c *gin.Context) {
	dungeonID, err := strconv.Atoi(c.Param("dungeonID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 dungeonID"})
		return
	}

	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	// 验证会话
	session := h.svc.GetSession(req.PlayerID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未进入秘境"})
		return
	}
	if session.DungeonID != dungeonID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "秘境ID不匹配"})
		return
	}

	if err := h.svc.ExitDungeon(req.PlayerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("player_id", req.PlayerID).Int("dungeon_id", dungeonID).Msg("退出秘境")
	c.JSON(http.StatusOK, gin.H{"message": "已退出秘境"})
}

// ---------- 领取奖励 ----------

// HandleClaimDungeonRewards 领取秘境通关奖励
//
// GET /api/v1/dungeon/:dungeonID/rewards
func (h *DungeonHandler) HandleClaimDungeonRewards(c *gin.Context) {
	dungeonID, err := strconv.Atoi(c.Param("dungeonID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 dungeonID"})
		return
	}

	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	session := h.svc.GetSession(playerID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未进入秘境"})
		return
	}
	if session.DungeonID != dungeonID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "秘境ID不匹配"})
		return
	}

	rewards, err := h.svc.ClaimFloorRewards(playerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 实际发放: 通知玩家服务
	go func() {
		h.grantCurrency(playerID, rewards.Money)
		h.grantExp(playerID, rewards.Exp)
		for _, item := range rewards.Items {
			h.grantItem(playerID, item.ID, item.Count)
		}
	}()

	// 清理会话
	h.svc.ClearSession(playerID)

	log.Info().Str("player_id", playerID).Int("dungeon_id", dungeonID).
		Int64("exp", rewards.Exp).Int64("money", rewards.Money).
		Msg("领取秘境奖励")

	c.JSON(http.StatusOK, gin.H{
		"message": "已领取奖励",
		"rewards": rewards,
	})
}

// ---------- 组队邀请 ----------

// HandleTeamInvite 创建组队邀请
//
// POST /api/v1/dungeon/team/invite
func (h *DungeonHandler) HandleTeamInvite(c *gin.Context) {
	var req struct {
		PlayerID  string `json:"player_id"`
		PlayerName string `json:"player_name,omitempty"`
		DungeonID int    `json:"dungeon_id"`
		TargetID  string `json:"target_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}
	if req.DungeonID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 dungeon_id"})
		return
	}

	invite, err := h.svc.CreateInvite(req.PlayerID, req.PlayerName, req.DungeonID, req.TargetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "邀请已发送",
		"invite":  invite,
	})
}

// HandleTeamAccept 接受组队邀请
//
// POST /api/v1/dungeon/team/accept
func (h *DungeonHandler) HandleTeamAccept(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
		InviteID string `json:"invite_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" || req.InviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}

	invite, err := h.svc.AcceptInvite(req.PlayerID, req.InviteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已加入队伍",
		"invite":  invite,
	})
}

// HandleTeamDecline 拒绝组队邀请(额外)
//
// POST /api/v1/dungeon/team/decline
func (h *DungeonHandler) HandleTeamDecline(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
		InviteID string `json:"invite_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.InviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}

	if err := h.svc.DeclineInvite(req.PlayerID, req.InviteID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已拒绝邀请"})
}

// HandleGetPendingInvites 获取待处理邀请
//
// GET /api/v1/dungeon/team/invites?player_id=xxx
func (h *DungeonHandler) HandleGetPendingInvites(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	invites := h.svc.GetPendingInvites(playerID)
	c.JSON(http.StatusOK, gin.H{
		"invites": invites,
	})
}

// ========================================================================
// 旧版 API 兼容路由: /api/v1/combat/dungeon/*
// ========================================================================

// HandleListDungeons 获取秘境列表(旧版)
// GET /api/v1/combat/dungeons
func (h *DungeonHandler) HandleListDungeons(c *gin.Context) {
	playerID := c.DefaultQuery("player_id", "")
	list := h.svc.GetDungeonList(playerID)

	// 旧版格式兼容
	type dungeonBrief struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		RealmReq   int    `json:"realm_req"`
		FloorCount int    `json:"floor_count"`
		DailyLimit int    `json:"daily_limit"`
		EntryFee   int64  `json:"entry_fee"`
	}

	briefs := make([]dungeonBrief, 0, len(list))
	for _, d := range list {
		cfg := h.svc.GetDungeonConfig(d.ID)
		fee := int64(0)
		limit := 3
		if cfg != nil {
			fee = cfg.EntryCost.SpiritStones
			limit = cfg.DailyFree
		}
		briefs = append(briefs, dungeonBrief{
			ID:         d.ID,
			Name:       d.Name,
			RealmReq:   0,
			FloorCount: d.TotalFloors,
			DailyLimit: limit,
			EntryFee:   fee,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"dungeons": briefs,
	})
}

// HandleEnterDungeon 进入秘境(旧版)
// POST /api/v1/combat/dungeon/enter
func (h *DungeonHandler) HandleEnterDungeon(c *gin.Context) {
	var req struct {
		PlayerID  string   `json:"player_id"`
		DungeonID int      `json:"dungeon_id"`
		Team      []string `json:"team,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}
	if req.DungeonID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 dungeon_id"})
		return
	}

	h.mu.Lock()
	if _, exists := h.sessions[req.PlayerID]; exists {
		h.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "已有进行中的秘境, 请先完成或放弃"})
		return
	}
	h.mu.Unlock()

	// 使用新版服务
	canEnter, msg := h.svc.CanEnter(req.PlayerID, req.DungeonID)
	if !canEnter {
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	dConfig := h.svc.GetDungeonConfig(req.DungeonID)
	if dConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "秘境不存在"})
		return
	}
	if dConfig.EntryCost.SpiritStones > 0 {
		if err := h.deductCurrency(req.PlayerID, dConfig.EntryCost.SpiritStones); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": fmt.Sprintf("灵石不足: %s", err.Error())})
			return
		}
	}

	session, err := h.svc.EnterDungeon(req.PlayerID, req.DungeonID, req.Team)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "已进入秘境",
		"dungeon_name":  dConfig.Name,
		"current_floor": session.CurrentFloor,
		"total_floors":  dConfig.TotalFloors,
	})
}

// HandleFight 挑战当前层(旧版)
// POST /api/v1/combat/dungeon/fight
func (h *DungeonHandler) HandleFight(c *gin.Context) {
	var req struct {
		PlayerID string              `json:"player_id"`
		Heroes   []PlayerFighterInfo `json:"heroes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	session := h.svc.GetSession(req.PlayerID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未进入秘境, 请先进入"})
		return
	}

	dConfig := h.svc.GetDungeonConfig(session.DungeonID)
	if dConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "秘境数据异常"})
		return
	}

	result, err := h.svc.FightFloor(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Win && result.Rewards != nil {
		go h.grantFloorRewards(req.PlayerID, result, dConfig)
	}
	if result.Completed && result.Win {
		go h.grantCompletionRewards(req.PlayerID, session.DungeonID, result.Rating, dConfig)
	}

	resp := gin.H{
		"result":        map[string]interface{}{
			"win":       result.Win,
			"floor":     result.Floor,
			"is_boss":   result.IsBoss,
			"completed": result.Completed,
		},
		"dungeon_name":  dConfig.Name,
		"current_floor": session.CurrentFloor,
		"total_floors":  dConfig.TotalFloors,
	}
	if result.Win && result.Rewards != nil {
		resp["floor_reward"] = gin.H{
			"exp":   result.Rewards.Exp,
			"money": result.Rewards.Money,
		}
	}
	resp["battle"] = gin.H{
		"logs": result.Logs,
	}

	c.JSON(http.StatusOK, resp)
}

// HandleClaimReward 领取奖励(旧版)
// POST /api/v1/combat/dungeon/claim
func (h *DungeonHandler) HandleClaimReward(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	rewards, err := h.svc.ClaimFloorRewards(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func() {
		h.grantCurrency(req.PlayerID, rewards.Money)
		h.grantExp(req.PlayerID, rewards.Exp)
		for _, item := range rewards.Items {
			h.grantItem(req.PlayerID, item.ID, item.Count)
		}
	}()

	h.svc.ClearSession(req.PlayerID)

	c.JSON(http.StatusOK, gin.H{
		"message": "已领取通关奖励",
		"exp":     rewards.Exp,
		"money":   rewards.Money,
		"items":   rewards.Items,
	})
}

// HandleStatus 获取秘境进度(旧版)
// GET /api/v1/combat/dungeon/status?player_id=xxx
func (h *DungeonHandler) HandleStatus(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}

	session := h.svc.GetSession(playerID)

	resp := gin.H{
		"in_dungeon": session != nil,
	}

	if session != nil {
		dConfig := h.svc.GetDungeonConfig(session.DungeonID)
		dungeonName := ""
		totalFloors := 0
		if dConfig != nil {
			dungeonName = dConfig.Name
			totalFloors = dConfig.TotalFloors
		}

		resp["session"] = gin.H{
			"dungeon_id":      session.DungeonID,
			"dungeon_name":    dungeonName,
			"current_floor":   session.CurrentFloor,
			"total_floors":    totalFloors,
			"team":            session.Team,
			"completed":       session.Completed,
			"failed":          session.Failed,
			"state":           session.State,
			"total_time_sec":  session.TotalTimeSec,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ========================================================================
// 内部辅助方法
// ========================================================================

// grantFloorRewards 发放层奖励(修为+灵石)
func (h *DungeonHandler) grantFloorRewards(playerID string, result *service.DungeonFightResult, d *service.DungeonConfig) {
	if result.Rewards == nil {
		return
	}

	if result.Rewards.Exp > 0 {
		h.grantExp(playerID, result.Rewards.Exp)
	}
	if result.Rewards.Money > 0 {
		h.grantCurrency(playerID, result.Rewards.Money)
	}
	for _, item := range result.Rewards.Items {
		h.grantItem(playerID, item.ID, item.Count)
	}

	log.Info().Str("player_id", playerID).Int("floor", result.Floor).
		Int64("exp", result.Rewards.Exp).Int64("money", result.Rewards.Money).
		Msg("层奖励已发放")
}

// grantCompletionRewards 发放通关奖励(包含首通奖励)
func (h *DungeonHandler) grantCompletionRewards(playerID string, dungeonID int, rating int, d *service.DungeonConfig) {
	bonusExp := int64(0)
	bonusMoney := int64(0)
	for _, floor := range d.Floors {
		bonusExp += floor.Rewards.Exp
		bonusMoney += floor.Rewards.Money
	}

	bonusExp = int64(float64(bonusExp) * d.CompletionBonus)
	bonusMoney = int64(float64(bonusMoney) * d.CompletionBonus)

	if bonusExp > 0 {
		h.grantExp(playerID, bonusExp)
	}
	if bonusMoney > 0 {
		h.grantCurrency(playerID, bonusMoney)
	}

	log.Info().Str("player_id", playerID).Int("dungeon_id", dungeonID).
		Int("rating", rating).Int64("bonus_exp", bonusExp).Int64("bonus_money", bonusMoney).
		Msg("通关奖励已发放")
}

// grantExp 发放修为
func (h *DungeonHandler) grantExp(playerID string, exp int64) {
	if exp <= 0 {
		return
	}
	h.postJSON(fmt.Sprintf("%s/api/v1/player/%s/add-exp", h.playerServiceAddr, playerID),
		marshalOrSkip(map[string]interface{}{"exp": exp}))
	go h.syncCultivationExp(playerID, exp)
}

// grantCurrency 发放灵石
func (h *DungeonHandler) grantCurrency(playerID string, amount int64) {
	if amount <= 0 {
		return
	}
	h.postJSON(fmt.Sprintf("%s/api/v1/player/%s/currency", h.playerServiceAddr, playerID),
		marshalOrSkip(map[string]interface{}{"gold": amount, "bound_gold": 0, "jade": 0}))
}

// grantItem 发放物品
func (h *DungeonHandler) grantItem(playerID string, itemID, count int) {
	if count <= 0 {
		return
	}
	h.postJSON(fmt.Sprintf("%s/api/v1/player/%s/inventory/add", h.playerServiceAddr, playerID),
		marshalOrSkip(map[string]interface{}{
			"item_id":  int64(itemID),
			"quantity": int32(count),
		}))
}

// deductCurrency 扣除玩家灵石(调用Player服务)
func (h *DungeonHandler) deductCurrency(playerID string, amount int64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"amount": amount,
		"reason": "dungeon_entry_fee",
	})
	url := fmt.Sprintf("%s/api/v1/player/%s/currency/deduct", h.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("调用Player服务失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("扣费失败: %s", string(respBody))
	}
	return nil
}

// syncCultivationExp 同步修为到修炼系统
func (h *DungeonHandler) syncCultivationExp(playerID string, exp int64) {
	pid, err := strconv.ParseInt(playerID, 10, 64)
	if err != nil {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{
		"player_id": pid,
		"exp":       exp,
	})
	url := fmt.Sprintf("%s/api/v1/sync-exp", h.cultivationSvcAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("player_id", playerID).Msg("同步修为到Cultivation服务失败")
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// postJSON HTTP POST 工具方法
func (h *DungeonHandler) postJSON(url string, body []byte) {
	if body == nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("url", url).Msg("HTTP POST 失败")
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// marshalOrSkip JSON 序列化, 失败返回 nil
func marshalOrSkip(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}
