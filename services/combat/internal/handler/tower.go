// Package handler 提供心魔塔系统的 HTTP API 处理器。
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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

// httpClient 复用 HTTP 客户端用于跨服务调用
var towerHTTPClient = &http.Client{Timeout: 5 * time.Second}

// TowerHandler 心魔塔 HTTP 处理器。
type TowerHandler struct {
	cfg                *config.Config
	towerSvc           *service.TowerService
	playerServiceAddr  string
	cultivationSvcAddr string

	// 层配置数据
	mu       sync.RWMutex
	floors   map[int]*model.TowerFloorConfig // 从 JSON 加载的层配置
	towerCfg *model.TowerConfig
}

// NewTowerHandler 创建心魔塔处理器。
func NewTowerHandler(cfg *config.Config) *TowerHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	cultivationAddr := os.Getenv("CULTIVATION_SERVICE_ADDR")
	if cultivationAddr == "" {
		cultivationAddr = "http://127.0.0.1:8080"
	}

	return &TowerHandler{
		cfg:                cfg,
		towerSvc:           service.NewTowerService(playerAddr, cultivationAddr),
		playerServiceAddr:  playerAddr,
		cultivationSvcAddr: cultivationAddr,
		floors:             make(map[int]*model.TowerFloorConfig),
		towerCfg: &model.TowerConfig{
			TotalFloors:   100,
			RealmRequired: 3, // 金丹期
			TimeLimitSec:  180,
			DailyFree:     1,
			MaxBuyTimes:   3,
			BuyCost:       100,
			Floors:        make(map[int]*model.TowerFloorConfig),
		},
	}
}

