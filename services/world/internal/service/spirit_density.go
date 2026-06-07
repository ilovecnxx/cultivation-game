// Package service 灵气浓度管理系统
//
// 功能:
//   - 每个区域有基础灵气浓度值 (0.5~5.0)
//   - 玩家当前区域决定修炼速度
//   - 游历中有概率发现「隐藏灵脉」(持续2小时的临时加成)
//   - 某些灵脉是公共资源，有数量上限
package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SpiritDensityService 灵气浓度管理服务
type SpiritDensityService struct {
	mu             sync.RWMutex
	regionDensities map[string]float64 // regionID -> spirit_density (0.5~5.0)

	// 隐藏灵脉（玩家私人临时加成）
	hiddenVeins map[string]*HiddenSpiritVein // sessionID -> vein

	// 公共灵脉（全服共享，有数量上限）
	publicVeins    map[string]*PublicSpiritVein // veinID -> vein
	publicVeinSlots int                          // 公共灵脉最大数量
}

// HiddenSpiritVein 隐藏灵脉（玩家游历中发现的临时灵气加成）
type HiddenSpiritVein struct {
	ID        string    `json:"id"`
	PlayerID  uint64    `json:"player_id"`
	RegionID  string    `json:"region_id"`
	Density   float64   `json:"density"`    // 额外灵气浓度
	ExpiresAt time.Time `json:"expires_at"` // 过期时间（2小时）
	FoundAt   time.Time `json:"found_at"`
}

// PublicSpiritVein 公共灵脉（全服共享资源）
type PublicSpiritVein struct {
	ID        string    `json:"id"`
	RegionID  string    `json:"region_id"`
	Density   float64   `json:"density"`    // 灵气浓度加成
	ExpiresAt time.Time `json:"expires_at"` // 过期时间
	OwnerID   uint64    `json:"owner_id"`   // 占领者玩家ID
	ClaimedAt time.Time `json:"claimed_at"`
}

// NewSpiritDensityService 创建灵气浓度服务
//
// regionsPath 指向 map_regions.json，用于初始化各区域基础灵气浓度
// maxPublicVeins 为公共灵脉最大数量上限
func NewSpiritDensityService(regionsPath string, maxPublicVeins int) (*SpiritDensityService, error) {
	s := &SpiritDensityService{
		regionDensities:  make(map[string]float64),
		hiddenVeins:      make(map[string]*HiddenSpiritVein),
		publicVeins:      make(map[string]*PublicSpiritVein),
		publicVeinSlots:  maxPublicVeins,
	}

	if err := s.loadRegionDensities(regionsPath); err != nil {
		return nil, fmt.Errorf("加载区域灵气浓度失败: %w", err)
	}

	return s, nil
}

// loadRegionDensities 从 map_regions.json 读取各区域灵气浓度
func (s *SpiritDensityService) loadRegionDensities(regionsPath string) error {
	path := regionsPath
	// 尝试从多个相对路径查找
	if _, err := os.Stat(path); os.IsNotExist(err) {
		altPaths := []string{
			filepath.Join("..", "..", regionsPath),
			filepath.Join("internal", "data", "map_regions.json"),
		}
		found := false
		for _, p := range altPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				found = true
				break
			}
		}
		if !found {
			// 使用默认值，不阻塞启动
			s.regionDensities["newbie_village_01"] = 0.5
			s.regionDensities["qingyun_range_01"] = 0.8
			s.regionDensities["star_city_01"] = 1.0
			return nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var regions []struct {
		ID        string `json:"id"`
		Resources struct {
			SpiritQi float64 `json:"spirit_qi"`
		} `json:"resources"`
		SpiritDensity float64 `json:"spirit_density,omitempty"` // V3 字段（可选，优先使用）
	}
	if err := json.Unmarshal(data, &regions); err != nil {
		return err
	}

	for _, r := range regions {
		density := r.SpiritDensity
		if density <= 0 {
			// 回退到旧的 spirit_qi 字段
			density = r.Resources.SpiritQi
		}
		if density < 0.5 {
			density = 0.5
		}
		if density > 5.0 {
			density = 5.0
		}
		s.regionDensities[r.ID] = density
	}
	return nil
}

