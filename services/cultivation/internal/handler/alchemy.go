// Package handler 炼丹 API HTTP 处理层
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"cultivation-game/services/cultivation/internal/service"
)

// AlchemyHandler 炼丹系统HTTP处理器
type AlchemyHandler struct {
	alchemySvc     *service.AlchemyService
	enhancedSvc    *service.EnhancedAlchemyService
	playerStore    PlayerStore
}

// NewAlchemyHandler 创建炼丹处理器实例
func NewAlchemyHandler(
	alchemySvc *service.AlchemyService,
	enhancedSvc *service.EnhancedAlchemyService,
	store PlayerStore,
) *AlchemyHandler {
	return &AlchemyHandler{
		alchemySvc:  alchemySvc,
		enhancedSvc: enhancedSvc,
		playerStore: store,
	}
}

// RegisterRoutes 注册炼丹相关HTTP路由
func (h *AlchemyHandler) RegisterRoutes(mux *http.ServeMux) {
	// Original routes
	mux.HandleFunc("/api/v1/alchemy/recipes", h.handleRecipes)
	mux.HandleFunc("/api/v1/alchemy/craft", h.handleCraft)
	mux.HandleFunc("/api/v1/alchemy/collect", h.handleCollect)
	mux.HandleFunc("/api/v1/alchemy/ingredients", h.handleIngredients)

	// Formula research
	mux.HandleFunc("/api/v1/alchemy/research", h.handleResearch)
	mux.HandleFunc("/api/v1/alchemy/formulas", h.handlePlayerFormulas)
	mux.HandleFunc("/api/v1/alchemy/formulas/available", h.handleAvailableFormulas)
	mux.HandleFunc("/api/v1/alchemy/research-attempts", h.handleResearchAttempts)

	// Mini-game crafting
	mux.HandleFunc("/api/v1/alchemy/start-craft", h.handleStartCraft)
	mux.HandleFunc("/api/v1/alchemy/minigame/heat", h.handleMiniGameHeat)
	mux.HandleFunc("/api/v1/alchemy/minigame/add-material", h.handleMiniGameAddMaterial)
	mux.HandleFunc("/api/v1/alchemy/complete-craft", h.handleCompleteCraft)
	mux.HandleFunc("/api/v1/alchemy/session", h.handleGetSession)

	// Toxicity
	mux.HandleFunc("/api/v1/alchemy/toxicity", h.handleGetToxicity)
	mux.HandleFunc("/api/v1/alchemy/detox", h.handleDetox)
	mux.HandleFunc("/api/v1/alchemy/toxicity-effect", h.handleToxicityEffect)

	// Furnace
	mux.HandleFunc("/api/v1/alchemy/furnace", h.handleGetFurnace)
	mux.HandleFunc("/api/v1/alchemy/furnace/repair", h.handleRepairFurnace)
	mux.HandleFunc("/api/v1/alchemy/furnace/upgrade", h.handleUpgradeFurnace)
}

// ---- Original Routes ----

// handleRecipes 查看可炼丹药配方（POST）
// 请求：{"player_id": 1}
// 响应：{"recipes": [...], "count": N}
func (h *AlchemyHandler) handleRecipes(w http.ResponseWriter, r *http.Request) {
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

	recipes := h.alchemySvc.GetRecipes(player)

	// Also get enhanced formulas the player has researched
	h.enhancedSvc.InitPlayerFormulas(req.PlayerID)
	playerFormulas := h.enhancedSvc.GetPlayerFormulas(req.PlayerID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recipes":        recipes,
		"count":          len(recipes),
		"player_formulas": playerFormulas,
	})
}

// handleCraft 炼制丹药（POST）
// 请求：{"player_id": 1, "recipe_id": 1}
// 响应：炼制结果(CraftResult)
func (h *AlchemyHandler) handleCraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		RecipeID int    `json:"recipe_id"`
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

	result := h.alchemySvc.Craft(player, req.RecipeID)

	// Save player data
	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleCollect 采集灵药（POST）
// 请求：{"player_id": 1, "ingredient_id": 101}
// 响应：采集结果(CollectResult)
func (h *AlchemyHandler) handleCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID     uint64 `json:"player_id"`
		IngredientID int    `json:"ingredient_id"`
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

	result := h.alchemySvc.Collect(player, req.IngredientID)

	// Save player data
	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleIngredients 查看已收集材料（GET）
// 查询参数：player_id=1
// 响应：{"ingredients": [...], "count": N}
func (h *AlchemyHandler) handleIngredients(w http.ResponseWriter, r *http.Request) {
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

	ingredients := h.alchemySvc.GetPlayerIngredients(player)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ingredients": ingredients,
		"count":       len(ingredients),
	})
}

// ---- Formula Research Routes ----

// handleResearch 研究新丹方（POST）
// 请求：{"player_id": 1, "formula_id": 5, "use_stones": false}
// 响应：ResearchRecord
func (h *AlchemyHandler) handleResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID uint64 `json:"player_id"`
		FormulaID int   `json:"formula_id"`
		UseStones bool  `json:"use_stones"`
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

	h.enhancedSvc.InitPlayerFormulas(req.PlayerID)
	result := h.enhancedSvc.AttemptResearch(req.PlayerID, req.FormulaID, req.UseStones, player.Luck, player.AlchemyLevel)

	writeJSON(w, http.StatusOK, result)
}

// handlePlayerFormulas 获取玩家已研究的丹方（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handlePlayerFormulas(w http.ResponseWriter, r *http.Request) {
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

	h.enhancedSvc.InitPlayerFormulas(playerID)
	formulas := h.enhancedSvc.GetPlayerFormulas(playerID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"formulas": formulas,
		"count":    len(formulas),
	})
}