// LoadTowerData 从 JSON 文件加载心魔塔层配置。
func (h *TowerHandler) LoadTowerData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取心魔塔数据文件失败: %w", err)
	}

	var raw struct {
		TotalFloors   int                       `json:"total_floors"`
		RealmRequired uint32                    `json:"realm_required"`
		TimeLimitSec  int                       `json:"time_limit_sec"`
		DailyFree     int                       `json:"daily_free"`
		MaxBuyTimes   int                       `json:"max_buy_times"`
		BuyCost       int64                     `json:"buy_cost"`
		Floors        []*model.TowerFloorConfig `json:"floors"`
		Milestones    []*model.MilestoneReward  `json:"milestones"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析心魔塔数据失败: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.towerCfg.TotalFloors = raw.TotalFloors
	if raw.RealmRequired > 0 {
		h.towerCfg.RealmRequired = raw.RealmRequired
	}
	if raw.TimeLimitSec > 0 {
		h.towerCfg.TimeLimitSec = raw.TimeLimitSec
	}
	if raw.DailyFree > 0 {
		h.towerCfg.DailyFree = raw.DailyFree
	}
	if raw.MaxBuyTimes > 0 {
		h.towerCfg.MaxBuyTimes = raw.MaxBuyTimes
	}
	if raw.BuyCost > 0 {
		h.towerCfg.BuyCost = raw.BuyCost
	}

	for _, f := range raw.Floors {
		h.floors[f.Floor] = f
		h.towerCfg.Floors[f.Floor] = f
	}

	// 处理里程碑
	if raw.Milestones != nil {
		for _, ms := range raw.Milestones {
			h.towerCfg.MilestoneRewards[ms.Floor] = ms
		}
	}

	log.Info().Int("floor_count", len(raw.Floors)).Msg("心魔塔数据加载完成")
	return nil
}

// ---------- API Handlers ----------

// HandleEnter 进入心魔塔。
//
// POST /api/v1/tower/enter
// Body: {"player_id": 123}
func (h *TowerHandler) HandleEnter(c *gin.Context) {
	var req struct {
		PlayerID uint64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	// 检查境界要求（金丹期 realm_id >= 3）
	h.mu.RLock()
	requiredRealm := h.towerCfg.RealmRequired
	h.mu.RUnlock()

	if !h.checkRealmRequirement(c.Request.Context(), req.PlayerID, requiredRealm) {
		c.JSON(http.StatusForbidden, gin.H{"error": "境界不足，需金丹期以上方可进入心魔塔"})
		return
	}

	// 检查每日次数
	canEnter, msg := h.towerSvc.CanEnter(req.PlayerID)
	if !canEnter {
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	useBuy := false
	if msg == "free_used" {
		// 免费用尽，需要使用购买次数
		useBuy = true
		// 检查灵石是否足够
		buyCost := h.towerSvc.GetBuyCost()
		if !h.deductCurrency(c.Request.Context(), req.PlayerID, buyCost) {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "灵石不足，无法购买挑战次数"})
			return
		}
	}

	// 消耗次数
	h.towerSvc.UseDailyAttempt(req.PlayerID, useBuy)

	// 进入心魔塔
	ok, errMsg := h.towerSvc.EnterTower(req.PlayerID)
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": errMsg})
		return
	}

	log.Info().Uint64("player_id", req.PlayerID).Bool("use_buy", useBuy).Msg("进入心魔塔")

	c.JSON(http.StatusOK, gin.H{
		"message":        "已进入心魔塔",
		"current_floor":  1,
		"total_floors":   100,
		"time_limit_sec": 180,
	})
}

// HandleFight 挑战当前层。
//
// POST /api/v1/tower/fight
// Body: {"player_id": 123, "use_item_id": 0, "ignor_choice": 2}
func (h *TowerHandler) HandleFight(c *gin.Context) {
	var req struct {
		PlayerID    uint64 `json:"player_id"`
		UseItemID   uint32 `json:"use_item_id,omitempty"`
		IgnorChoice int    `json:"ignor_choice,omitempty"` // 痴心魔选择的选项索引（-1表示跳过）
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id"})
		return
	}

	// 获取状态以确定当前层
	status := h.towerSvc.GetStatus(req.PlayerID)
	if !status.InSession {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未进入心魔塔，请先进入"})
		return
	}
	if status.Completed || status.Failed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "本次挑战已结束"})
		return
	}
	if status.RemainingTime <= 0 {
		// 超时，标记失败
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "挑战时间已用完"})
		return
	}

	currentFloor := status.CurrentFloor

	// 获取层配置
	h.mu.RLock()
	floorCfg, ok := h.floors[currentFloor]
	h.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("层 %d 配置不存在", currentFloor)})
		return
	}

	// 计算已用时间
	elapsed := 180 - status.RemainingTime
	if elapsed < 0 {
		elapsed = 0
	}

	// 转换配置为 service 层可用的格式
	svcCfg := &service.TowerFloorConfig{
		Floor:           floorCfg.Floor,
		DemonType:       string(floorCfg.DemonType),
		IsBoss:          floorCfg.IsBoss,
		Name:            floorCfg.Name,
		Description:     floorCfg.Description,
		MonsterHP:       floorCfg.MonsterHP,
		MonsterAtk:      floorCfg.MonsterAtk,
		MonsterDef:      floorCfg.MonsterDef,
		MonsterSpeed:    floorCfg.MonsterSpeed,
		GreedCost:       floorCfg.GreedCost,
		WrathMultiplier: floorCfg.WrathMultiplier,
		IgnorQuestion:   floorCfg.IgnorQuestion,
		IgnorAnswer:     floorCfg.IgnorAnswer,
		IgnorChoices:    floorCfg.IgnorChoices,
		RewardExp:       floorCfg.RewardExp,
		RewardMoney:     floorCfg.RewardMoney,
		RewardItems:     floorCfg.RewardItems,
		RewardTitle:     floorCfg.RewardTitle,
	}

	// 执行挑战
	result, errMsg := h.towerSvc.FightFloor(req.PlayerID, svcCfg, currentFloor, elapsed)
	if errMsg != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	log.Info().Uint64("player_id", req.PlayerID).
		Int("floor", currentFloor).
		Str("demon_type", string(floorCfg.DemonType)).
		Bool("win", result.Win).
		Msg("心魔塔战斗")

	c.JSON(http.StatusOK, gin.H{
		"result": result,
	})
}

// HandleStatus 获取心魔塔状态。
//
// GET /api/v1/tower/status?player_id=123
func (h *TowerHandler) HandleStatus(c *gin.Context) {
	playerIDStr := c.Query("player_id")
	if playerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 player_id 参数"})
		return
	}
	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 player_id"})
		return
	}

	status := h.towerSvc.GetStatus(playerID)

	c.JSON(http.StatusOK, status)
}

// HandleRanking 获取排行榜。
//
// GET /api/v1/tower/ranking?limit=100
func (h *TowerHandler) HandleRanking(c *gin.Context) {
	_ = c.Query("limit")

	rankings := h.towerSvc.GetRankings()

	// 包装为响应格式
	type rankEntry struct {
		Rank         int    `json:"rank"`
		PlayerID     uint64 `json:"player_id"`
		Nickname     string `json:"nickname"`
		HighestFloor int    `json:"highest_floor"`
		BestTimeSec  int    `json:"best_time_sec"`
	}

	entries := make([]rankEntry, 0, len(rankings))
	for _, entry := range rankings {
		entries = append(entries, rankEntry{
			Rank:         entry.HighestFloor,
			PlayerID:     entry.PlayerID,
			HighestFloor: entry.HighestFloor,
			BestTimeSec:  entry.BestTimeSec,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"rankings": entries,
		"total":    len(entries),
	})
}

// ---------- 内部辅助方法 ----------

// checkRealmRequirement 检查玩家境界是否满足要求。
// 调用 Player 服务 GetPlayer 接口获取真实境界数据。
func (h *TowerHandler) checkRealmRequirement(ctx context.Context, playerID uint64, requiredRealm uint32) bool {
	url := fmt.Sprintf("%s/api/v1/player/%d", h.playerServiceAddr, playerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Msg("创建查询玩家请求失败，默认允许进入")
		return true
	}

	resp, err := towerHTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Msg("调用 Player 服务失败，默认允许进入")
		return true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Msg("读取 Player 服务响应失败，默认允许进入")
		return true
	}

	// Player 服务 GET /api/v1/player/:id 返回: {"code":0,"msg":"success","data":{...}}
	var result struct {
		Code int `json:"code"`
		Data *struct {
			Realm int32 `json:"realm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Data == nil {
		log.Error().Err(err).Uint64("player_id", playerID).Msg("解析 Player 服务响应失败，默认允许进入")
		return true
	}

	playerRealm := uint32(result.Data.Realm)
	log.Info().Uint64("player_id", playerID).Uint32("realm", playerRealm).Uint32("required", requiredRealm).Msg("玩家境界检查")
	return playerRealm >= requiredRealm
}

