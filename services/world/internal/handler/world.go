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

	"cultivation-game/services/world/internal/service"

	"github.com/gin-gonic/gin"
)

// WorldHandler HTTP 请求处理器
type WorldHandler struct {
	exploreSvc           *service.ExploreService
	encounterSvc         *service.EncounterService
	spiritDensitySvc     *service.SpiritDensityService
	playerServiceAddr    string // Player 服务 HTTP 地址
	cultivationSvcAddr   string // Cultivation 服务 HTTP 地址
}

// NewWorldHandler 创建世界服务处理器
func NewWorldHandler(exploreSvc *service.ExploreService, encounterSvc *service.EncounterService, spiritDensitySvc *service.SpiritDensityService) *WorldHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	cultivationAddr := os.Getenv("CULTIVATION_SERVICE_ADDR")
	if cultivationAddr == "" {
		cultivationAddr = "http://127.0.0.1:8080"
	}
	return &WorldHandler{
		exploreSvc:           exploreSvc,
		encounterSvc:         encounterSvc,
		spiritDensitySvc:     spiritDensitySvc,
		playerServiceAddr:    playerAddr,
		cultivationSvcAddr:   cultivationAddr,
	}
}

// RegisterRoutes 注册所有路由
func (h *WorldHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/world/regions", h.handleRegions)
	r.GET("/api/v1/world/region/:id", h.handleRegion)
	r.POST("/api/v1/world/explore", h.handleExplore)
	r.POST("/api/v1/world/move", h.handleMove)
	r.POST("/api/v1/world/encounter/act", h.handleEncounterAct)
	r.POST("/api/v1/world/gather", h.handleGather)
	r.POST("/api/v1/world/npc/interact", h.handleNpcInteract)
	r.GET("/api/v1/world/player/:player_id", h.handlePlayerExploreStatus)
	// V3 灵气浓度接口
	r.GET("/api/v1/world/spirit/:region_id", h.handleSpiritDensity)
}

// ============================================================
// 请求/响应结构体
// ============================================================

// apiResponse 通用API响应
type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// exploreRequest 探索请求
type exploreRequest struct {
	UserID      string `json:"user_id"`
	PlayerLevel int    `json:"player_level,omitempty"`
}

// moveRequest 移动请求
type moveRequest struct {
	UserID         string `json:"user_id"`
	TargetRegionID string `json:"target_region_id"`
}

// encounterActRequest 奇遇选择请求
type encounterActRequest struct {
	UserID       string `json:"user_id"`
	EncounterID  string `json:"encounter_id"`
	ChoiceIndex  int    `json:"choice_index"`
}

// gatherRequest 采集请求
type gatherRequest struct {
	UserID string `json:"user_id"`
	SpotID string `json:"spot_id"`
}

// npcInteractRequest NPC交互请求
type npcInteractRequest struct {
	UserID string `json:"user_id"`
	NPCID  string `json:"npc_id"`
}

// regionDetailResponse 区域详情响应
type regionDetailResponse struct {
	Region         interface{} `json:"region"`
	NPCs           interface{} `json:"npcs"`
	GatheringSpots interface{} `json:"gathering_spots"`
	Connections    interface{} `json:"connections"`
	Encounters     interface{} `json:"encounters"`
}

// ============================================================
// 辅助函数
// ============================================================

// writeJSON 写入 JSON 响应
func writeJSON(c *gin.Context, statusCode int, resp *apiResponse) {
	c.JSON(statusCode, resp)
}

// writeError 写入错误响应
func writeError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, &apiResponse{
		Code:    statusCode,
		Message: message,
	})
}

// ============================================================
// 处理器实现
// ============================================================

// handleRegions 获取可探索区域列表
// GET /api/v1/world/regions
func (h *WorldHandler) handleRegions(c *gin.Context) {
	regions := h.exploreSvc.GetAllRegions()
	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取区域列表成功",
		Data:    regions,
	})
}

// handleRegion 查看区域详情(NPC/资源/出口/奇遇)
// GET /api/v1/world/region/{id}
func (h *WorldHandler) handleRegion(c *gin.Context) {
	regionID := c.Param("id")

	region, ok := h.exploreSvc.GetRegion(regionID)
	if !ok {
		writeError(c, http.StatusNotFound, "区域不存在")
		return
	}

	// 获取关联数据
	npcs := h.exploreSvc.GetRegionNPCs(regionID)
	spots := h.exploreSvc.GetRegionGatheringSpots(regionID)
	connections := h.exploreSvc.GetRegionConnections(regionID)
	encounters := h.encounterSvc.GetEncountersByRegion(regionID)

	resp := &regionDetailResponse{
		Region:         region,
		NPCs:           npcs,
		GatheringSpots: spots,
		Connections:    connections,
		Encounters:     encounters,
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取区域详情成功",
		Data:    resp,
	})
}

