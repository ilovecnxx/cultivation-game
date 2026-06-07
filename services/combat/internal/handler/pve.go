package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/engine"
	"cultivation-game/services/combat/internal/model"
	"cultivation-game/services/combat/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// PVEHandler PVE 战斗处理器
type PVEHandler struct {
	cfg                  *config.Config
	monsterLoader        *MonsterLoader
	instanceLoader       *InstanceLoader
	rewardCalc           *service.RewardCalculator
	playerServiceAddr    string // Player 服务 HTTP 地址
	questServiceAddr     string // World/Quest 服务 HTTP 地址
	cultivationSvcAddr   string // Cultivation 服务 HTTP 地址
}

// MonsterLoader 怪物数据加载器
type MonsterLoader struct {
	Monsters map[string]*model.Fighter `json:"monsters"`
}

// InstanceConfig 副本配置
type InstanceConfig struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Level       int                 `json:"level"`
	Description string              `json:"description"`
	MonsterGroups [][]string        `json:"monster_groups"` // 每波怪物的ID列表
	Rewards     *service.DropTable  `json:"rewards"`
}

// InstanceLoader 副本数据加载器
type InstanceLoader struct {
	Instances map[string]*InstanceConfig `json:"instances"`
}

// NewPVEHandler 创建 PVE 处理器
func NewPVEHandler(cfg *config.Config) *PVEHandler {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	questAddr := os.Getenv("QUEST_SERVICE_ADDR")
	if questAddr == "" {
		questAddr = "http://127.0.0.1:8083"
	}
	cultivationAddr := os.Getenv("CULTIVATION_SERVICE_ADDR")
	if cultivationAddr == "" {
		cultivationAddr = "http://127.0.0.1:8080"
	}
	return &PVEHandler{
		cfg:                  cfg,
		monsterLoader:        &MonsterLoader{Monsters: make(map[string]*model.Fighter)},
		instanceLoader:       &InstanceLoader{Instances: make(map[string]*InstanceConfig)},
		rewardCalc:           service.NewRewardCalculator(),
		playerServiceAddr:    playerAddr,
		questServiceAddr:     questAddr,
		cultivationSvcAddr:   cultivationAddr,
	}
}

// StartBattleRequest PVE 战斗请求
type StartBattleRequest struct {
	PlayerTeam []PlayerFighterInfo `json:"player_team"`
	MonsterID  string              `json:"monster_id,omitempty"`
	InstanceID string              `json:"instance_id,omitempty"`
}

// PlayerFighterInfo 玩家战斗信息
type PlayerFighterInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Element   string `json:"element"`
	Attack    float64 `json:"attack"`
	Defense   float64 `json:"defense"`
	Speed     float64 `json:"speed"`
	HP        float64 `json:"hp"`
	MaxHP     float64 `json:"max_hp"`
	MP        int    `json:"mp"`
	CritRate  float64 `json:"crit_rate"`
	CritDmg   float64 `json:"crit_damage"`
}

