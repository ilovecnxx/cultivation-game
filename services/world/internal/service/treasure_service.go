// Package service 藏宝图系统
//
// 游历/奇遇概率获得藏宝图碎片(1/4)
// 拼合4张碎片->完整藏宝图->揭示埋宝地点
// 挖宝: 概率获得稀有物品/大量灵石/功法/碎片化本命法宝
package service

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// 使用内存存储（实际项目中替换为数据库/Redis）
type playerTreasure struct {
	Fragments    [4]bool // 四片碎片收集标记
	Completed    bool    // 已拼合
	Digged       bool    // 已挖宝
	CreatedAt    int64
}

// TreasureService 藏宝图业务
type TreasureService struct {
	mu   sync.RWMutex
	data map[int64]*playerTreasure // key: playerID
}

// NewTreasureService 创建 TreasureService
func NewTreasureService() *TreasureService {
	return &TreasureService{
		data: make(map[int64]*playerTreasure),
	}
}

// GetFragments 获取玩家藏宝图碎片状态
// GET /api/v1/treasure/fragments?player_id=xxx
func (s *TreasureService) GetFragments(playerID int64) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pt, ok := s.data[playerID]
	if !ok {
		return map[string]interface{}{
			"fragments":  [4]bool{false, false, false, false},
			"count":      0,
			"completed":  false,
			"digged":     false,
		}
	}
	count := 0
	for _, f := range pt.Fragments {
		if f {
			count++
		}
	}
	return map[string]interface{}{
		"fragments":  pt.Fragments,
		"count":      count,
		"completed":  pt.Completed,
		"digged":     pt.Digged,
	}
}

// AddFragment 添加藏宝图碎片（游历/奇遇时调用）
func (s *TreasureService) AddFragment(playerID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pt, ok := s.data[playerID]
	if !ok {
		pt = &playerTreasure{
			Fragments: [4]bool{false, false, false, false},
			CreatedAt: time.Now().Unix(),
		}
		s.data[playerID] = pt
	}

	// 如果已拼合完成，不再掉落碎片
	if pt.Completed {
		return false, fmt.Errorf("已拥有完整藏宝图，请先挖宝")
	}

	// 选一个未收集的碎片位
	var avail []int
	for i, f := range pt.Fragments {
		if !f {
			avail = append(avail, i)
		}
	}
	if len(avail) == 0 {
		return false, fmt.Errorf("碎片已收集完毕，请拼合藏宝图")
	}

	idx := avail[rand.Intn(len(avail))]
	pt.Fragments[idx] = true

	return true, nil
}

// CombineFragments 拼合藏宝图
// POST /api/v1/treasure/combine
func (s *TreasureService) CombineFragments(playerID int64) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pt, ok := s.data[playerID]
	if !ok {
		return nil, fmt.Errorf("没有藏宝图碎片")
	}

	// 检查是否4片齐全
	for i, f := range pt.Fragments {
		if !f {
			return nil, fmt.Errorf("碎片不全，缺少第 %d 片", i+1)
		}
	}

	if pt.Completed {
		return nil, fmt.Errorf("藏宝图已拼合")
	}

	pt.Completed = true

	return map[string]interface{}{
		"message": "拼合成功！一张完整的藏宝图出现在你面前",
		"map_id":  fmt.Sprintf("treasure_%d_%d", playerID, time.Now().Unix()),
	}, nil
}

// DigTreasure 挖宝
// POST /api/v1/treasure/dig
func (s *TreasureService) DigTreasure(playerID int64) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pt, ok := s.data[playerID]
	if !ok || !pt.Completed {
		return nil, fmt.Errorf("尚未拼合藏宝图，无法挖宝")
	}
	if pt.Digged {
		return nil, fmt.Errorf("该藏宝图已挖过宝")
	}

	// 抽取奖励
	reward := model.DrawReward()
	pt.Digged = true

	return map[string]interface{}{
		"message": "挖宝成功！",
		"reward": map[string]interface{}{
			"type":   reward.Type,
			"name":   reward.Name,
			"amount": reward.Amount,
		},
	}, nil
}

// TryDropFragment 游历/奇遇时尝试掉落碎片（20%概率）
func (s *TreasureService) TryDropFragment(playerID int64) (bool, string, error) {
	if rand.Float64() >= 0.20 {
		return false, "", nil
	}

	ok, err := s.AddFragment(playerID)
	if err != nil {
		return false, "", err
	}
	if ok {
		return true, "获得一张藏宝图碎片！", nil
	}
	return false, "", nil
}
