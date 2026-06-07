// Package handler 心魔系统 HTTP API 处理器
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cultivation-game/services/cultivation/internal/model"
	"cultivation-game/services/cultivation/internal/service"
)

// HeartDemonHandler 心魔系统HTTP处理器
type HeartDemonHandler struct {
	logger *slog.Logger
	heartDemonSvc *service.HeartDemonService
	playerStore   PlayerStore
}

// NewHeartDemonHandler 创建心魔处理器
func NewHeartDemonHandler(logger *slog.Logger, hdSvc *service.HeartDemonService, store PlayerStore) *HeartDemonHandler {
	return &HeartDemonHandler{
		logger: logger,
		heartDemonSvc: hdSvc,
		playerStore:   store,
	}
}

// RegisterRoutes 注册心魔系统路由
func (h *HeartDemonHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/heart-demon/illusion/enter", h.handleIllusionEnter)
	mux.HandleFunc("/api/v1/heart-demon/illusion/fight", h.handleIllusionFight)
	mux.HandleFunc("/api/v1/heart-demon/suppress", h.handleSuppress)
	mux.HandleFunc("/api/v1/heart-demon/cleanse", h.handleCleanse)
	mux.HandleFunc("/api/v1/heart-demon/learn-bodhi", h.handleLearnBodhi)
	mux.HandleFunc("/api/v1/heart-demon/", h.handleHeartDemonQuery)
}

// handleHeartDemonQuery 处理心魔查询
// GET /api/v1/heart-demon/{playerID} - 获取心魔列表
// GET /api/v1/heart-demon/{playerID}?mode=value - 获取心魔值
// GET /api/v1/heart-demon/{playerID}?mode=history - 获取历史记录
func (h *HeartDemonHandler) handleHeartDemonQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持GET"})
		return
	}

	// 解析路径: /api/v1/heart-demon/{playerID}
	path := r.URL.Path
	prefix := "/api/v1/heart-demon/"
	if len(path) <= len(prefix) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少玩家ID"})
		return
	}
	idStr := path[len(prefix):]
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少玩家ID"})
		return
	}

	playerID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的玩家ID"})
		return
	}

	player, err := h.playerStore.GetPlayer(playerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	mode := r.URL.Query().Get("mode")

	switch mode {
	case "value":
		info := h.heartDemonSvc.GetHeartDemonValueInfo(player)
		writeJSON(w, http.StatusOK, info)
	case "history":
		history := h.heartDemonSvc.GetDemonHistory(player)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"history": history,
			"count":   len(history),
		})
	case "multipliers":
		multipliers := h.heartDemonSvc.GetMultipliers(player)
		writeJSON(w, http.StatusOK, multipliers)
	default:
		// 默认获取活跃心魔
		active := h.heartDemonSvc.GetActiveDemons(player)
		value := h.heartDemonSvc.CalculateHeartDemonValue(player)
		debuffs := h.heartDemonSvc.GetDebuffs(player)
		multipliers := h.heartDemonSvc.GetMultipliers(player)

		// 补充心魔名称和颜色
		type DemonDisplay struct {
			model.PersistentHeartDemon
			Name  string `json:"name"`
			Icon  string `json:"icon"`
			Color string `json:"color"`
		}

		displayDemons := make([]DemonDisplay, 0, len(active))
		for _, d := range active {
			displayDemons = append(displayDemons, DemonDisplay{
				PersistentHeartDemon: d,
				Name:  h.heartDemonSvc.GetDemonName(d.DemonType),
				Icon:  h.getDemonIcon(d.DemonType),
				Color: h.getDemonColor(d.DemonType),
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"demons":      displayDemons,
			"total_count": len(displayDemons),
			"max_count":   3,
			"value":       value,
			"debuffs":     debuffs,
			"multipliers": multipliers,
			"has_bodhi":   player.HasBodhiTechnique,
		})
	}
}

