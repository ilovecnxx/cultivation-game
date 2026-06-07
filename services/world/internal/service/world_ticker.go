package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// WorldTickerService 世界事件系统
// 每30分钟检查一次世界事件触发条件
// 支持的事件类型：
//   - 世界Boss刷新（每天20:00固定时间）
//   - 天降异宝（随机区域出现临时采集点）
//   - 灵气潮汐（全服修炼效率临时提升50%）
type WorldTickerService struct {
	mu                sync.Mutex
	exploreSvc        *ExploreService
	regions           []string       // 区域ID列表（用于随机选择）
	events            []*ActiveEvent // 当前活跃的世界事件
	ticker            *time.Ticker
	stopCh            chan struct{}
	running           bool
	socialServiceAddr string // Social 服务 HTTP 地址（用于通知在线玩家）

	// 事件冷却跟踪
	lastBossSpawn    time.Time
	lastTreasureRain time.Time
	lastQiTide       time.Time
}

// ActiveEvent 当前活跃的世界事件
type ActiveEvent struct {
	Type      model.WorldEventType
	Title     string
	StartTime time.Time
	EndTime   time.Time
	RegionID  string
	Params    map[string]interface{}
}

// NewWorldTickerService 创建世界事件服务
func NewWorldTickerService(exploreSvc *ExploreService) *WorldTickerService {
	// 收集所有区域ID
	allRegions := exploreSvc.GetAllRegions()
	regionIDs := make([]string, 0, len(allRegions))
	for _, r := range allRegions {
		regionIDs = append(regionIDs, r.ID)
	}

	socialAddr := os.Getenv("SOCIAL_SERVICE_ADDR")
	if socialAddr == "" {
		socialAddr = "http://127.0.0.1:8084"
	}

	return &WorldTickerService{
		exploreSvc:        exploreSvc,
		regions:           regionIDs,
		events:            make([]*ActiveEvent, 0),
		stopCh:            make(chan struct{}),
		socialServiceAddr: socialAddr,
	}
}

// Start 启动世界事件循环(30分钟间隔)
func (s *WorldTickerService) Start() {
	if s.running {
		return
	}
	s.running = true
	s.ticker = time.NewTicker(30 * time.Minute)

	log.Println("[世界事件] 世界事件系统启动，检查间隔30分钟")

	// 启动时立刻检查一次
	go func() {
		s.checkEvents()
	}()

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkEvents()
			case <-s.stopCh:
				s.ticker.Stop()
				log.Println("[世界事件] 世界事件系统已停止")
				return
			}
		}
	}()
}

// Stop 停止世界事件循环
func (s *WorldTickerService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// GetActiveEvents 获取当前活跃的事件列表
func (s *WorldTickerService) GetActiveEvents() []*ActiveEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*ActiveEvent, 0, len(s.events))
	for _, e := range s.events {
		if time.Now().Before(e.EndTime) {
			result = append(result, e)
		}
	}
	// 清理已过期的事件
	s.events = result
	return result
}

// IsQiTideActive 检查当前是否处于灵气潮汐期间
func (s *WorldTickerService) IsQiTideActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, e := range s.events {
		if e.Type == model.EventMysticMist && now.Before(e.EndTime) {
			return true
		}
	}
	return false
}

// checkEvents 检查所有世界事件触发条件
func (s *WorldTickerService) checkEvents() {
	start := time.Now()
	log.Println("[世界事件] 开始检查事件触发条件...")

	s.checkWorldBoss()
	s.checkTreasureRain()
	s.checkQiTide()

	// 清理过期事件
	s.cleanExpiredEvents()

	log.Printf("[世界事件] 事件检查完成，耗时: %v", time.Since(start))
}

// checkWorldBoss 检查世界Boss刷新
// 每天20:00固定刷新，如果今天还没刷过则触发
func (s *WorldTickerService) checkWorldBoss() {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	bossHour := 20 // 每天20点

	// 只在20:00-20:30的时间窗口内检查
	if now.Hour() != bossHour || now.Minute() > 30 {
		return
	}

	// 检查今天是否已经刷过
	if s.lastBossSpawn.After(today) {
		return
	}

	// 随机选择一个区域
	if len(s.regions) == 0 {
		return
	}
	regionID := s.regions[rand.Intn(len(s.regions))]
	region, ok := s.exploreSvc.GetRegion(regionID)
	if !ok {
		return
	}

	// 确保不选新手村
	if region.Type == model.RegionNewbie && len(s.regions) > 1 {
		for {
			regionID = s.regions[rand.Intn(len(s.regions))]
			if r, ok2 := s.exploreSvc.GetRegion(regionID); ok2 && r.Type != model.RegionNewbie {
				region = r
				break
			}
		}
	}

	// Boss持续2小时
	endTime := now.Add(2 * time.Hour)

	s.mu.Lock()
	s.lastBossSpawn = now
	s.events = append(s.events, &ActiveEvent{
		Type:      model.EventWorldBoss,
		Title:     "世界 Boss 降临",
		StartTime: now,
		EndTime:   endTime,
		RegionID:  regionID,
		Params: map[string]interface{}{
			"region_name": region.Name,
			"boss_level":  rand.Intn(20) + 30, // 30-49级随机
			"boss_name":   "远古妖兽",
		},
	})
	s.mu.Unlock()

	log.Printf("[世界事件] 世界Boss在区域 '%s' 刷新，持续至 %s", region.Name, endTime.Format("15:04"))

		// 通知在线玩家
		msg := fmt.Sprintf("【世界Boss】远古妖兽已降临「%s」，修为30-49级的修士速去讨伐！", region.Name)
		go s.notifyPlayers(msg)
}

