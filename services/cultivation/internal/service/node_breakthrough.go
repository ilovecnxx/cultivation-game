// Package service 突破节点小游戏引擎
//
// 突破流程:
//   1. StartBreakthrough 创建突破会话，确定节点总数和时限
//   2. 前端在一定时间内点击出现的节点，每次调用 CollectNode
//   3. 收集足够节点后自动成功，超时则失败
//   4. 道心层数影响节点生成速度（每层-5%）
//   5. 丹药影响时限和判定范围
package service

import (
	"fmt"
	"sync"
	"time"

	"cultivation-game/services/cultivation/internal/model"
)

// BreakthroughSession 突破节点收集会话
type BreakthroughSession struct {
	PlayerID    uint64    `json:"player_id"`
	SessionID   string    `json:"session_id"`
	RealmID     int       `json:"realm_id"`     // 当前大境界ID
	RealmLevel  int       `json:"realm_level"`  // 当前小境界等级
	TotalNodes  int       `json:"total_nodes"`  // 需要收集的总节点数
	Collected   int       `json:"collected"`    // 已收集数
	TimeLimit   int64     `json:"time_limit"`   // 总时限（秒）
	NodeInterval int64    `json:"node_interval"` // 节点出现间隔（毫秒）
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"` // active / success / failed
}

// BreakthroughResult 突破结果
type BreakthroughNodeResult struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	Collected int    `json:"collected"`
	Total     int    `json:"total"`
	Message   string `json:"message"`
}

// NodeBreakthroughService 突破节点小游戏服务
type NodeBreakthroughService struct {
	mu       sync.RWMutex
	sessions map[string]*BreakthroughSession // sessionID -> session
}

// NewNodeBreakthroughService 创建突破节点服务
func NewNodeBreakthroughService() *NodeBreakthroughService {
	return &NodeBreakthroughService{
		sessions: make(map[string]*BreakthroughSession),
	}
}

// StartBreakthrough 开始突破节点小游戏
//
// 参数:
//   - player: 玩家对象（用于读取境界、道心层数）
//   - pillTimeBonus: 丹药附加的时限（秒），如凝神丹 +30s
//   - pillRangeBonus: 丹药附加的判定范围加成（百分比），如聚灵丹 +0.2
//
// 节点数规则:
//   - 大境界突破（如练气→筑基）: totalNodes = 30+, timeLimit = 2min
//   - 炼气8-10层: totalNodes = 16-20, timeLimit = 2.5min
//   - 炼气4-7层: totalNodes = 8-14, timeLimit = 3min
//   - 炼气1-3层: totalNodes = 2-6, timeLimit = 3min
//
// 道心效果: 每层 DaoXinStacks 使节点生成间隔 +5%（减速）
// 丹药效果: pillTimeBonus 增加总时限, pillRangeBonus 增加判定范围
func (s *NodeBreakthroughService) StartBreakthrough(player *model.Player, pillTimeBonus int64, pillRangeBonus float64) *BreakthroughSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 根据境界和等级确定节点数和时限
	totalNodes, timeLimitSec := calcBreakthroughParams(player.RealmID, player.RealmLevel)

	// 丹药加成：增加时限
	timeLimitSec += pillTimeBonus
	if timeLimitSec < 30 {
		timeLimitSec = 30 // 最少30秒
	}

	// 节点出现间隔（毫秒）
	// 基础间隔 = 总时限(秒) * 1000 / 总节点数 / 2，确保有足够时间收集
	nodeIntervalMs := int64(float64(timeLimitSec*1000) / float64(totalNodes) / 2.5)
	if nodeIntervalMs < 200 {
		nodeIntervalMs = 200 // 最快200ms一个节点
	}
	if nodeIntervalMs > 3000 {
		nodeIntervalMs = 3000 // 最慢3s一个节点
	}

	// 道心影响：每层 +5% 间隔（减速）
	daoXinPenalty := 1.0 + float64(player.DaoXinStacks)*0.05
	nodeIntervalMs = int64(float64(nodeIntervalMs) * daoXinPenalty)

	// 生成唯一 SessionID
	sessionID := fmt.Sprintf("bt_%d_%d", player.ID, time.Now().UnixNano())

	session := &BreakthroughSession{
		PlayerID:     player.ID,
		SessionID:    sessionID,
		RealmID:      player.RealmID,
		RealmLevel:   player.RealmLevel,
		TotalNodes:   totalNodes,
		Collected:    0,
		TimeLimit:    timeLimitSec,
		NodeInterval: nodeIntervalMs,
		CreatedAt:    time.Now(),
		Status:       "active",
	}

	s.sessions[sessionID] = session
	return session
}

