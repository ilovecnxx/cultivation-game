// Package handler 提供世界服务的HTTP处理器
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cultivation-game/services/world/internal/model"
	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// QuestHandler 任务系统 HTTP 请求处理器
type QuestHandler struct {
	questSvc         *service.QuestService
	playerServiceAddr string
}

// NewQuestHandler 创建任务系统处理器
func NewQuestHandler(questSvc *service.QuestService) *QuestHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &QuestHandler{
		questSvc:          questSvc,
		playerServiceAddr: playerAddr,
	}
}

// RegisterRoutes 注册任务系统路由
func (h *QuestHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/quest/list", h.handleList)
	r.GET("/api/v1/quest/my", h.handleMyQuests)
	r.POST("/api/v1/quest/accept", h.handleAccept)
	r.POST("/api/v1/quest/submit", h.handleSubmit)
	r.GET("/api/v1/quest/daily", h.handleDaily)
	r.POST("/api/v1/quest/progress", h.handleProgress)
	// 每日任务相关路由
	r.GET("/api/v1/daily-tasks", h.handleDailyTasks)
	r.GET("/api/v1/daily-tasks/progress", h.handleDailyTaskProgress)
	r.POST("/api/v1/daily-tasks/:id/claim", h.handleClaimDailyTask)
	r.GET("/api/v1/daily-tasks/activity-chests", h.handleActivityChests)
	r.POST("/api/v1/daily-tasks/activity-chests/claim", h.handleClaimActivityChest)
}

// ============================================================
// 请求/响应结构体
// ============================================================

// acceptQuestRequest 接取任务请求
type acceptQuestRequest struct {
	PlayerID    string `json:"player_id"`
	QuestID     string `json:"quest_id"`
	PlayerLevel int    `json:"player_level"` // 当前等级(用于检查境界要求)
}

// submitQuestRequest 提交任务请求
type submitQuestRequest struct {
	PlayerID string `json:"player_id"`
	QuestID  string `json:"quest_id"`
}

// listQuestsRequest 任务列表查询参数
type listQuestsRequest struct {
	PlayerID    string `json:"player_id"`
	PlayerLevel int    `json:"player_level"`
}

// ============================================================
// 处理器实现
// ============================================================

// handleList 获取玩家可接任务列表
// GET /api/v1/quest/list?player_id=xxx&player_level=xx
func (h *QuestHandler) handleList(c *gin.Context) {
	playerID := c.Query("player_id")
	levelStr := c.Query("player_level")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	playerLevel := 1
	if levelStr != "" {
		if lvl, err := strconv.Atoi(levelStr); err == nil && lvl > 0 {
			playerLevel = lvl
		}
	}

	quests := h.questSvc.GetAvailableQuests(playerID, playerLevel)
	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取可接任务列表成功",
		Data:    quests,
	})
}

// handleMyQuests 获取玩家已接任务列表
// GET /api/v1/quest/my?player_id=xxx
func (h *QuestHandler) handleMyQuests(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	quests := h.questSvc.GetPlayerQuests(playerID)

	// 为每个任务附加完整配置信息
	type questWithDetail struct {
		Progress *model.PlayerQuest `json:"progress"`
		Config   *model.Quest       `json:"config"`
	}
	var result []questWithDetail
	for _, pq := range quests {
		if cfg, ok := h.questSvc.GetQuest(pq.QuestID); ok {
			result = append(result, questWithDetail{
				Progress: pq,
				Config:   cfg,
			})
		}
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取我的任务列表成功",
		Data:    result,
	})
}

// handleAccept 接取任务
// POST /api/v1/quest/accept
func (h *QuestHandler) handleAccept(c *gin.Context) {
	var req acceptQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" || req.QuestID == "" {
		writeError(c, http.StatusBadRequest, "player_id 和 quest_id 不能为空")
		return
	}

	if err := h.questSvc.AcceptQuest(req.PlayerID, req.QuestID, req.PlayerLevel); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 获取任务配置用于响应
	quest, _ := h.questSvc.GetQuest(req.QuestID)

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "接取任务成功",
		Data: map[string]interface{}{
			"quest":        quest,
			"dialogue":     quest.DialogueStart,
		},
	})
}

// handleSubmit 提交已完成的任务
// POST /api/v1/quest/submit
func (h *QuestHandler) handleSubmit(c *gin.Context) {
	var req submitQuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" || req.QuestID == "" {
		writeError(c, http.StatusBadRequest, "player_id 和 quest_id 不能为空")
		return
	}

	rewards, err := h.questSvc.CompleteQuest(req.PlayerID, req.QuestID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 异步发放奖励到 Player 服务
	playerID, _ := strconv.ParseInt(req.PlayerID, 10, 64)
	if playerID > 0 && len(rewards) > 0 {
		go h.sendQuestRewards(playerID, rewards)
	}

	quest, _ := h.questSvc.GetQuest(req.QuestID)

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "提交任务成功",
		Data: map[string]interface{}{
			"rewards":    rewards,
			"dialogue":   quest.DialogueEnd,
		},
	})
}

// handleDaily 获取今日每日任务
// GET /api/v1/quest/daily
func (h *QuestHandler) handleDaily(c *gin.Context) {
	dailyQuests := h.questSvc.GetDailyQuests()

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取每日任务成功",
		Data:    dailyQuests,
	})
}