// StartBattle 处理 PVE 战斗开始请求
//
// POST /api/v1/pve/battle
func (h *PVEHandler) StartBattle(c *gin.Context) {
	var req StartBattleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 构建玩家队伍
	playerTeam, err := h.buildPlayerTeam(req.PlayerTeam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构建敌人队伍
	var enemyTeam []*model.Fighter
	if req.InstanceID != "" {
		enemyTeam, err = h.buildInstanceEnemy(req.InstanceID)
	} else if req.MonsterID != "" {
		enemyTeam, err = h.buildMonsterEnemy(req.MonsterID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定怪物ID或副本ID"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建并开始战斗
	battle := engine.NewBattle(playerTeam, enemyTeam, &h.cfg.Game)
	result := battle.Start()

	// 计算奖励
	if result.State == engine.BattleStatePlayerWin {
		playerLevel := 1
		playerID := int64(0)
		if len(req.PlayerTeam) > 0 {
			playerLevel = req.PlayerTeam[0].Level
			// 尝试从玩家 ID 中提取数字部分作为 Player 服务中的 playerID
			id, err := strconv.ParseInt(req.PlayerTeam[0].ID, 10, 64)
			if err == nil {
				playerID = id
			}
		}
		rewards := h.rewardCalc.CalculateRewards(result, playerLevel, len(playerTeam), 0)
		result.Rewards = rewards

		// 战斗胜利后，异步通知 Player 服务增加经验和物品
		if playerID > 0 {
			go h.sendBattleRewards(playerID, rewards)
			go h.sendQuestProgress(playerID, result.EnemyTeam)
			// 同步历练修为给 Cultivation 服务（基础修炼30分钟）
			cultivationExp := int64(10+int64(playerLevel-1)*5) * 1800
			go h.syncCultivationExp(playerID, cultivationExp)
		}
	}

	log.Info().Str("battle_id", battle.ID).Str("result", string(result.State)).Msg("PVE战斗结束")

	c.JSON(http.StatusOK, result)
}

// GetMonsters 获取怪物列表
//
// GET /api/v1/pve/monsters
// GET /api/v1/combat/monsters?region_id=1  (支持按区域过滤)
func (h *PVEHandler) GetMonsters(c *gin.Context) {
	regionID := c.Query("region_id")
	if regionID == "" {
		// 不传 region_id 返回全部
		c.JSON(http.StatusOK, h.monsterLoader.Monsters)
		return
	}

	// 按区域筛选
	filtered := make(map[string]*model.Fighter)
	for id, m := range h.monsterLoader.Monsters {
		if m.RegionID == regionID {
			filtered[id] = m
		}
	}
	c.JSON(http.StatusOK, filtered)
}

// SweepRequest 快速扫荡请求
type SweepRequest struct {
	PlayerID  int64  `json:"player_id"`
	MonsterID string `json:"monster_id"`
	Count     int    `json:"count"`
}

// SweepResponse 快速扫荡响应
type SweepResponse struct {
	TotalExp  int                `json:"total_exp"`
	TotalGold int                `json:"total_gold"`
	Items     []*engine.DropItem `json:"items,omitempty"`
	Count     int                `json:"count"`
	APCost    int                `json:"ap_cost"` // 消耗的行动力
}

// Sweep 快速扫荡
//
// POST /api/v1/combat/sweep
// 批量计算 N 次战斗结果，不模拟完整回合，直接按公式累加奖励。
// 每次扫荡消耗 1 点行动力。
func (h *PVEHandler) Sweep(c *gin.Context) {
	var req SweepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	if req.Count <= 0 || req.Count > 100 {
		req.Count = 1 // 默认 1 次，上限 100
	}
	if req.MonsterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定怪物ID"})
		return
	}

	monster, ok := h.monsterLoader.Monsters[req.MonsterID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "怪物不存在"})
		return
	}

	// 计算单次战斗的奖励（简化：不跑完整战斗引擎，直接使用 RewardCalculator）
	// 构建一个简化的战斗结果
	battleResult := &engine.BattleResult{
		State:    engine.BattleStatePlayerWin,
		EnemyTeam: []*model.Fighter{monster},
	}

	// 玩家等级取怪物等级上下（简化：使用怪物等级）
	playerLevel := monster.Level
	rewards := h.rewardCalc.CalculateRewards(battleResult, playerLevel, 1, 0)

	totalExp := rewards.Exp * req.Count
	totalGold := rewards.Gold * req.Count

	// 合并掉落物品（按 count 累加数量）
	itemMap := make(map[string]*engine.DropItem)
	for i := 0; i < req.Count; i++ {
		// 重新计算每次的掉落概率判定以获得变化
		perResult := h.rewardCalc.CalculateRewards(battleResult, playerLevel, 1, 0)
		for _, item := range perResult.Items {
			if existing, ok := itemMap[item.ID]; ok {
				existing.Quantity += item.Quantity
			} else {
				itemMap[item.ID] = &engine.DropItem{
					ID:       item.ID,
					Name:     item.Name,
					Quantity: item.Quantity,
					Rarity:   item.Rarity,
				}
			}
		}
	}

	items := make([]*engine.DropItem, 0, len(itemMap))
	for _, item := range itemMap {
		items = append(items, item)
	}

	// 消耗行动力（实际应调用 Player/World 服务的行动力接口）
	if req.PlayerID > 0 {
		go h.deductActionPoints(req.PlayerID, req.Count)
	}

	// 异步发放奖励
	if req.PlayerID > 0 {
		go h.sendSweepRewards(req.PlayerID, totalExp, totalGold, items)
		// 扫荡也获得历练修为（基础修炼30分钟 × 次数）
		cultivationExp := int64(10+int64(playerLevel-1)*5) * 1800 * int64(req.Count)
		go h.syncCultivationExp(req.PlayerID, cultivationExp)
	}

	log.Info().Int64("player_id", req.PlayerID).Str("monster_id", req.MonsterID).Int("count", req.Count).Int("total_exp", totalExp).Msg("扫荡完成")

	c.JSON(http.StatusOK, SweepResponse{
		TotalExp:  totalExp,
		TotalGold: totalGold,
		Items:     items,
		Count:     req.Count,
		APCost:    req.Count, // 每次扫荡消耗 1 点行动力
	})
}

