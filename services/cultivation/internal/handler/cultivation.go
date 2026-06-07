// Package handler HTTP API 处理层，暴露修炼相关接口
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"cultivation-game/services/cultivation/internal/model"
	"cultivation-game/services/cultivation/internal/service"
)

// PlayerStore 玩家数据存储接口（可替换为数据库实现）
type PlayerStore interface {
	GetPlayer(id uint64) (*model.Player, error)
	SavePlayer(player *model.Player) error
	CreatePlayer(name string, spiritRoots map[string]float64) *model.Player
	// Ping 健康检查，返回数据库连接状态
	Ping() error
}

// SimpleEventBus 简易事件总线实现
type SimpleEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]func(data interface{})
}

func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		handlers: make(map[string][]func(data interface{})),
	}
}

func (eb *SimpleEventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	handlers := eb.handlers[event]
	eb.mu.RUnlock()
	for _, h := range handlers {
		go h(data)
	}
}

func (eb *SimpleEventBus) Subscribe(event string, handler func(data interface{})) func() {
	eb.mu.Lock()
	eb.handlers[event] = append(eb.handlers[event], handler)
	eb.mu.Unlock()
	return func() {
		// 取消订阅（简化跳过实现）
	}
}

// CultivationHandler HTTP处理器
type CultivationHandler struct {
	logger *slog.Logger
	realmSvc             *service.RealmService
	techniqueSvc         *service.TechniqueService
	breakthroughSvc      *service.BreakthroughService
	tribulationSvc       *service.TribulationService
	tribulationMgr       *service.TribulationManager
	meditateSvc          *service.MeditateService
	nodeBreakthroughSvc  *service.NodeBreakthroughService
	playerStore          PlayerStore
	playerServiceAddr    string // Player 服务 HTTP 地址
}

// NewCultivationHandler 创建处理器实例
func NewCultivationHandler(logger *slog.Logger,
	realmSvc *service.RealmService,
	techniqueSvc *service.TechniqueService,
	breakthroughSvc *service.BreakthroughService,
	tribulationSvc *service.TribulationService,
	tribulationMgr *service.TribulationManager,
	meditateSvc *service.MeditateService,
	nodeBreakthroughSvc *service.NodeBreakthroughService,
	store PlayerStore,
) *CultivationHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &CultivationHandler{
		logger: logger,
		realmSvc:             realmSvc,
		techniqueSvc:         techniqueSvc,
		breakthroughSvc:      breakthroughSvc,
		tribulationSvc:       tribulationSvc,
		tribulationMgr:       tribulationMgr,
		meditateSvc:          meditateSvc,
		nodeBreakthroughSvc:  nodeBreakthroughSvc,
		playerStore:          store,
		playerServiceAddr:    playerAddr,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *CultivationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/player/create", h.handleCreatePlayer)
	mux.HandleFunc("/api/v1/player/", h.handlePlayerQuery)
	mux.HandleFunc("/api/v1/player/status", h.handlePlayerStatus)
	mux.HandleFunc("/api/v1/player/status/set", h.handlePlayerStatusSet)
	mux.HandleFunc("/api/v1/cultivate", h.handleCultivate)
	mux.HandleFunc("/api/v1/cultivate/status", h.handleCultivateStatus)
	mux.HandleFunc("/api/v1/breakthrough", h.handleBreakthrough)
	mux.HandleFunc("/api/v1/breakthrough/start", h.handleBreakthroughStart)
	mux.HandleFunc("/api/v1/breakthrough/node", h.handleBreakthroughNode)
	mux.HandleFunc("/api/v1/breakthrough/status", h.handleBreakthroughStatus)
	mux.HandleFunc("/api/v1/breakthrough/tribulation", h.handleMajorBreakthroughTribulation)
	mux.HandleFunc("/api/v1/tribulation/info", h.handleTribulationInfo)
	mux.HandleFunc("/api/v1/tribulation/start", h.handleTribulationStart)
	mux.HandleFunc("/api/v1/tribulation/action", h.handleTribulationAction)
	mux.HandleFunc("/api/v1/tribulation/status", h.handleTribulationSessionStatus)
	mux.HandleFunc("/api/v1/tribulation/guardian", h.handleTribulationGuardian)
	mux.HandleFunc("/api/v1/tribulation/complete", h.handleTribulationComplete)
	mux.HandleFunc("/api/v1/tribulation/preview", h.handleTribulationPreview)
	mux.HandleFunc("/api/v1/technique/learn", h.handleLearnTechnique)
	mux.HandleFunc("/api/v1/techniques/available", h.handleAvailableTechniques)
	mux.HandleFunc("/api/v1/sync-exp", h.handleSyncExp)
}