// CollectNode 收集一个突破节点
//
// 返回:
//   - (true, nil) 表示收集成功，但未完成
//   - (true, result) 表示收集成功且突破完成
//   - (false, nil) 表示会话不存在或已结束
func (s *NodeBreakthroughService) CollectNode(sessionID, nodeID string) (bool, *BreakthroughNodeResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return false, nil
	}
	if session.Status != "active" {
		return false, nil
	}

	// 收集节点
	session.Collected++

	// 检查是否完成
	if session.Collected >= session.TotalNodes {
		session.Status = "success"
		result := &BreakthroughNodeResult{
			Success:   true,
			SessionID: session.SessionID,
			Collected: session.Collected,
			Total:     session.TotalNodes,
			Message:   "突破成功！成功收集所有节点！",
		}
		// 清理已完成会话（延时清理，防止并发问题）
		go func() {
			time.Sleep(5 * time.Second)
			s.mu.Lock()
			delete(s.sessions, sessionID)
			s.mu.Unlock()
		}()
		return true, result
	}

	return true, nil
}

// CheckTimeout 检查突破会话是否超时
//
// 返回:
//   - nil: 会话活跃中或不存在
//   - *BreakthroughNodeResult: 超时失败的突破结果
func (s *NodeBreakthroughService) CheckTimeout(sessionID string) *BreakthroughNodeResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	if session.Status != "active" {
		return nil
	}

	// 检查是否超时
	elapsed := time.Since(session.CreatedAt)
	if elapsed.Seconds() >= float64(session.TimeLimit) {
		session.Status = "failed"
		result := &BreakthroughNodeResult{
			Success:   false,
			SessionID: session.SessionID,
			Collected: session.Collected,
			Total:     session.TotalNodes,
			Message:   fmt.Sprintf("突破失败！超时未完成，已收集 %d/%d 个节点", session.Collected, session.TotalNodes),
		}
		// 清理
		delete(s.sessions, sessionID)
		return result
	}

	return nil
}

// GetSession 获取突破会话（供前端轮询状态）
func (s *NodeBreakthroughService) GetSession(sessionID string) *BreakthroughSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	// 返回副本
	copied := *session
	return &copied
}

// calcBreakthroughParams 根据境界等级计算突破节点参数
//
// 大境界突破:
//   - totalNodes = realmLevel * 3 (最低30)
//   - timeLimit 固定 120s
//
// 小境界突破:
//   - totalNodes = realmLevel * 2
//   - timeLimit 随层数增加递减
//
// realmID=1 → 炼气期, realmLevel=1~10
func calcBreakthroughParams(realmID, realmLevel int) (totalNodes int, timeLimitSec int64) {
	if realmLevel >= 10 {
		// 大境界突破（满级突破）
		totalNodes = 30 + (realmID-1)*5
		if totalNodes < 30 {
			totalNodes = 30
		}
		timeLimitSec = 120
	} else if realmLevel >= 8 {
		// 炼气8-10层（高等级小境界）
		totalNodes = 16 + (realmLevel-8)*2 // 8层=16, 9层=18, 10层=20
		timeLimitSec = 150                  // 2.5min
	} else if realmLevel >= 4 {
		// 炼气4-7层（中等级小境界）
		totalNodes = 8 + (realmLevel-4)*2 // 4层=8, 5层=10, 6层=12, 7层=14
		timeLimitSec = 180                  // 3min
	} else {
		// 炼气1-3层（低等级小境界）
		totalNodes = 2 + (realmLevel-1)*2 // 1层=2, 2层=4, 3层=6
		timeLimitSec = 180                  // 3min
	}
	return
}
