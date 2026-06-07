// Package service 道心系统
//
// 道心机制:
//   - 突破失败时叠加一层道心（最多3层）
//   - 每层道心使下次突破的节点生成速度降低 5%（节点间隔 +5%）
//   - 突破成功时清零
//   - 道心不直接增益或减益修炼速度，仅影响突破小游戏的节点生成频率
package service

import (
	"sync"

	"cultivation-game/services/cultivation/internal/model"
)

// DaoXin 道心实例（内存管理）
type DaoXin struct {
	PlayerID uint64 `json:"player_id"`
	Stacks   int    `json:"stacks"` // 0~3层
}

// DaoXinService 道心管理服务
type DaoXinService struct {
	mu       sync.RWMutex
	daoxins  map[uint64]*DaoXin // playerID -> DaoXin
}

// NewDaoXinService 创建道心服务
func NewDaoXinService() *DaoXinService {
	return &DaoXinService{
		daoxins: make(map[uint64]*DaoXin),
	}
}

// AddDaoXin 突破失败时增加一层道心
//
//   - 每次失败 +1 层
//   - 最多 3 层
//   - 同时更新 player.DaoXinStacks
func (s *DaoXinService) AddDaoXin(player *model.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.daoxins[player.ID]
	if !ok {
		d = &DaoXin{PlayerID: player.ID, Stacks: 0}
		s.daoxins[player.ID] = d
	}

	if d.Stacks < 3 {
		d.Stacks++
	}
	player.DaoXinStacks = d.Stacks
}

// ClearDaoXin 突破成功时清零道心
//
//   - 成功突破后重置为 0
//   - 同时更新 player.DaoXinStacks
func (s *DaoXinService) ClearDaoXin(player *model.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.daoxins[player.ID]
	if !ok {
		d = &DaoXin{PlayerID: player.ID, Stacks: 0}
		s.daoxins[player.ID] = d
	}

	d.Stacks = 0
	player.DaoXinStacks = 0
}

// GetDaoXinStacks 获取玩家当前道心层数（从内存）
func (s *DaoXinService) GetDaoXinStacks(playerID uint64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.daoxins[playerID]
	if !ok {
		return 0
	}
	return d.Stacks
}

// GetDaoXinBonus 获取道心对节点生成速度的减速百分比
//
// 返回值: 减速比例，如 0.05 表示减慢 5%
// 公式: stacks * 0.05
func (s *DaoXinService) GetDaoXinBonus(playerID uint64) float64 {
	stacks := s.GetDaoXinStacks(playerID)
	return float64(stacks) * 0.05
}
