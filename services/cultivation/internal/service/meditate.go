// Package service 离线修炼系统
package service
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
	"cultivation-game/services/cultivation/internal/model"
	"github.com/redis/go-redis/v9"
)
// MeditationState 闭关状态快照
type MeditationState struct {
	PlayerID       uint64  // 玩家ID
	StartTime      int64   // 闭关开始时间(unix时间戳)
	AccumulatedExp int64   // 已累计修为
	LastTickTime   int64   // 上次心跳处理时间
	ExpPerSecond   int64   // 离线修炼效率(修为/秒, 在线的20%)
	PlayerName     string  // 玩家名(日志用)
	RealmID        int     // 闭关时的境界ID(日志用)
	RealmLevel     int     // 闭关时的境界等级(日志用)
}
// MeditateService 离线修炼服务
// 管理玩家闭关状态的记录、心跳累计和领取
type MeditateService struct {
	logger *slog.Logger
	mu                sync.RWMutex
	states            map[uint64]*MeditationState // 闭关中的玩家状态
	realmSvc          *RealmService               // 用于计算修炼效率
	rdb               *redis.Client               // 用于更新排行榜
	playerServiceAddr string                      // Player 服务 HTTP 地址
}
// NewMeditateService 创建离线修炼服务
func NewMeditateService(logger *slog.Logger, realmSvc *RealmService, rdb *redis.Client) *MeditateService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &MeditateService{
		logger: logger,
		states:            make(map[uint64]*MeditationState),
		realmSvc:          realmSvc,
		rdb:               rdb,
		playerServiceAddr: playerAddr,
	}
}
// StartMeditation 开始闭关
// 记录闭关开始时间，缓存离线效率快照
func (s *MeditateService) StartMeditation(player *model.Player) {
	// 计算玩家修炼效率（在线）
	eff := s.realmSvc.CalculateCultivationEfficiency(player)
	// eff.ExpPerSecond 是每分钟修为值
	// 离线效率 = 在线效率的 20%（修为/秒）= ExpPerSecond / 60 / 5
	offlineExpPerSec := eff.ExpPerSecond / 300
	if offlineExpPerSec < 1 {
		offlineExpPerSec = 1
	}
	now := time.Now().Unix()
	s.mu.Lock()
	s.states[player.ID] = &MeditationState{
		PlayerID:       player.ID,
		StartTime:      now,
		AccumulatedExp: 0,
		LastTickTime:   now,
		ExpPerSecond:   offlineExpPerSec,
		PlayerName:     player.Name,
		RealmID:        player.RealmID,
		RealmLevel:     player.RealmLevel,
	}
	s.mu.Unlock()
	// 同步更新玩家对象
	player.IsMeditating = true
	player.MeditationStart = now
	player.AccumulatedExp = 0
	s.logger.Info("玩家开始闭关", "player_id", player.ID, "player_name", player.Name, "offline_exp_per_sec", offlineExpPerSec, "realm_id", player.RealmID, "realm_level", player.RealmLevel)
}
// ProcessTick 心跳处理：为所有闭关中的玩家累计修为
// 每秒增加 "离线效率 × 自上次心跳的秒数" 修为
// 上限7天，超过不再累计
func (s *MeditateService) ProcessTick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	maxDuration := int64(7 * 24 * 3600) // 最多累计7天
	for id, state := range s.states {
		// 计算自上次心跳以来的时间差(秒)
		elapsed := now - state.LastTickTime
		if elapsed <= 0 {
			continue
		}
		// 总闭关时长
		totalElapsed := now - state.StartTime
		if totalElapsed > maxDuration {
			// 超过7天不再累计，但仍保留状态等待领取
			state.LastTickTime = now
			continue
		}
		// 限制单次tick处理不超过60秒（防止服务器卡顿时一次加太多）
		if elapsed > 60 {
			elapsed = 60
		}
		// 计算本次应得修为
		gained := state.ExpPerSecond * elapsed
		state.AccumulatedExp += gained
		state.LastTickTime = now
		_ = id // 日志调试时可使用
	}
}
// ClaimMeditation 领取闭关收益
// 返回本次闭关累计获得的修为值
// 同时更新玩家修为、重置闭关状态，并同步排行榜
func (s *MeditateService) ClaimMeditation(player *model.Player) int64 {
	s.mu.Lock()
	state, hasState := s.states[player.ID]
	if hasState {
		delete(s.states, player.ID)
	}
	s.mu.Unlock()
	if !hasState {
		// 内存中无状态（可能是服务器重启），回退到传统结算方式
		return s.settleFallback(player)
	}
	// 如果在tick后又流逝了一些时间，补上最后的收益
	now := time.Now().Unix()
	lastElapsed := now - state.LastTickTime
	if lastElapsed > 0 && lastElapsed < 60 {
		extraGain := state.ExpPerSecond * lastElapsed
		state.AccumulatedExp += extraGain
	}
	gainedExp := state.AccumulatedExp
	if gainedExp < 0 {
		gainedExp = 0
	}
	// 发放修为
	player.Experience += gainedExp
	player.IsMeditating = false
	player.MeditationStart = 0
	player.AccumulatedExp = 0
	s.logger.Info("玩家领取闭关收益", "player_id", player.ID, "player_name", state.PlayerName, "gained_exp", gainedExp, "accumulated", state.AccumulatedExp, "total_exp", player.Experience)
	// 异步更新排行榜分数
	go s.updateRanking(player.ID, player.Experience)
	// 异步通知Player服务
	go s.notifyPlayerService(player.ID, gainedExp)
	return gainedExp
}
// GetState 查询玩家的闭关状态（只读）
func (s *MeditateService) GetState(playerID uint64) *MeditationState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[playerID]
	if !ok {
		return nil
	}
	// 返回副本防止外部篡改
	copy := *state
	return &copy
}
// GetMeditatingCount 获取当前闭关人数
func (s *MeditateService) GetMeditatingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}
// settleFallback 回退结算：当内存状态丢失时，根据玩家数据计算闭关收益
func (s *MeditateService) settleFallback(player *model.Player) int64 {
	if !player.IsMeditating || player.MeditationStart <= 0 {
		return 0
	}
	elapsed := time.Now().Unix() - player.MeditationStart
	if elapsed < 0 {
		elapsed = 0
	}
	maxDuration := int64(7 * 24 * 3600)
	if elapsed > maxDuration {
		elapsed = maxDuration
	}
	// 计算在线效率
	eff := s.realmSvc.CalculateCultivationEfficiency(player)
	// 使用20%离线效率 (ExpPerSecond是每分钟, 转为每秒再除5)
	gainedExp := eff.ExpPerSecond / 300 * elapsed
	if gainedExp < 0 {
		gainedExp = 0
	}
	player.Experience += gainedExp
	player.IsMeditating = false
	player.MeditationStart = 0
	player.AccumulatedExp = 0
	s.logger.Info("玩家回退结算闭关收益", "player_id", player.ID, "gained_exp", gainedExp)
	go s.updateRanking(player.ID, player.Experience)
	go s.notifyPlayerService(player.ID, gainedExp)
	return gainedExp
}
// updateRanking 异步更新排行榜分数
// 修为榜评分 = 玩家当前总修为
func (s *MeditateService) updateRanking(playerID uint64, exp int64) {
	if s.rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := s.rdb.ZAdd(ctx, "ranking:cultivation", redis.Z{
		Score:  float64(exp),
		Member: playerID,
	}).Err()
	if err != nil {
		s.logger.Error("更新玩家修为榜失败", "player_id", playerID, "error", err)
	}
}
// notifyPlayerService 异步通知 Player 服务修为变化
func (s *MeditateService) notifyPlayerService(playerID uint64, gainedExp int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"spirit_power": gainedExp,
	})
	url := fmt.Sprintf("%s/api/v1/player/%d/update-exp", s.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("通知Player服务修为变化失败", "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}