// requireOwnPlayerID 检查认证用户是否操作自己的数据。
// 从请求上下文中获取 auth_player_id，与传入的 playerID 比对。
func (h *CultivationHandler) requireOwnPlayerID(r *http.Request, playerID uint64) bool {
	authID, ok := GetAuthPlayerID(r)
	if !ok {
		return false
	}
	return authID == playerID
}

// ---- HTTP Handlers ----

// handleCreatePlayer 创建新角色
func (h *CultivationHandler) handleCreatePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		Name        string             `json:"name"`
		SpiritRoots map[string]float64 `json:"spirit_roots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "角色名不能为空"})
		return
	}

	if req.SpiritRoots == nil {
		req.SpiritRoots = map[string]float64{"金": 0.3, "木": 0.3, "水": 0.3, "火": 0.3, "土": 0.3}
	}

	player := h.playerStore.CreatePlayer(req.Name, req.SpiritRoots)

	// 计算初始属性
	atk, def, hp := h.realmSvc.CalculateStats(player)
	player.BaseAttack = atk
	player.BaseDefense = def
	player.BaseHP = hp

	writeJSON(w, http.StatusCreated, player)
}

// handlePlayerQuery 查询玩家信息
func (h *CultivationHandler) handlePlayerQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	idStr := r.URL.Path[len("/api/v1/player/"):]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	// 补充境界信息
	_, subStage, ok := h.realmSvc.GetCurrentRealm(player)
	progress := h.realmSvc.GetRealmProgress(player)

	resp := map[string]interface{}{
		"player":          player,
		"realm_name":      "",
		"sub_stage_name":  "",
		"progress":        progress,
	}
	if ok && subStage != nil {
		resp["realm_name"] = subStage.Name
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleCultivate 修炼（在线挂机）
//
// POST /api/v1/cultivate
//
// 请求体:
//   - action: "start" -> 开始挂机修炼
//   - action: "stop"  -> 停止修炼并结算
//   - 默认（不传 action，传 duration_seconds）-> 累积 duration_seconds 秒的修为（每10秒调用一次）
func (h *CultivationHandler) handleCultivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID        uint64 `json:"player_id"`
		Action          string `json:"action,omitempty"`          // "start", "stop"
		Mode            string `json:"mode,omitempty"`            // "online"（在线/默认）, "offline"（离线）
		DurationSeconds int    `json:"duration_seconds,omitempty"` // 修炼时长（秒，默认10），旧版兼容
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	switch req.Action {
	case "start":
		// 开始修炼（在线/离线）
		h.startCultivation(player, req.Mode, w)
		return
	case "stop":
		// 停止修炼并结算
		h.stopCultivation(player, w)
		return
	default:
		// 默认行为：累积指定时长的修为（旧版兼容 / 前端每10秒调用一次）
		h.accumulateCultivation(player, req.DurationSeconds, w)
		return
	}
}

// startCultivation 开始修炼（在线/离线）
// - 在线: 效率100%，每秒tick，前端需定时调用 action:"stop" 结算
// - 离线: 效率20%，退出页面后后端tick继续累积，上限7天
func (h *CultivationHandler) startCultivation(player *model.Player, mode string, w http.ResponseWriter) {
	// 检查玩家状态
	if player.Status != "" && player.Status != "idle" {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error":  "当前状态不允许修炼",
			"status": player.Status,
		})
		return
	}

	if mode == "" {
		mode = "online"
	}

	now := time.Now().Unix()

	if mode == "offline" {
		// ---- 离线修炼（使用 MeditateService） ----
		if player.IsMeditating {
			// 已在离线修炼，先结算再重新开始
			h.meditateSvc.ClaimMeditation(player)
		}
		h.meditateSvc.StartMeditation(player)
		player.Status = "cultivating"
		player.CultivationMode = "offline"

		if err := h.playerStore.SavePlayer(player); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":          "开始离线修炼",
			"mode":             "offline",
			"meditation_start": player.MeditationStart,
		})
		return
	}

	// ---- 在线修炼 ----
	if player.IsCultivating {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":           "已在挂机修炼中",
			"mode":              "online",
			"cultivation_start": player.CultivationStart,
		})
		return
	}

	// 计算效率
	efficiency := h.techniqueSvc.CalculateEfficiency(player)

	player.IsCultivating = true
	player.CultivationStart = now
	player.Status = "cultivating"
	player.CultivationMode = "online"

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":           "开始在线修炼",
		"mode":              "online",
		"cultivation_start": now,
		"efficiency":        efficiency,
	})
}

// stopCultivation 停止修炼并结算
func (h *CultivationHandler) stopCultivation(player *model.Player, w http.ResponseWriter) {
	// 检查是否为离线修炼模式
	if player.CultivationMode == "offline" || player.IsMeditating {
		var gainedExp int64
		if player.IsMeditating {
			gainedExp = h.meditateSvc.ClaimMeditation(player)
		}
		player.Status = "idle"
		player.CultivationMode = ""

		if err := h.playerStore.SavePlayer(player); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "离线修炼结束，修为已结算",
			"gained_exp": gainedExp,
			"total_exp":  player.Experience,
		})
		return
	}

	// 在线修炼结算
	if !player.IsCultivating || player.CultivationStart <= 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "当前没有进行中的修炼",
			"gained_exp": 0,
			"total_exp":  player.Experience,
		})
		return
	}

	now := time.Now().Unix()
	elapsed := now - player.CultivationStart
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > 86400 {
		elapsed = 86400 // 最多24小时
	}

	efficiency := h.techniqueSvc.CalculateEfficiency(player)
	gainedExp := int64(efficiency.ExpPerMinute * float64(elapsed) / 60.0)

	player.Experience += gainedExp
	player.IsCultivating = false
	player.CultivationStart = 0
	player.Status = "idle"
	player.CultivationMode = ""

	// 异步同步修为到 Player 服务
	go h.syncExpToPlayer(player.ID, gainedExp)

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "在线修炼结束，修为已结算",
		"gained_exp":      gainedExp,
		"total_exp":       player.Experience,
		"cultivation_sec": elapsed,
		"efficiency":      efficiency,
	})
}

// accumulateCultivation 累积修为（旧版兼容 / 前端每10秒调用一次）
func (h *CultivationHandler) accumulateCultivation(player *model.Player, durationSeconds int, w http.ResponseWriter) {
	if durationSeconds <= 0 {
		durationSeconds = 10 // 默认每次累积10秒
	}
	if durationSeconds > 86400 {
		durationSeconds = 86400
	}

	// 计算效率
	efficiency := h.techniqueSvc.CalculateEfficiency(player)
	gainedExp := int64(efficiency.ExpPerMinute * float64(durationSeconds) / 60.0)
	player.Experience += gainedExp
	player.Status = "cultivating"

	// 异步同步修为到 Player 服务
	go h.syncExpToPlayer(player.ID, gainedExp)

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "修炼进度已累积",
		"gained_exp":       gainedExp,
		"total_exp":        player.Experience,
		"efficiency":       efficiency,
		"duration_seconds": durationSeconds,
	})
}

// handleCultivateStatus 获取修炼状态
// GET /api/v1/cultivate/status?player_id=5
func (h *CultivationHandler) handleCultivateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(playerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(playerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	efficiency := h.techniqueSvc.CalculateEfficiency(player)
	elapsed := int64(0)
	if player.IsCultivating && player.CultivationStart > 0 {
		elapsed = time.Now().Unix() - player.CultivationStart
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed > 86400 {
			elapsed = 86400
		}
	}
	accumulatedExp := efficiency.ExpPerSecond * elapsed

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"is_cultivating":   player.IsCultivating,
		"cultivation_start": player.CultivationStart,
		"elapsed_sec":      elapsed,
		"accumulated_exp":  accumulatedExp,
		"total_exp":        player.Experience,
		"efficiency":       efficiency,
	})
}

// handleBreakthrough 突破
func (h *CultivationHandler) handleBreakthrough(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64   `json:"player_id"`
		ItemIDs  []string `json:"item_ids,omitempty"` // 使用的辅助物品列表
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	// 使用辅助物品
	for _, itemID := range req.ItemIDs {
		if _, err := h.breakthroughSvc.UseBreakthroughItem(player, itemID); err != nil {
			h.logger.Warn("使用物品失败", "item_id", itemID, "error", err)
		}
	}

	// 判断是否为大境界突破
	gc := h.realmSvc.GetConfig().GetConfig()
	isMajorRealm := false
	if realm, ok := gc.GetRealm(player.RealmID); ok {
		maxLevel := len(realm.SubStages)
		isMajorRealm = player.RealmLevel >= maxLevel
	}

	// 计算各加成因子
	pillBonus := player.GetBreakthroughBonus()
	luckBonus := float64(player.Luck) * 0.001       // 每点气运 +0.1%
	guardianBonus := 0.0                             // 护法系统预留
	karmaPenalty := float64(player.Karma) * 0.001    // 每点业力 -0.1%

	// 计算最终突破率
	rate := h.breakthroughSvc.CalculateBreakthroughRate(
		player, isMajorRealm, pillBonus, luckBonus, guardianBonus, karmaPenalty,
	)

	// 执行突破判定（内部已保存玩家）
	result, err := h.breakthroughSvc.AttemptBreakthrough(player, rate, isMajorRealm)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "突破判定失败: " + err.Error()})
		return
	}

	// 如果突破成功且是大境界突破，触发天劫
	if result.Success && isMajorRealm {
		if nextRealm, ok := gc.GetRealm(player.RealmID); ok && nextRealm.HasTribulation {
			tribulation := h.tribulationSvc.ProcessTribulation(player)
			result.Tribulation = tribulation
			result.KarmaGained = tribulation.KarmaGained

			// 业力增加
			player.Karma += tribulation.KarmaGained

			// 天劫失败则境界倒退
			if tribulation.Triggered && !tribulation.Success {
				player.RealmID = result.NewRealmID - 1
				if prevRealm, ok := gc.GetRealm(player.RealmID); ok {
					player.RealmLevel = len(prevRealm.SubStages)
				}
				result.Success = false
				result.NewRealmID = player.RealmID
				result.NewRealmLevel = player.RealmLevel
				h.logger.Warn("玩家渡劫失败，境界退回", "player_id", player.ID, "realm_id", player.RealmID, "realm_level", player.RealmLevel)
			}
		}
	}

	// 保存玩家（天劫后的状态覆盖）
	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleMajorBreakthroughTribulation 大境界突破（交互式渡劫版）
// POST /api/v1/breakthrough/tribulation
// 请求: {player_id, item_ids?}
// 响应: {rate_check_passed, session?, result}
// 流程：概率判定 -> 通过则自动创建渡劫会话 -> 前端引导渡劫
func (h *CultivationHandler) handleMajorBreakthroughTribulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64   `json:"player_id"`
		ItemIDs  []string `json:"item_ids,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	// 检查是否为大境界突破
	gc := h.realmSvc.GetConfig().GetConfig()
	isMajorRealm := false
	if realm, ok := gc.GetRealm(player.RealmID); ok {
		maxLevel := len(realm.SubStages)
		isMajorRealm = player.RealmLevel >= maxLevel
	}
	if !isMajorRealm {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "当前非大境界突破，无需渡劫"})
		return
	}

	// 检查目标境界是否有天劫
	nextRealm, ok := gc.GetRealm(player.RealmID + 1)
	if !ok || !nextRealm.HasTribulation {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "该境界突破无需渡劫"})
		return
	}

	// 检查是否已有进行中的渡劫
	playerIDStr := fmt.Sprintf("%d", req.PlayerID)
	existingSession := h.tribulationMgr.GetActiveTribulation(playerIDStr)
	if existingSession != nil {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error":   "已有进行中的渡劫",
			"session": existingSession,
		})
		return
	}

	// 使用辅助物品
	for _, itemID := range req.ItemIDs {
		if _, err := h.breakthroughSvc.UseBreakthroughItem(player, itemID); err != nil {
			h.logger.Warn("使用物品失败", "item_id", itemID, "error", err)
		}
	}

	// 计算各加成因子
	pillBonus := player.GetBreakthroughBonus()
	luckBonus := float64(player.Luck) * 0.001
	guardianBonus := 0.0
	karmaPenalty := float64(player.Karma) * 0.001

	// 计算最终突破率
	rate := h.breakthroughSvc.CalculateBreakthroughRate(
		player, true, pillBonus, luckBonus, guardianBonus, karmaPenalty,
	)

	// 使用交互式渡劫突破
	result, err := h.breakthroughSvc.AttemptMajorBreakthroughWithTribulation(player, rate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "突破渡劫失败: " + err.Error()})
		return
	}

	// 如果突破判定未通过（未进入渡劫阶段）
	if !result.Success {
		if err := h.playerStore.SavePlayer(player); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rate_check_passed": false,
			"result":            result,
		})
		return
	}

	// 突破判定通过，渡劫会话已创建
	session := h.tribulationMgr.GetActiveTribulation(playerIDStr)

	h.logger.Info("玩家通过突破概率判定，进入交互式渡劫阶段", "player_id", req.PlayerID, "rate", rate*100)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rate_check_passed": true,
		"session":           session,
		"result":            result,
	})
}

