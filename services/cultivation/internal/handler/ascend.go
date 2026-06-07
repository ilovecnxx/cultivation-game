// Package handler 飞升仙界 HTTP 处理器
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
	"time"

	"cultivation-game/services/cultivation/internal/service"
)

// AscendHandler 飞升处理器
type AscendHandler struct {
	logger *slog.Logger
	realmSvc          *service.RealmService
	playerStore       PlayerStore
	playerServiceAddr string
	worldServiceAddr  string
}

// NewAscendHandler 创建飞升处理器
func NewAscendHandler(logger *slog.Logger, realmSvc *service.RealmService, store PlayerStore) *AscendHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	worldAddr := os.Getenv("WORLD_SERVICE_ADDR")
	if worldAddr == "" {
		worldAddr = "http://127.0.0.1:8081"
	}
	return &AscendHandler{
		logger: logger,
		realmSvc:          realmSvc,
		playerStore:       store,
		playerServiceAddr: playerAddr,
		worldServiceAddr:  worldAddr,
	}
}

// RegisterRoutes 注册飞升相关路由
func (h *AscendHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/ascend", h.handleAscend)
	mux.HandleFunc("/api/v1/ascend/status", h.handleAscendStatus)
}

// ----- 请求/响应结构体 -----

type ascendRequest struct {
	PlayerID uint64 `json:"player_id"`
}

type ascendStatusResponse struct {
	CanAscend  bool   `json:"can_ascend"`
	RealmID    int    `json:"realm_id"`
	RealmLevel int    `json:"realm_level"`
	RealmName  string `json:"realm_name"`
	Message    string `json:"message"`
}

type ascendResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	NewRealmID   int    `json:"new_realm_id,omitempty"`
	NewRealmName string `json:"new_realm_name,omitempty"`
}

// ----- Handlers -----

// handleAscend 执行飞升
// POST /api/v1/ascend
func (h *AscendHandler) handleAscend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持POST"})
		return
	}

	var req ascendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if req.PlayerID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少player_id"})
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

	// 检查是否为渡劫圆满（realmID=9, realmLevel=10）
	if player.RealmID != 9 || player.RealmLevel != 10 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "境界不足，需渡劫圆满方可飞升",
		})
		return
	}

	// ---- 执行飞升 ----

	// 1. 更新玩家境界为天仙一层（realmID=10, realmLevel=1）
	player.RealmID = 10
	player.RealmLevel = 1

	// 2. 计算飞升后基础属性（天仙基础属性大幅提升）
	hp := int64(10000 + 10*50 + 1*10)
	attack := int64(1000 + 10*20 + 1*5)
	defense := int64(1000 + 10*15 + 1*3)

	player.BaseAttack = attack
	player.BaseDefense = defense
	player.BaseHP = hp

	// 3. 将修为置为天仙一层的初始值
	player.Experience = 0

	// 4. 保存玩家
	if err := h.playerStore.SavePlayer(player); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存玩家数据失败"})
		return
	}

	// 5. 通知 Player 服务更新属性和境界
	go h.notifyPlayerService(req.PlayerID, 10, 1, attack, defense, hp)

	// 6. 通知 RealmService 处理排行榜更新
	h.realmSvc.AfterBreakthrough(req.PlayerID, 10, 1)

	// 7. 触发全服公告和天降异象（通过世界服务）
	go h.triggerAscendAnnouncement(req.PlayerID, player.Name)

	h.logger.Info("玩家成功飞升仙界", "player_name", player.Name, "player_id", req.PlayerID)

	writeJSON(w, http.StatusOK, &ascendResponse{
		Success:      true,
		Message:      "飞升成功！恭喜踏入仙界，位列仙班！",
		NewRealmID:   10,
		NewRealmName: "天仙一层",
	})
}

// handleAscendStatus 查询飞升状态
// GET /api/v1/ascend/status?player_id=5
func (h *AscendHandler) handleAscendStatus(w http.ResponseWriter, r *http.Request) {
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

	canAscend := player.RealmID == 9 && player.RealmLevel == 10

	// 境界名称
	realmName := ""
	if player.RealmID <= 9 {
		realmNames := map[int]string{
			1: "炼气", 2: "筑基", 3: "金丹", 4: "元婴",
			5: "化神", 6: "炼虚", 7: "合体", 8: "大乘", 9: "渡劫",
		}
		if name, ok := realmNames[player.RealmID]; ok {
			realmName = fmt.Sprintf("%s第%d层", name, player.RealmLevel)
		}
	} else {
		xianjieNames := map[int]string{
			10: "天仙", 11: "金仙", 12: "仙君", 13: "仙帝",
		}
		if name, ok := xianjieNames[player.RealmID]; ok {
			realmName = fmt.Sprintf("%s第%d层", name, player.RealmLevel)
		}
	}

	var msg string
	switch {
	case player.RealmID >= 10:
		msg = "已飞升仙界"
	case canAscend:
		msg = "渡劫圆满，可飞升仙界！"
	default:
		msg = "继续修炼，达渡劫圆满后方可飞升"
	}

	writeJSON(w, http.StatusOK, &ascendStatusResponse{
		CanAscend:  canAscend,
		RealmID:    player.RealmID,
		RealmLevel: player.RealmLevel,
		RealmName:  realmName,
		Message:    msg,
	})
}

// ----- 内部方法 -----

// notifyPlayerService 通知 Player 服务更新玩家境界和属性
func (h *AscendHandler) notifyPlayerService(playerID uint64, realmID, realmLevel int, attack, defense, hp int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"realm_id":    realmID,
		"realm_level": realmLevel,
		"attack":      attack,
		"defense":     defense,
		"max_hp":      hp,
	})
	url := fmt.Sprintf("%s/api/v1/player/%d/update-attributes", h.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("通知Player服务失败", "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// triggerAscendAnnouncement 触发飞升全服公告（通过世界服务）
func (h *AscendHandler) triggerAscendAnnouncement(playerID uint64, playerName string) {
	announcement := map[string]interface{}{
		"type":      "ascend",
		"message":   fmt.Sprintf("【天道公告】玩家 %s 渡劫圆满，破空飞升仙界！天降祥瑞，万物生辉！", playerName),
		"player_id": playerID,
		"timestamp": time.Now().Unix(),
	}
	body, _ := json.Marshal(announcement)
	url := fmt.Sprintf("%s/api/v1/world/ascend/announce", h.worldServiceAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("触发全服公告失败", "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// requireOwnPlayerID 检查认证用户是否操作自己的数据
func (h *AscendHandler) requireOwnPlayerID(r *http.Request, playerID uint64) bool {
	authID, ok := GetAuthPlayerID(r)
	if !ok {
		return false
	}
	return authID == playerID
}