// checkTreasureRain 检查天降异宝
// 随机触发：每次检查有15%概率触发，在随机区域生成采集点
// 两次触发间隔至少3小时
func (s *WorldTickerService) checkTreasureRain() {
	now := time.Now()

	// 冷却检查：至少间隔3小时
	if now.Sub(s.lastTreasureRain) < 3*time.Hour {
		return
	}

	// 15%概率触发
	if rand.Float64() > 0.15 {
		return
	}

	if len(s.regions) == 0 {
		return
	}

	// 随机选区域
	regionID := s.regions[rand.Intn(len(s.regions))]
	region, ok := s.exploreSvc.GetRegion(regionID)
	if !ok {
		return
	}

	// 异宝持续1小时
	endTime := now.Add(1 * time.Hour)

	s.mu.Lock()
	s.lastTreasureRain = now
	s.events = append(s.events, &ActiveEvent{
		Type:      model.EventTreasureRain,
		Title:     "天降异宝",
		StartTime: now,
		EndTime:   endTime,
		RegionID:  regionID,
		Params: map[string]interface{}{
			"region_name": region.Name,
			"item_count":  rand.Intn(5) + 3, // 3-7个采集点
			"quality":     []string{"凡品", "下品", "中品", "上品"}[rand.Intn(4)],
		},
	})
	s.mu.Unlock()

	log.Printf("[世界事件] 天降异宝在区域 '%s' 出现，持续至 %s", region.Name, endTime.Format("15:04"))

		// 通知在线玩家
		msg := fmt.Sprintf("【天降异宝】「%s」区域出现了大量天材地宝，快去采集！", region.Name)
		go s.notifyPlayers(msg)
}

// checkQiTide 检查灵气潮汐
// 随机触发：每次检查有10%概率触发
// 触发后全服修炼效率提升50%，持续1小时
// 两次触发间隔至少6小时
func (s *WorldTickerService) checkQiTide() {
	now := time.Now()

	// 冷却检查：至少间隔6小时
	if now.Sub(s.lastQiTide) < 6*time.Hour {
		return
	}

	// 10%概率触发
	if rand.Float64() > 0.10 {
		return
	}

	endTime := now.Add(1 * time.Hour)

	s.mu.Lock()
	s.lastQiTide = now
	s.events = append(s.events, &ActiveEvent{
		Type:      model.EventMysticMist,
		Title:     "灵气潮汐",
		StartTime: now,
		EndTime:   endTime,
		RegionID:  "",
		Params: map[string]interface{}{
			"cultivation_bonus": 0.5, // 修炼效率+50%
		},
	})
	s.mu.Unlock()

	log.Printf("[世界事件] 灵气潮汐降临全服，修炼效率提升50%%，持续至 %s", endTime.Format("15:04"))

		// 通知在线玩家
		msg := "【灵气潮汐】全服灵气浓度大幅提升，修炼效率+50%%，持续1小时！"
		go s.notifyPlayers(msg)
}

// cleanExpiredEvents 清理已过期的事件
func (s *WorldTickerService) cleanExpiredEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	active := make([]*ActiveEvent, 0, len(s.events))
	for _, e := range s.events {
		if now.Before(e.EndTime) {
			active = append(active, e)
		}
	}
	removed := len(s.events) - len(active)
	s.events = active
	if removed > 0 {
		log.Printf("[世界事件] 清理了%d个过期事件", removed)
	}
}

// notifyPlayers 通过 Social 服务的系统频道通知在线玩家
func (s *WorldTickerService) notifyPlayers(content string) {
	body, _ := json.Marshal(map[string]interface{}{
		"content": content,
	})
	url := fmt.Sprintf("%s/api/v1/chat/system-notify", s.socialServiceAddr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[世界事件] 通知Social服务失败: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}
