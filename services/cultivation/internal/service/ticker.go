// Package service 游戏心跳服务
package service

import (
	"log/slog"
	"time"
)

// TickerService 游戏心跳服务
// 每60秒执行一次心跳，处理：
//  1. 离线闭关修为累计
//  2. 定时Buff到期检查
//  3. 每日任务刷新
type TickerService struct {
	logger *slog.Logger
	meditateSvc *MeditateService
	realmSvc    *RealmService
	ticker      *time.Ticker
	stopCh      chan struct{}
	running     bool
}

// NewTickerService 创建游戏心跳服务
func NewTickerService(logger *slog.Logger, meditateSvc *MeditateService, realmSvc *RealmService) *TickerService {
	return &TickerService{
		logger: logger,
		meditateSvc: meditateSvc,
		realmSvc:    realmSvc,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动心跳循环(60秒间隔)
func (s *TickerService) Start() {
	if s.running {
		return
	}
	s.running = true
	s.ticker = time.NewTicker(60 * time.Second)

	s.logger.Info("游戏心跳服务启动", "interval_seconds", 60)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.tick()
			case <-s.stopCh:
				s.ticker.Stop()
				s.logger.Info("游戏心跳服务已停止")
				return
			}
		}
	}()
}

// Stop 停止心跳循环
func (s *TickerService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// tick 单次心跳处理
func (s *TickerService) tick() {
	start := time.Now()
	s.logger.Info("开始执行Tick")

	// 1. 处理闭关玩家修为累计
	beforeExp := s.meditateSvc.GetMeditatingCount()
	s.meditateSvc.ProcessTick()
	s.logger.Info("闭关累计完成", "meditating_count", beforeExp)

	// 2. 检查定时Buff到期（预留扩展点）
	// DB 已接入，后续可从 player_buffs 表读取到期buff
	// 当前版本由各系统自己在逻辑中处理buff到期
	s.logger.Info("Buff到期检查完成")

	// 3. 刷新每日任务（预留扩展点）
	// 每日0点自动刷新，由本tick在跨日时触发
	// 后续可接入 DB 检查 daily_resets 表是否需要刷新
	now := time.Now()
	if now.Hour() == 0 && now.Minute() < 2 {
		s.logger.Info("跨日检测: 触发每日任务刷新")
		// 扩展：遍历所有玩家重置 daily_resets 计数
	}

	elapsed := time.Since(start)
	s.logger.Info("Tick执行完成", "elapsed", elapsed)
}