// deductCurrency 调用 Player 服务扣除玩家灵石。
// 返回 true 表示扣费成功。
func (h *TowerHandler) deductCurrency(ctx context.Context, playerID uint64, amount int64) bool {
	url := fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID)
	body := map[string]int64{"gold": -amount, "bound_gold": 0, "jade": 0}
	data, err := json.Marshal(body)
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Int64("amount", amount).Msg("序列化货币扣减请求失败")
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Msg("创建货币扣减请求失败")
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := towerHTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Uint64("player_id", playerID).Int64("amount", amount).Msg("调用 Player 服务扣减灵石失败")
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Error().Uint64("player_id", playerID).Int64("amount", amount).Int("status", resp.StatusCode).Str("body", string(respBody)).Msg("Player 服务返回扣减失败")
		return false
	}

	log.Info().Uint64("player_id", playerID).Int64("amount", amount).Msg("灵石扣减成功")
	return true
}

// ---------- 处理注册（与主路由集成）----------

// ---------- 测试辅助 ----------

// GenerateMockRankings 生成模拟排行榜数据（测试用）。
func (h *TowerHandler) GenerateMockRankings(count int) {
	for i := 0; i < count; i++ {
		pid := uint64(10000 + i)
		p := h.towerSvc.GetOrCreatePlayer(pid)
		p.HighestFloor = 10 + rand.Intn(90)
		p.BestTimeSec = 60 + rand.Intn(600)
	}
}