// ---- 突破小游戏（NodeBreakthroughService） ----

// handleBreakthroughStart 开始突破小游戏
// POST /api/v1/breakthrough/start
// 请求: {player_id, pill_time_bonus (秒), pill_range_bonus (百分比)}
// 响应: BreakthroughSession
func (h *CultivationHandler) handleBreakthroughStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}
	var req struct {
		PlayerID       uint64  `json:"player_id"`
		PillTimeBonus  int64   `json:"pill_time_bonus"`
		PillRangeBonus float64 `json:"pill_range_bonus"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作"})
		return
	}
	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}
	session := h.nodeBreakthroughSvc.StartBreakthrough(player, req.PillTimeBonus, req.PillRangeBonus)
	writeJSON(w, http.StatusOK, session)
}

// handleBreakthroughNode 收集灵气节点
// POST /api/v1/breakthrough/node
// 请求: {session_id, node_id}
// 响应: {collected: bool, result?: BreakthroughNodeResult}
func (h *CultivationHandler) handleBreakthroughNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		NodeID    string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	collected, result := h.nodeBreakthroughSvc.CollectNode(req.SessionID, req.NodeID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"collected": collected,
		"result":    result,
	})
}

// handleBreakthroughStatus 查询突破状态
// GET /api/v1/breakthrough/status?session_id=xxx
// 先检查超时, 返回当前session信息
func (h *CultivationHandler) handleBreakthroughStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少session_id"})
		return
	}
	// 检查超时
	timeoutResult := h.nodeBreakthroughSvc.CheckTimeout(sessionID)
	if timeoutResult != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"session": nil,
			"timeout": true,
			"result":  timeoutResult,
		})
		return
	}
	session := h.nodeBreakthroughSvc.GetSession(sessionID)
	if session == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"session": nil,
			"timeout": false,
			"result":  nil,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session": session,
		"timeout": false,
		"result":  nil,
	})
}

// handleTribulationInfo 天劫信息预览
func (h *CultivationHandler) handleTribulationInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(playerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	info := h.tribulationSvc.GetTribulationInfo(player)
	writeJSON(w, http.StatusOK, info)
}

// ---- 交互式渡劫系统 V2 处理器 ----

// handleTribulationStart 开始渡劫
// POST /api/v1/tribulation/start
// 请求: {player_id, player_name}
// 响应: TribulationSession
func (h *CultivationHandler) handleTribulationStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID   uint64 `json:"player_id"`
		PlayerName string `json:"player_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	playerIDStr := fmt.Sprintf("%d", req.PlayerID)
	playerName := req.PlayerName
	if playerName == "" {
		playerName = player.Name
	}

	session, err := h.tribulationMgr.StartTribulation(playerIDStr, playerName, player)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	h.logger.Info("玩家开始渡劫", "player_id", playerIDStr, "tribulation_type", session.TypeName, "total_waves", session.TotalWaves, "hp", session.MaxHP)

	// 推送全服公告（简易实现：打印日志）
	h.logger.Info("玩家开始渡劫，天地异象", "player_name", playerName, "tribulation_type", session.TypeName)

	writeJSON(w, http.StatusOK, session)
}