// syncCultivationExp 同步历练修为到 Cultivation 服务
// 等同于基础修炼30分钟的修为
func (h *PVEHandler) syncCultivationExp(playerID int64, exp int64) {
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
		log.Warn().Err(err).Int64("player_id", playerID).Msg("同步修为到Cultivation服务失败")
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// deductActionPoints 扣除玩家行动力（调用 Player 服务）
func (h *PVEHandler) deductActionPoints(playerID int64, count int) {
	body, _ := json.Marshal(map[string]interface{}{
		"action_points": count,
	})
	url := fmt.Sprintf("%s/api/v1/player/%d/action-points/deduct", h.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Int64("player_id", playerID).Msg("扣除行动力失败(Player服务可能不支持)")
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// sendSweepRewards 扫荡后异步发放总奖励
func (h *PVEHandler) sendSweepRewards(playerID int64, totalExp int, totalGold int, items []*engine.DropItem) {
	// 经验
	if totalExp > 0 {
		expBody, _ := json.Marshal(map[string]interface{}{"exp": int64(totalExp)})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/add-exp", h.playerServiceAddr, playerID), expBody)
	}
	// 金币
	if totalGold > 0 {
		goldBody, _ := json.Marshal(map[string]interface{}{"gold": int64(totalGold), "bound_gold": 0, "jade": 0})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID), goldBody)
	}
	// 物品
	for _, item := range items {
		itemID, err := strconv.ParseInt(item.ID, 10, 64)
		if err != nil {
			continue
		}
		itemBody, _ := json.Marshal(map[string]interface{}{
			"item_id":  itemID,
			"quantity": int32(math.Max(1, float64(item.Quantity))),
		})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/inventory/add", h.playerServiceAddr, playerID), itemBody)
	}
}

// GetInstances 获取副本列表
//
// GET /api/v1/pve/instances
func (h *PVEHandler) GetInstances(c *gin.Context) {
	c.JSON(http.StatusOK, h.instanceLoader.Instances)
}

// sendBattleRewards 战斗胜利后异步通知 Player 服务增加经验和物品
func (h *PVEHandler) sendBattleRewards(playerID int64, rewards *engine.BattleRewards) {
	// 1. 增加经验
	if rewards.Exp > 0 {
		expBody, _ := json.Marshal(map[string]interface{}{"exp": int64(rewards.Exp)})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/add-exp", h.playerServiceAddr, playerID), expBody)
	}

	// 2. 增加货币（金币）
	if rewards.Gold > 0 {
		goldBody, _ := json.Marshal(map[string]interface{}{"gold": int64(rewards.Gold), "bound_gold": 0, "jade": 0})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/currency", h.playerServiceAddr, playerID), goldBody)
	}

	// 3. 掉落物品
	for _, item := range rewards.Items {
		itemID, err := strconv.ParseInt(item.ID, 10, 64)
		if err != nil {
			continue
		}
		itemBody, _ := json.Marshal(map[string]interface{}{
			"item_id":  itemID,
			"quantity": item.Quantity,
		})
		h.postJSON(fmt.Sprintf("%s/api/v1/player/%d/inventory/add", h.playerServiceAddr, playerID), itemBody)
	}
}

// sendQuestProgress 战斗胜利后异步通知 Quest 服务更新任务进度
func (h *PVEHandler) sendQuestProgress(playerID int64, enemyTeam []*model.Fighter) {
	for _, enemy := range enemyTeam {
		if enemy.Type != model.FighterTypeMonster {
			continue
		}
		body, _ := json.Marshal(map[string]interface{}{
			"player_id": fmt.Sprintf("%d", playerID),
			"type":      "kill_monster",
			"target_id": enemy.ID,
			"count":     1,
		})
		h.postJSON(fmt.Sprintf("%s/api/v1/quest/progress", h.questServiceAddr), body)
	}
}

// postJSON HTTP POST 工具方法
func (h *PVEHandler) postJSON(url string, body []byte) {
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

// buildPlayerTeam 从请求体构建玩家队伍
func (h *PVEHandler) buildPlayerTeam(infos []PlayerFighterInfo) ([]*model.Fighter, error) {
	team := make([]*model.Fighter, 0, len(infos))
	for _, info := range infos {
		fighter := model.NewFighter(info.ID, info.Name, model.FighterTypePlayer, model.ElementType(info.Element), info.Level)
		fighter.BaseAttack = info.Attack
		fighter.BaseDefense = info.Defense
		fighter.BaseSpeed = info.Speed
		fighter.BaseHP = info.HP
		fighter.BaseMaxHP = info.MaxHP
		fighter.BaseDefense = info.Defense
		fighter.MP = info.MP
		fighter.MaxMP = info.MP
		fighter.CritRate = info.CritRate
		fighter.CritDamage = info.CritDmg
		if fighter.CritDamage <= 0 {
			fighter.CritDamage = 2.0
		}
		team = append(team, fighter)
	}
	return team, nil
}

// buildMonsterEnemy 根据怪物ID构建敌人
func (h *PVEHandler) buildMonsterEnemy(monsterID string) ([]*model.Fighter, error) {
	monster, ok := h.monsterLoader.Monsters[monsterID]
	if !ok {
		return nil, nil
	}
	return []*model.Fighter{monster}, nil
}

// buildInstanceEnemy 根据副本ID构建敌人波次
func (h *PVEHandler) buildInstanceEnemy(instanceID string) ([]*model.Fighter, error) {
	inst, ok := h.instanceLoader.Instances[instanceID]
	if !ok {
		return nil, nil // 副本不存在
	}

	enemies := make([]*model.Fighter, 0)
	for _, group := range inst.MonsterGroups {
		for _, monsterID := range group {
			if monster, ok := h.monsterLoader.Monsters[monsterID]; ok {
				enemies = append(enemies, monster)
			}
		}
	}
	return enemies, nil
}