// ============================================================
// 每日任务处理器
// ============================================================

// handleDailyTasks 获取玩家所有每日任务及进度
// GET /api/v1/daily-tasks?player_id=xxx
func (h *QuestHandler) handleDailyTasks(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	tasks := h.questSvc.GetDailyTasksWithProgress(playerID)
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取每日任务成功",
		Data:    tasks,
	})
}

// handleDailyTaskProgress 获取指定每日任务的进度
// GET /api/v1/daily-tasks/progress?player_id=xxx&task_id=xxx
func (h *QuestHandler) handleDailyTaskProgress(c *gin.Context) {
	playerID := c.Query("player_id")
	taskID := c.Query("task_id")
	if playerID == "" || taskID == "" {
		writeError(c, http.StatusBadRequest, "player_id 和 task_id 不能为空")
		return
	}

	task, err := h.questSvc.GetDailyTaskProgress(playerID, taskID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取任务进度成功",
		Data:    task,
	})
}

// handleClaimDailyTask 领取每日任务奖励
// POST /api/v1/daily-tasks/:id/claim
func (h *QuestHandler) handleClaimDailyTask(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	rewards, err := h.questSvc.ClaimDailyTaskReward(req.PlayerID, taskID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 异步发放奖励
	playerID, _ := strconv.ParseInt(req.PlayerID, 10, 64)
	if playerID > 0 && len(rewards) > 0 {
		go h.sendQuestRewards(playerID, rewards)
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "领取每日任务奖励成功",
		Data:    rewards,
	})
}

// handleActivityChests 获取活跃度宝箱状态
// GET /api/v1/daily-tasks/activity-chests?player_id=xxx
func (h *QuestHandler) handleActivityChests(c *gin.Context) {
	playerID := c.Query("player_id")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	chests := h.questSvc.GetActivityChests(playerID)
	tiers := h.questSvc.GetChestTiers()

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取活跃度宝箱成功",
		Data: map[string]interface{}{
			"activity_points": chests,
			"chest_tiers":     tiers,
		},
	})
}

// handleClaimActivityChest 领取活跃度宝箱
// POST /api/v1/daily-tasks/activity-chests/claim
func (h *QuestHandler) handleClaimActivityChest(c *gin.Context) {
	var req struct {
		PlayerID string `json:"player_id"`
		Tier     int    `json:"tier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.PlayerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	rewards, err := h.questSvc.ClaimActivityChest(req.PlayerID, model.ChestTier(req.Tier))
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 异步发放奖励
	playerID, _ := strconv.ParseInt(req.PlayerID, 10, 64)
	if playerID > 0 && len(rewards) > 0 {
		go h.sendQuestRewards(playerID, rewards)
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "领取活跃度宝箱成功",
		Data:    rewards,
	})
}

	// handleProgress 接收其他服务的任务进度更新（如战斗击杀怪物）
	// POST /api/v1/quest/progress
	func (h *QuestHandler) handleProgress(c *gin.Context) {
		var req struct {
			PlayerID string `json:"player_id"`
			Type     string `json:"type"`
			TargetID string `json:"target_id"`
			Count    int    `json:"count"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "请求格式错误")
			return
		}
		if req.PlayerID == "" || req.Type == "" {
			writeError(c, http.StatusBadRequest, "player_id 和 type 不能为空")
			return
		}
		if req.Count <= 0 {
			req.Count = 1
		}
		event := model.QuestEvent{Type: req.Type, TargetID: req.TargetID, Count: req.Count}
		h.questSvc.UpdateProgress(req.PlayerID, event)
		writeJSON(c,http.StatusOK, &apiResponse{Code: 0, Message: "任务进度已更新"})
	}
// ============================================================
// ============================================================
// 奖励发放
// ============================================================

// sendQuestRewards 异步发放任务奖励到 Player 服务
func (h *QuestHandler) sendQuestRewards(playerID int64, rewards []model.QuestReward) {
	for _, reward := range rewards {
		switch reward.Type {
		case "exp":
			body, _ := json.Marshal(map[string]interface{}{"exp": reward.Quantity})
			h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/add-exp", h.playerServiceAddr, playerID), body)

		case "money":
			body, _ := json.Marshal(map[string]interface{}{
				"gold":       reward.Quantity,
				"bound_gold": int64(0),
				"jade":       int64(0),
			})
			h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID), body)

		case "item":
			itemID, err := strconv.ParseInt(reward.ID, 10, 64)
			if err != nil {
				log.Printf("任务奖励物品ID解析失败: %s", reward.ID)
				continue
			}
			body, _ := json.Marshal(map[string]interface{}{
				"item_id":  itemID,
				"quantity": int32(reward.Quantity),
			})
			h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/inventory/add", h.playerServiceAddr, playerID), body)

		case "reputation":
			// 声望增加(通过玩家服务或直接存储在社交服务)
			body, _ := json.Marshal(map[string]interface{}{
				"reputation": reward.Quantity,
			})
			h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/reputation", h.playerServiceAddr, playerID), body)

		default:
			log.Printf("未知奖励类型: %s", reward.Type)
		}
	}
}

// postToPlayer HTTP POST 工具方法
func (h *QuestHandler) postToPlayer(url string, body []byte) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}