// handleTribulationAction 处理一波雷劫
// POST /api/v1/tribulation/action
// 请求: {player_id, action: "endure"|"dodge"|"artifact", item_id?: string}
// 响应: WaveResult
func (h *CultivationHandler) handleTribulationAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		Action   string `json:"action"`
		ItemID   string `json:"item_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	// 验证action
	validActions := map[string]bool{"endure": true, "dodge": true, "artifact": true}
	if !validActions[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的行动类型，请使用 endure/dodge/artifact"})
		return
	}

	playerIDStr := fmt.Sprintf("%d", req.PlayerID)

	waveAction := model.WaveAction{
		Action: req.Action,
		ItemID: req.ItemID,
	}

	result, err := h.tribulationMgr.ProcessWave(playerIDStr, waveAction)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleTribulationSessionStatus 获取渡劫状态
// GET /api/v1/tribulation/status?player_id=xxx
// 响应: TribulationSession（进行中或已结束）
func (h *CultivationHandler) handleTribulationSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	playerIDStr := r.URL.Query().Get("player_id")
	if playerIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少player_id"})
		return
	}

	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	if !h.requireOwnPlayerID(r, playerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	session := h.tribulationMgr.GetTribulationStatus(playerIDStr)
	if session == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"has_active": false,
			"session":    nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_active": session.Status == "active",
		"session":    session,
	})
}

// handleTribulationGuardian 加入护法
// POST /api/v1/tribulation/guardian
// 请求: {player_id, guardian_id, guardian_name}
// 响应: {guardian_count, message}
func (h *CultivationHandler) handleTribulationGuardian(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID      uint64 `json:"player_id"`      // 渡劫者ID
		GuardianID    uint64 `json:"guardian_id"`     // 护法者ID
		GuardianName  string `json:"guardian_name"`   // 护法者名称
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.GuardianID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	playerIDStr := fmt.Sprintf("%d", req.PlayerID)
	guardianIDStr := fmt.Sprintf("%d", req.GuardianID)

	count, err := h.tribulationMgr.AddGuardian(playerIDStr, guardianIDStr, req.GuardianName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"guardian_count": count,
		"message":        fmt.Sprintf("已加入护法，当前%d名道友护法", count),
	})
}

// handleTribulationComplete 渡劫完成，应用奖励或失败惩罚
// POST /api/v1/tribulation/complete
// 请求: {player_id}
// 响应: {success, bonus?, realm_id, realm_level}
func (h *CultivationHandler) handleTribulationComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	playerIDStr := fmt.Sprintf("%d", req.PlayerID)

	// 检查渡劫结果
	success, _ := h.tribulationMgr.CheckTribulationResult(playerIDStr)
	if !success {
		// 渡劫失败：境界回退
		player.RealmID--
		if player.RealmID < 1 {
			player.RealmID = 1
		}
		gc := h.realmSvc.GetConfig().GetConfig()
		if prevRealm, ok := gc.GetRealm(player.RealmID); ok {
			player.RealmLevel = len(prevRealm.SubStages)
		}

		// 道心受损
		player.DaoXinStacks++
		if player.DaoXinStacks > 3 {
			player.DaoXinStacks = 3
		}

		// 重新计算原境界属性
		atk, def, hp := h.realmSvc.CalculateStats(player)
		player.BaseAttack = atk
		player.BaseDefense = def
		player.BaseHP = hp

		h.logger.Warn("玩家渡劫失败，境界退回", "player_id", req.PlayerID, "realm_id", player.RealmID, "realm_level", player.RealmLevel)

		if err := h.playerStore.SavePlayer(player); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":     false,
			"message":     "渡劫失败！境界倒退，道心受损！",
			"realm_id":    player.RealmID,
			"realm_level": player.RealmLevel,
			"dao_xin":     player.DaoXinStacks,
		})
		return
	}

	// 渡劫成功：应用奖励
	bonusApplied, err := h.tribulationMgr.ApplyTribulationBonus(playerIDStr, player)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "应用渡劫奖励失败: " + err.Error()})
		return
	}

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	// 通知其他服务
	h.realmSvc.AfterBreakthrough(player.ID, player.RealmID, player.RealmLevel)

	h.logger.Info("玩家成功渡劫", "player_name", player.Name)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"bonus":       bonusApplied,
		"realm_id":    player.RealmID,
		"realm_level": player.RealmLevel,
	})
}

// handleTribulationPreview 渡劫预览信息
// GET /api/v1/tribulation/preview?player_id=xxx
// 响应: {has_tribulation, type_name, total_waves, ...}
func (h *CultivationHandler) handleTribulationPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	playerIDStr := r.URL.Query().Get("player_id")
	if playerIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少player_id"})
		return
	}

	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(playerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	preview := h.tribulationMgr.GetTribulationPreview(player)
	writeJSON(w, http.StatusOK, preview)
}

// handleLearnTechnique 学习功法
func (h *CultivationHandler) handleLearnTechnique(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID    uint64 `json:"player_id"`
		TechniqueID int    `json:"technique_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	result := h.techniqueSvc.LearnTechnique(player, req.TechniqueID)

	// 重新计算属性
	if result.Success {
		atk, def, hp := h.realmSvc.CalculateStats(player)
		player.BaseAttack = atk
		player.BaseDefense = def
		player.BaseHP = hp

		if err := h.playerStore.SavePlayer(player); err != nil {
			h.logger.Error("保存玩家数据失败", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// handleAvailableTechniques 获取可学功法列表
func (h *CultivationHandler) handleAvailableTechniques(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(playerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	techniques := h.techniqueSvc.GetAvailableTechniques(player)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"techniques": techniques,
		"count":      len(techniques),
	})
}

// handlePlayerStatus 获取玩家当前状态
// POST /api/v1/player/status {player_id}
func (h *CultivationHandler) handlePlayerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if !h.requireOwnPlayerID(r, req.PlayerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作该玩家"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            player.Status,
		"cultivation_mode":  player.CultivationMode,
		"is_cultivating":    player.IsCultivating,
		"is_meditating":     player.IsMeditating,
	})
}