// handleExplore 探索当前区域(触发事件/遇怪/采集/无事)
// POST /api/v1/world/explore
// 概率: 奇遇30% | 遇怪25% | 发现资源25% | 无事20%
func (h *WorldHandler) handleExplore(c *gin.Context) {
	var req exploreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" {
		writeError(c, http.StatusBadRequest, "user_id 不能为空")
		return
	}

	// 获取玩家等级(默认为1)
	playerLevel := req.PlayerLevel
	if playerLevel <= 0 {
		playerLevel = 1
	}

	result := h.exploreSvc.Explore(req.UserID, playerLevel, h.encounterSvc)
	if result == nil {
		writeError(c, http.StatusInternalServerError, "探索失败")
		return
	}

	// 探索获得修炼修为（基础修炼5分钟）
	playerID, _ := strconv.ParseInt(req.UserID, 10, 64)
	if playerID > 0 {
		exp := int64(10+int64(playerLevel-1)*5) * 300 // 5分钟
		go h.syncCultivationExp(playerID, exp)
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "探索完成",
		Data:    result,
	})
}

// handleMove 移动到相邻区域
// POST /api/v1/world/move
func (h *WorldHandler) handleMove(c *gin.Context) {
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" {
		writeError(c, http.StatusBadRequest, "user_id 不能为空")
		return
	}
	if req.TargetRegionID == "" {
		writeError(c, http.StatusBadRequest, "target_region_id 不能为空")
		return
	}

	result, err := h.exploreSvc.MoveTo(req.UserID, req.TargetRegionID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: result.Message,
		Data:    result,
	})
}

// handleEncounterAct 奇遇事件选择分支
// POST /api/v1/world/encounter/act
func (h *WorldHandler) handleEncounterAct(c *gin.Context) {
	var req encounterActRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.EncounterID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 encounter_id 不能为空")
		return
	}

	description, err := h.encounterSvc.ExecuteChoice(req.UserID, req.EncounterID, req.ChoiceIndex)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 奇遇执行完毕后，异步发放奖励到 Player 服务
	playerID, _ := strconv.ParseInt(req.UserID, 10, 64)
	if playerID > 0 {
		go h.sendEncounterRewards(playerID, req.EncounterID, req.ChoiceIndex)
		// 奇遇获得修炼修为（基础修炼15分钟）
		playerLevel := 1
		exp := int64(10+int64(playerLevel-1)*5) * 900 // 15分钟
		go h.syncCultivationExp(playerID, exp)
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "选择已执行",
		Data: map[string]interface{}{
			"description": description,
		},
	})
}

// handleGather 采集资源
// POST /api/v1/world/gather
func (h *WorldHandler) handleGather(c *gin.Context) {
	var req gatherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.SpotID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 spot_id 不能为空")
		return
	}

	drops, msg, err := h.exploreSvc.Gather(req.UserID, req.SpotID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 采集成功后，异步将资源发放到 Player 服务背包
	if len(drops) > 0 {
		playerID, _ := strconv.ParseInt(req.UserID, 10, 64)
		if playerID > 0 {
			go h.sendGatherRewards(playerID, drops)
			// 采集获得修炼修为（基础修炼3分钟）
			playerLevel := 1
			exp := int64(10+int64(playerLevel-1)*5) * 180 // 3分钟
			go h.syncCultivationExp(playerID, exp)
		}
	}

	// 获取剩余行动力
	ap, maxAP := h.exploreSvc.GetPlayerActionPoints(req.UserID)

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: msg,
		Data: map[string]interface{}{
			"resources":        drops,
			"action_points":    ap,
			"max_action_points": maxAP,
		},
	})
}