// handleIllusionEnter 进入心魔幻境
// POST /api/v1/heart-demon/illusion/enter
// Body: {"player_id": 1, "demon_id": "hd_xxx"}
func (h *HeartDemonHandler) handleIllusionEnter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		DemonID  string `json:"demon_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	illusion, errMsg := h.heartDemonSvc.EnterIllusion(player, req.DemonID)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	// 获取痴心魔的问答题目
	if illusion["demon_type"] == string(model.DemonIgnorance) {
		illusion["questions"] = h.heartDemonSvc.GetIllusionQuestions()
	}

	h.logger.Info("玩家进入心魔幻境", "player_id", req.PlayerID, "demon_id", req.DemonID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "已进入心魔幻境",
		"illusion": illusion,
	})
}

// handleIllusionFight 挑战心魔幻境Boss
// POST /api/v1/heart-demon/illusion/fight
// Body: {"player_id": 1, "demon_id": "hd_xxx", "special_choice": "fight/sacrifice/ans1/wait/clone1", "player_atk": 100, "player_def": 50, "player_hp": 1000}
func (h *HeartDemonHandler) handleIllusionFight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID      uint64 `json:"player_id"`
		DemonID       string `json:"demon_id"`
		SpecialChoice string `json:"special_choice"`
		PlayerAtk     int64  `json:"player_atk"`
		PlayerDef     int64  `json:"player_def"`
		PlayerHP      int64  `json:"player_hp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	result, errMsg := h.heartDemonSvc.FightIllusionBoss(
		player, req.DemonID, req.SpecialChoice,
		req.PlayerAtk, req.PlayerDef, req.PlayerHP,
	)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	// 保存玩家状态
	if err := h.playerStore.SavePlayer(player); err != nil {
		h.logger.Error("保存玩家数据失败", "player_id", req.PlayerID, "error", err)
	}

	h.logger.Info("玩家幻境战斗结束", "player_id", req.PlayerID, "win", result["win"])

	writeJSON(w, http.StatusOK, result)
}

// handleSuppress 使用镇魔符压制心魔
// POST /api/v1/heart-demon/suppress
// Body: {"player_id": 1}
func (h *HeartDemonHandler) handleSuppress(w http.ResponseWriter, r *http.Request) {
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

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	msg, errMsg := h.heartDemonSvc.UseSuppressionItem(player)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	if err := h.playerStore.SavePlayer(player); err != nil {
		h.logger.Error("保存玩家数据失败", "player_id", req.PlayerID, "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": msg,
		"success": true,
	})
}

// handleCleanse 使用清心丹净化心魔
// POST /api/v1/heart-demon/cleanse
// Body: {"player_id": 1}
func (h *HeartDemonHandler) handleCleanse(w http.ResponseWriter, r *http.Request) {
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

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	msg, errMsg := h.heartDemonSvc.UseCleansingPill(player)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}

	if err := h.playerStore.SavePlayer(player); err != nil {
		h.logger.Error("保存玩家数据失败", "player_id", req.PlayerID, "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": msg,
		"success": true,
	})
}

// handleLearnBodhi 学习菩提心法
// POST /api/v1/heart-demon/learn-bodhi
// Body: {"player_id": 1}
func (h *HeartDemonHandler) handleLearnBodhi(w http.ResponseWriter, r *http.Request) {
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

	player, err := h.playerStore.GetPlayer(req.PlayerID)
	if err != nil || player == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "玩家不存在"})
		return
	}

	h.heartDemonSvc.LearnBodhiTechnique(player)

	if err := h.playerStore.SavePlayer(player); err != nil {
		h.logger.Error("保存玩家数据失败", "player_id", req.PlayerID, "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "已习得菩提心法，获得心魔防护能力",
		"success": true,
	})
}

// ---------- 辅助方法 ----------

func (h *HeartDemonHandler) getDemonIcon(demonType model.PersistentDemonType) string {
	icons := map[model.PersistentDemonType]string{
		model.DemonGreed:     "💰",
		model.DemonWrath:     "🔥",
		model.DemonIgnorance: "🧠",
		model.DemonDoubt:     "👻",
		model.DemonSloth:     "🐌",
	}
	return icons[demonType]
}

func (h *HeartDemonHandler) getDemonColor(demonType model.PersistentDemonType) string {
	colors := map[model.PersistentDemonType]string{
		model.DemonGreed:     "#ffd700",
		model.DemonWrath:     "#ff5722",
		model.DemonIgnorance: "#6495ed",
		model.DemonDoubt:     "#9c27b0",
		model.DemonSloth:     "#4caf50",
	}
	return colors[demonType]
}