// GetSpiritDensity 获取指定区域的灵气浓度
//
// 返回值:
//   - density: 灵气浓度 (0.5~5.0)
//   - bonus: 该区域上的隐藏灵脉/公共灵脉额外加成（如有）
//   - ok: 区域是否存在
func (s *SpiritDensityService) GetSpiritDensity(regionID string) (density float64, bonus float64, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	base, exists := s.regionDensities[regionID]
	if !exists {
		return 0, 0, false
	}

	// 计算活跃的隐藏灵脉加成
	var totalBonus float64
	now := time.Now()
	for _, vein := range s.hiddenVeins {
		if vein.RegionID == regionID && now.Before(vein.ExpiresAt) {
			totalBonus += vein.Density
		}
	}

	// 计算活跃的公共灵脉加成
	for _, vein := range s.publicVeins {
		if vein.RegionID == regionID && now.Before(vein.ExpiresAt) {
			totalBonus += vein.Density
		}
	}

	result := base + totalBonus
	if result > 5.0 {
		result = 5.0
	}
	if result < 0.5 {
		result = 0.5
	}

	return result, totalBonus, true
}

// TryDiscoverHiddenVein 游历中有概率发现隐藏灵脉
//
// 参数:
//   - playerID: 发现者
//   - regionID: 发现区域
//   - discoverProb: 发现概率 (0~1)
//
// 返回:
//   - vein: 发现的灵脉（nil 表示未发现）
//   - message: 提示信息
func (s *SpiritDensityService) TryDiscoverHiddenVein(playerID uint64, regionID string, discoverProb float64) (*HiddenSpiritVein, string) {
	if rand.Float64() >= discoverProb {
		return nil, ""
	}

	// 发现隐藏灵脉：灵气浓度 +0.5~2.0，持续2小时
	densityBonus := 0.5 + rand.Float64()*1.5
	veinID := fmt.Sprintf("hidden_%d_%d", playerID, time.Now().UnixNano())

	vein := &HiddenSpiritVein{
		ID:        veinID,
		PlayerID:  playerID,
		RegionID:  regionID,
		Density:   densityBonus,
		ExpiresAt: time.Now().Add(2 * time.Hour),
		FoundAt:   time.Now(),
	}

	s.mu.Lock()
	s.hiddenVeins[veinID] = vein
	s.mu.Unlock()

	message := fmt.Sprintf("你发现了一处隐藏灵脉！区域灵气浓度 +%.1f，持续2小时", densityBonus)
	return vein, message
}

// ClaimPublicVein 占领公共灵脉（有数量上限）
//
// 参数:
//   - playerID: 占领者
//   - regionID: 灵脉所在区域
//   - density: 灵脉灵气浓度加成
//   - duration: 持续时间
//
// 返回:
//   - vein: 创建的公共灵脉（nil 表示已达上限）
//   - message: 提示信息
func (s *SpiritDensityService) ClaimPublicVein(playerID uint64, regionID string, density float64, duration time.Duration) (*PublicSpiritVein, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查上限
	activeCount := 0
	now := time.Now()
	for _, v := range s.publicVeins {
		if now.Before(v.ExpiresAt) {
			activeCount++
		}
	}
	if activeCount >= s.publicVeinSlots {
		return nil, "公共灵脉已满，无法占领"
	}

	veinID := fmt.Sprintf("public_%d_%d", playerID, time.Now().UnixNano())
	vein := &PublicSpiritVein{
		ID:        veinID,
		RegionID:  regionID,
		Density:   density,
		ExpiresAt: time.Now().Add(duration),
		OwnerID:   playerID,
		ClaimedAt: time.Now(),
	}

	s.publicVeins[veinID] = vein

	message := fmt.Sprintf("成功占领公共灵脉！区域灵气浓度 +%.1f，持续 %d 分钟", density, int(duration.Minutes()))
	return vein, message
}

// UpdateRegionDensity 动态调整区域基础灵气浓度（GM/事件用）
func (s *SpiritDensityService) UpdateRegionDensity(regionID string, density float64) error {
	if density < 0.5 || density > 5.0 {
		return fmt.Errorf("灵气浓度必须在 0.5~5.0 之间")
	}
	s.mu.Lock()
	s.regionDensities[regionID] = density
	s.mu.Unlock()
	return nil
}

// CleanExpiredVeins 清理过期灵脉（定时任务调用）
func (s *SpiritDensityService) CleanExpiredVeins() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, vein := range s.hiddenVeins {
		if now.After(vein.ExpiresAt) {
			delete(s.hiddenVeins, id)
		}
	}
	for id, vein := range s.publicVeins {
		if now.After(vein.ExpiresAt) {
			delete(s.publicVeins, id)
		}
	}
}