// handleAvailableFormulas 获取可研究的丹方（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handleAvailableFormulas(w http.ResponseWriter, r *http.Request) {
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

	h.enhancedSvc.InitPlayerFormulas(playerID)
	formulas := h.enhancedSvc.GetAvailableFormulasForResearch(playerID, player.AlchemyLevel, player.RealmID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"formulas": formulas,
		"count":    len(formulas),
	})
}

// handleResearchAttempts 获取研究尝试信息（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handleResearchAttempts(w http.ResponseWriter, r *http.Request) {
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

	attempts := h.enhancedSvc.GetResearchAttempts(playerID)

	writeJSON(w, http.StatusOK, attempts)
}

// ---- Mini-Game Crafting Routes ----

// handleStartCraft 开始炼丹小游戏（POST）
// 请求：{"player_id": 1, "formula_id": 5}
// 响应：AlchemySession
func (h *AlchemyHandler) handleStartCraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID  uint64 `json:"player_id"`
		FormulaID int    `json:"formula_id"`
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

	session, err := h.enhancedSvc.StartCraftSession(req.PlayerID, req.FormulaID, player.AlchemyLevel)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// handleMiniGameHeat 设置火候（POST）
// 请求：{"player_id": 1, "formula_id": 5, "zone": 1}
// 响应：AlchemySession
func (h *AlchemyHandler) handleMiniGameHeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID  uint64 `json:"player_id"`
		FormulaID int    `json:"formula_id"`
		Zone      int    `json:"zone"` // 0=low, 1=medium, 2=high
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	session, err := h.enhancedSvc.SetHeatZone(req.PlayerID, req.FormulaID, req.Zone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// handleMiniGameAddMaterial 添加材料（POST）
// 请求：{"player_id": 1, "formula_id": 5, "material_id": "101"}
// 响应：AlchemySession
func (h *AlchemyHandler) handleMiniGameAddMaterial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID   uint64 `json:"player_id"`
		FormulaID  int    `json:"formula_id"`
		MaterialID string `json:"material_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	session, err := h.enhancedSvc.AddMaterial(req.PlayerID, req.FormulaID, req.MaterialID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// handleCompleteCraft 完成炼丹并结算（POST）
// 请求：{"player_id": 1, "formula_id": 5}
// 响应：CraftResultEnhanced
func (h *AlchemyHandler) handleCompleteCraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID  uint64 `json:"player_id"`
		FormulaID int    `json:"formula_id"`
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

	result := h.enhancedSvc.CompleteCraft(req.PlayerID, req.FormulaID, player.Luck, player.AlchemyLevel)

	// Update player alchemy exp if successful
	if result.Success {
		player.AlchemyExp += result.AlchemyExp
		// Check level up
		for player.AlchemyLevel < len(h.alchemySvc.GetLevelExpRequirements()) &&
			player.AlchemyExp >= h.alchemySvc.GetLevelExpRequirements()[player.AlchemyLevel] {
			player.AlchemyLevel++
		}
		player.Experience += result.ExpGained
	}

	// Save player data
	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGetSession 获取当前炼丹会话（GET）
// 查询参数：player_id=1&formula_id=5
func (h *AlchemyHandler) handleGetSession(w http.ResponseWriter, r *http.Request) {
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

	formulaIDStr := r.URL.Query().Get("formula_id")
	formulaID, err := strconv.Atoi(formulaIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的丹方ID"})
		return
	}

	session := h.enhancedSvc.GetActiveSession(playerID, formulaID)
	if session == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"session": nil,
			"active":  false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session": session,
		"active":  true,
	})
}

// ---- Toxicity Routes ----

// handleGetToxicity 获取丹毒值（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handleGetToxicity(w http.ResponseWriter, r *http.Request) {
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

	toxicity := h.enhancedSvc.GetPlayerToxicity(playerID)

	writeJSON(w, http.StatusOK, toxicity)
}

// handleDetox 使用解毒丹药（POST）
// 请求：{"player_id": 1}
func (h *AlchemyHandler) handleDetox(w http.ResponseWriter, r *http.Request) {
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

	result := h.enhancedSvc.UseDetox(req.PlayerID)

	writeJSON(w, http.StatusOK, result)
}

// handleToxicityEffect 获取丹毒负面效果（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handleToxicityEffect(w http.ResponseWriter, r *http.Request) {
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

	effects := h.enhancedSvc.GetToxicityEffect(playerID)

	writeJSON(w, http.StatusOK, effects)
}

// ---- Furnace Routes ----

// handleGetFurnace 获取丹炉信息（GET）
// 查询参数：player_id=1
func (h *AlchemyHandler) handleGetFurnace(w http.ResponseWriter, r *http.Request) {
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

	furnace := h.enhancedSvc.GetFurnace(playerID)

	writeJSON(w, http.StatusOK, furnace)
}

// handleRepairFurnace 修复丹炉（POST）
// 请求：{"player_id": 1}
func (h *AlchemyHandler) handleRepairFurnace(w http.ResponseWriter, r *http.Request) {
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

	furnace := h.enhancedSvc.RepairFurnace(req.PlayerID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"furnace": furnace,
		"message": fmt.Sprintf("丹炉已修复，耐久度：%d/%d", furnace.Durability, furnace.MaxDurability),
	})
}

// handleUpgradeFurnace 升级丹炉（POST）
// 请求：{"player_id": 1, "has_rare_material": true}
func (h *AlchemyHandler) handleUpgradeFurnace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req struct {
		PlayerID        uint64 `json:"player_id"`
		HasRareMaterial bool   `json:"has_rare_material"`
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

	result := h.enhancedSvc.UpgradeFurnace(req.PlayerID, player.AlchemyLevel, req.HasRareMaterial)

	writeJSON(w, http.StatusOK, result)
}