// handlePlayerStatusSet 设置玩家状态（其他服务内部调用）
// POST /api/v1/player/status/set {player_id, status}
func (h *CultivationHandler) handlePlayerStatusSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	// 检查合法状态值
	validStatuses := map[string]bool{"idle": true, "cultivating": true, "adventuring": true, "exploring": true}
	if !validStatuses[req.Status] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的状态值"})
		return
	}

	player.Status = req.Status

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "状态已更新",
		"status":  player.Status,
	})
}

// handleSyncExp 其他服务同步修为到玩家
// POST /api/v1/sync-exp {player_id, exp}
//
// 例如战斗胜利后增加历练修为
func (h *CultivationHandler) handleSyncExp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		Exp      int64  `json:"exp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		player = h.autoCreatePlayer(req.PlayerID)
		if player != nil {
			_ = h.playerStore.SavePlayer(player)
		}
	}
	if player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	if req.Exp <= 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "无需增加修为",
			"added_exp": 0,
			"total_exp": player.Experience,
		})
		return
	}

	player.Experience += req.Exp

	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "修为已同步",
		"added_exp": req.Exp,
		"total_exp": player.Experience,
	})
}

// syncExpToPlayer 异步同步修为到 Player 服务
func (h *CultivationHandler) syncExpToPlayer(playerID uint64, exp int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"spirit_power": exp,
	})
	url := fmt.Sprintf("%s/api/v1/player/%d/update-exp", h.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("同步修为到 Player 服务失败", "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// handleHealth 健康检查（含数据库状态）
func (h *CultivationHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.playerStore.Ping(); err != nil {
		dbStatus = "degraded: " + err.Error()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"service":  "cultivation",
		"database": dbStatus,
	})
}

// writeJSON 写出JSON响应
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("JSON编码失败", "error", err)
	}
}

// autoCreatePlayer 当玩家在修炼系统不存在时自动创建。
func (h *CultivationHandler) autoCreatePlayer(playerID uint64) *model.Player {
	p := &model.Player{}
	p.ID = playerID
	p.RealmID = 1
	p.RealmLevel = 1
	p.Experience = 0
	p.AccumulatedExp = 0
	p.Status = "idle"
	return p
}