// handleNpcInteract 与NPC交互(对话/查看商店/查看任务)
// POST /api/v1/world/npc/interact
func (h *WorldHandler) handleNpcInteract(c *gin.Context) {
	var req npcInteractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.UserID == "" || req.NPCID == "" {
		writeError(c, http.StatusBadRequest, "user_id 和 npc_id 不能为空")
		return
	}

	// 查找NPC
	npc, ok := h.exploreSvc.GetNPC(req.NPCID)
	if !ok {
		writeError(c, http.StatusNotFound, "NPC不存在")
		return
	}

	// 检查NPC是否在当前玩家所在区域
	state := h.exploreSvc.GetPlayerExploreInfo(req.UserID)
	if npc.RegionID != state.RegionID {
		writeError(c, http.StatusBadRequest, "该NPC不在你当前所在的区域")
		return
	}

	// 构建交互响应
	resp := map[string]interface{}{
		"npc":         npc,
		"dialogues":   npc.Dialogues,
		"interaction_type": string(npc.Type),
	}

	// 根据不同NPC类型返回不同交互选项
	switch npc.Type {
	case "shop":
		resp["shop_items"] = npc.ShopItems
	case "quest_giver":
		resp["quests"] = npc.Quests
	case "trainer":
		resp["trainable"] = true
	case "cultivator":
		resp["special_dialogue"] = true
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "与 " + npc.Name + " 交谈中",
		Data:    resp,
	})
}

// sendEncounterRewards 奇遇完成后异步发放奖励到 Player 服务
func (h *WorldHandler) sendEncounterRewards(playerID int64, encounterID string, choiceIndex int) {
	// 查找奇遇配置及选择对应的奖励（简化：发放通用经验奖励）
	expAmount := int64(50 + choiceIndex*20) // 根据选择不同发放不同经验
	goldAmount := int64(10 + choiceIndex*5)

	// 增加经验
	expBody, _ := json.Marshal(map[string]interface{}{"exp": expAmount})
	h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/add-exp", h.playerServiceAddr, playerID), expBody)

	// 增加货币
	goldBody, _ := json.Marshal(map[string]interface{}{"gold": goldAmount, "bound_gold": 0, "jade": 0})
	h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID), goldBody)
}

// sendGatherRewards 采集成功后异步将资源发放到 Player 服务背包
func (h *WorldHandler) sendGatherRewards(playerID int64, drops []service.ResourceDrop) {
	for _, drop := range drops {
		itemID, err := strconv.ParseInt(drop.ItemID, 10, 64)
		if err != nil {
			continue
		}
		body, _ := json.Marshal(map[string]interface{}{
			"item_id":  itemID,
			"quantity": int32(drop.Amount),
		})
		h.postToPlayer(fmt.Sprintf("%s/api/v1/player/%d/inventory/add", h.playerServiceAddr, playerID), body)
	}
}

// syncCultivationExp 同步修为到 Cultivation 服务
func (h *WorldHandler) syncCultivationExp(playerID int64, exp int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"player_id": playerID,
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
		log.Printf("[同步] 同步修为到Cultivation服务失败: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// postToPlayer HTTP POST 工具方法
func (h *WorldHandler) postToPlayer(url string, body []byte) {
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

// handlePlayerExploreStatus 获取玩家探索状态
// GET /api/v1/world/player/{player_id}
func (h *WorldHandler) handlePlayerExploreStatus(c *gin.Context) {
	playerID := c.Param("player_id")
	if playerID == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	state := h.exploreSvc.GetPlayerExploreInfo(playerID)
	ap, maxAP := h.exploreSvc.GetPlayerActionPoints(playerID)

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取探索状态成功",
		Data: map[string]interface{}{
			"user_id":            state.UserID,
			"region_id":          state.RegionID,
			"discovered_regions": state.DiscoveredRegions,
			"action_points":      ap,
			"max_action_points":  maxAP,
			"last_move_at":       state.LastMoveAt,
		},
	})
}

// handleSpiritDensity 获取区域灵气浓度
// GET /api/v1/world/spirit/{region_id}
func (h *WorldHandler) handleSpiritDensity(c *gin.Context) {
	regionID := c.Param("region_id")

	density, bonus, ok := h.spiritDensitySvc.GetSpiritDensity(regionID)
	if !ok {
		writeError(c, http.StatusNotFound, "区域不存在")
		return
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取灵气浓度成功",
		Data: map[string]interface{}{
			"region_id":       regionID,
			"spirit_density": density,
			"bonus":          bonus,
			"base_density":   density - bonus,
		},
	})
}

// handleHealth 健康检查
// GET /api/v1/health
func (h *WorldHandler) handleHealth(c *gin.Context) {
	storageStatus := "ok"
	if err := h.exploreSvc.Ping(); err != nil {
		storageStatus = "degraded: " + err.Error()
	}

	writeJSON(c,http.StatusOK, &apiResponse{
		Code:    0,
		Message: "ok",
		Data: map[string]interface{}{
			"service":       "world-service",
			"status":        "running",
			"storage":       storageStatus,
		},
	})
}
