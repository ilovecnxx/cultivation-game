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

// DivinationHandler 天机阁推演 HTTP 处理器
type DivinationHandler struct {
	divineSvc         *service.DivinationService
	playerServiceAddr string
}

// NewDivinationHandler 创建 DivinationHandler
func NewDivinationHandler(divineSvc *service.DivinationService) *DivinationHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &DivinationHandler{
		divineSvc:         divineSvc,
		playerServiceAddr: playerAddr,
	}
}

// RegisterRoutes 注册天机阁推演路由
func (h *DivinationHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/world/divine", h.handleDivine)
	r.GET("/api/v1/world/divine/result/:id", h.handleGetResult)
	r.GET("/api/v1/world/divine/level", h.handleGetLevel)
	r.GET("/api/v1/world/divine/options", h.handleGetOptions)
}

// ============================================================
// 请求/响应结构体
// ============================================================

// divineRequest 推演请求
type divineRequest struct {
	PlayerID  int64                `json:"player_id"`
	Type      model.DivinationType `json:"type"`
	ExtraGold int64                `json:"extra_gold"` // 额外投入灵石
	ExtraJade int64                `json:"extra_jade"` // 额外投入仙玉
}

// ============================================================
// 处理器实现
// ============================================================

// handleDivine 执行推演
// POST /api/v1/world/divine
func (h *DivinationHandler) handleDivine(c *gin.Context) {
	var req divineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.PlayerID <= 0 {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	// 验证推演类型
	switch req.Type {
	case model.DivinationBreakthrough, model.DivinationTreasure, model.DivinationWeather:
	default:
		writeError(c, http.StatusBadRequest, "无效的推演类型，可选: breakthrough/treasure/weather")
		return
	}

	if req.ExtraGold < 0 {
		req.ExtraGold = 0
	}
	if req.ExtraJade < 0 {
		req.ExtraJade = 0
	}

	result, err := h.divineSvc.Divine(req.PlayerID, req.Type, req.ExtraGold, req.ExtraJade)
	if err != nil {
		log.Printf("[天机阁] 推演失败: %v", err)
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 异步同步消耗到 Player 服务
	if result.CostGold > 0 || result.CostJade > 0 {
		log.Printf("[天机阁] 玩家 %d 推演消耗 gold=%d jade=%d", req.PlayerID, result.CostGold, result.CostJade)
		go h.syncCurrencyDeduction(req.PlayerID, result.CostGold, result.CostJade)
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "推演完成",
		Data:    result,
	})
}

// handleGetResult 查看推演结果
// GET /api/v1/world/divine/result/{id}?player_id=xxx
func (h *DivinationHandler) handleGetResult(c *gin.Context) {
	resultID := c.Param("id")
	playerIDStr := c.Query("player_id")

	if resultID == "" || playerIDStr == "" {
		writeError(c, http.StatusBadRequest, "缺少参数: id 和 player_id")
		return
	}

	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的 player_id")
		return
	}

	result, err := h.divineSvc.GetResult(playerID, resultID)
	if err != nil {
		log.Printf("[天机阁] 查询推演结果失败: %v", err)
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取推演结果成功",
		Data:    result,
	})
}

// handleGetLevel 获取天机阁等级
// GET /api/v1/world/divine/level?player_id=xxx
func (h *DivinationHandler) handleGetLevel(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	if playerIDStr == "" {
		writeError(c, http.StatusBadRequest, "player_id 不能为空")
		return
	}

	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的 player_id")
		return
	}

	level := h.divineSvc.GetLevel(playerID)
	canFree := h.divineSvc.CanFreeDivine(playerID)

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取天机阁等级成功",
		Data: map[string]interface{}{
			"level":    level,
			"can_free": canFree,
		},
	})
}

// handleGetOptions 获取推演消耗选项
// GET /api/v1/world/divine/options?player_id=xxx&type=xxx
func (h *DivinationHandler) handleGetOptions(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	typeStr := c.Query("type")

	if playerIDStr == "" || typeStr == "" {
		writeError(c, http.StatusBadRequest, "player_id 和 type 不能为空")
		return
	}

	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的 player_id")
		return
	}

	divType := model.DivinationType(typeStr)
	switch divType {
	case model.DivinationBreakthrough, model.DivinationTreasure, model.DivinationWeather:
	default:
		writeError(c, http.StatusBadRequest, "无效的推演类型")
		return
	}

	opts := h.divineSvc.GetConsumeOptions(playerID, divType)
	canFree := h.divineSvc.CanFreeDivine(playerID)

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取推演选项成功",
		Data: map[string]interface{}{
			"options":  opts,
			"can_free": canFree,
			"base_cost": map[string]int64{
				"gold": model.DivineCostGoldBase,
				"jade": model.DivineCostJadeBase,
			},
		},
	})
}

// syncCurrencyDeduction 异步同步货币消耗到 Player 服务
func (h *DivinationHandler) syncCurrencyDeduction(playerID int64, gold, jade int64) {
	url := fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID)
	body := map[string]int64{"gold": -gold, "bound_gold": 0, "jade": -jade}
	data, err := json.Marshal(body)
	if err != nil {
		log.Printf("[天机阁] 序列化货币扣减请求失败: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		log.Printf("[天机阁] 创建货币扣减请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[天机阁] 调用 Player 服务扣减货币失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[天机阁] Player 服务返回扣减失败: status=%d body=%s", resp.StatusCode, string(respBody))
		return
	}

	log.Printf("[天机阁] 玩家 %d 货币扣减成功: gold=%d jade=%d", playerID, gold, jade)
}
