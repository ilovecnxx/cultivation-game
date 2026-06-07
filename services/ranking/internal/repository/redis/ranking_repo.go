// Package redis 提供基于 Redis Sorted Set 的排行榜数据访问层。
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"cultivation-game/services/ranking/internal/model"

	"github.com/redis/go-redis/v9"
)

// RankingRepo 排行榜 Redis 仓库。
// 使用 Redis Sorted Set 作为底层存储，每个排行榜类型对应一个 ZSET。
type RankingRepo struct {
	rdb *redis.Client
	log *slog.Logger
}

// NewRankingRepo 创建 RankingRepo。
func NewRankingRepo(rdb *redis.Client, log *slog.Logger) *RankingRepo {
	return &RankingRepo{rdb: rdb, log: log}
}

// ---- 分数更新 ----

// UpdateScore 原子更新玩家在指定排行榜中的分数。
// 同时写入玩家附属信息（昵称、境界）到关联 Hash。
func (r *RankingRepo) UpdateScore(ctx context.Context, rankingType model.RankingType, playerID uint64, score float64, nickname, realmName string) error {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)

	pipe := r.rdb.Pipeline()

	// 更新 Sorted Set 分数
	pipe.ZAdd(ctx, zsetKey, redis.Z{
		Score:  score,
		Member: playerID,
	})

	// 写入玩家附属信息
	entry := &model.RankingEntry{
		PlayerID:  playerID,
		Nickname:  nickname,
		RealmName: realmName,
		Score:     score,
		UpdatedAt: time.Now().Unix(),
	}
	data, _ := json.Marshal(entry)
	pipe.HSet(ctx, infoKey, fmt.Sprintf("%d", playerID), data)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.ErrorContext(ctx, "更新排行榜分数失败",
			"type", rankingType, "player_id", playerID, "error", err)
		return fmt.Errorf("更新排行榜分数失败: %w", err)
	}
	return nil
}

// BatchUpdateScores 批量更新多个玩家的分数（异步批量提交）。
// 使用 Pipeline 减少网络往返。
func (r *RankingRepo) BatchUpdateScores(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry) error {
	if len(entries) == 0 {
		return nil
	}

	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)

	pipe := r.rdb.Pipeline()

	for _, entry := range entries {
		pipe.ZAdd(ctx, zsetKey, redis.Z{
			Score:  entry.Score,
			Member: entry.PlayerID,
		})

		data, _ := json.Marshal(entry)
		pipe.HSet(ctx, infoKey, fmt.Sprintf("%d", entry.PlayerID), data)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.ErrorContext(ctx, "批量更新排行榜分数失败",
			"type", rankingType, "count", len(entries), "error", err)
		return fmt.Errorf("批量更新排行榜分数失败: %w", err)
	}
	return nil
}

// RemovePlayer 从排行榜中移除指定玩家。
func (r *RankingRepo) RemovePlayer(ctx context.Context, rankingType model.RankingType, playerID uint64) error {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)
	activeKey := fmt.Sprintf(model.RedisKeyLastActivity, rankingType)

	pipe := r.rdb.Pipeline()
	pipe.ZRem(ctx, zsetKey, playerID)
	pipe.HDel(ctx, infoKey, fmt.Sprintf("%d", playerID))
	pipe.HDel(ctx, activeKey, fmt.Sprintf("%d", playerID))

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.ErrorContext(ctx, "从排行榜移除玩家失败",
			"type", rankingType, "player_id", playerID, "error", err)
		return fmt.Errorf("从排行榜移除玩家失败: %w", err)
	}
	return nil
}

// ---- 查询 ----

// GetTopN 获取排行榜前 N 名（含分数）。
// ZREVRANGE key 0 N-1 WITHSCORES
func (r *RankingRepo) GetTopN(ctx context.Context, rankingType model.RankingType, n int64) ([]*model.RankingEntry, error) {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)

	results, err := r.rdb.ZRevRangeWithScores(ctx, zsetKey, 0, n-1).Result()
	if err != nil {
		r.log.ErrorContext(ctx, "查询排行榜 TopN 失败", "type", rankingType, "error", err)
		return nil, fmt.Errorf("查询排行榜 TopN 失败: %w", err)
	}

	if len(results) == 0 {
		return []*model.RankingEntry{}, nil
	}

	// 批量查询玩家附属信息
	playerIDs := make([]string, len(results))
	for i, z := range results {
		playerIDs[i] = fmt.Sprintf("%v", z.Member)
	}
	infoData, err := r.rdb.HMGet(ctx, infoKey, playerIDs...).Result()
	if err != nil {
		r.log.ErrorContext(ctx, "查询排行榜玩家信息失败", "type", rankingType, "error", err)
		// 降级：返回只含分数的基础数据
		return r.buildBasicEntries(results), nil
	}

	return r.buildEntriesWithInfo(results, infoData), nil
}

// GetRankingByPage 分页查询排行榜。
// ZREVRANGE key offset count WITHSCORES
func (r *RankingRepo) GetRankingByPage(ctx context.Context, rankingType model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error) {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)

	// 获取总人数
	total, err := r.rdb.ZCard(ctx, zsetKey).Result()
	if err != nil {
		r.log.ErrorContext(ctx, "查询排行榜总人数失败", "type", rankingType, "error", err)
		return nil, 0, fmt.Errorf("查询排行榜总人数失败: %w", err)
	}

	offset := page.Offset()
	count := page.Count()

	results, err := r.rdb.ZRevRangeWithScores(ctx, zsetKey, offset, offset+count-1).Result()
	if err != nil {
		r.log.ErrorContext(ctx, "分页查询排行榜失败", "type", rankingType, "error", err)
		return nil, 0, fmt.Errorf("分页查询排行榜失败: %w", err)
	}

	if len(results) == 0 {
		return []*model.RankingEntry{}, total, nil
	}

	// 批量查询附属信息
	playerIDs := make([]string, len(results))
	for i, z := range results {
		playerIDs[i] = fmt.Sprintf("%v", z.Member)
	}
	infoData, err := r.rdb.HMGet(ctx, infoKey, playerIDs...).Result()
	if err != nil {
		r.log.WarnContext(ctx, "查询排行榜玩家信息失败，降级返回",
			"type", rankingType, "error", err)
		return r.buildBasicEntries(results), total, nil
	}

	entries := r.buildEntriesWithInfo(results, infoData)
	return entries, total, nil
}

// GetPlayerRank 获取玩家在指定排行榜中的排名（从 1 开始）和分数。
// ZREVRANK + ZSCORE
func (r *RankingRepo) GetPlayerRank(ctx context.Context, rankingType model.RankingType, playerID uint64) (int64, float64, error) {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)

	rank, err := r.rdb.ZRevRank(ctx, zsetKey, fmt.Sprintf("%d", playerID)).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, 0, nil // 玩家不在排行榜中
		}
		r.log.ErrorContext(ctx, "查询玩家排名失败", "type", rankingType, "player_id", playerID, "error", err)
		return -1, 0, fmt.Errorf("查询玩家排名失败: %w", err)
	}

	score, err := r.rdb.ZScore(ctx, zsetKey, fmt.Sprintf("%d", playerID)).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, 0, nil
		}
		return -1, 0, fmt.Errorf("查询玩家分数失败: %w", err)
	}

	return rank + 1, score, nil // ZREVRANK 是 0-based，转为 1-based
}

// GetNeighbors 获取玩家排名周围的邻居（上下各 N 名）。
func (r *RankingRepo) GetNeighbors(ctx context.Context, rankingType model.RankingType, playerID uint64, neighborCount int64) (above, below []*model.RankingEntry, err error) {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)

	rank, err := r.rdb.ZRevRank(ctx, zsetKey, fmt.Sprintf("%d", playerID)).Result()
	if err != nil {
		if err == redis.Nil {
			return []*model.RankingEntry{}, []*model.RankingEntry{}, nil
		}
		return nil, nil, fmt.Errorf("查询玩家排名失败: %w", err)
	}

	// 上方（排名更靠前，rank 更小）
	aboveStart := int64(math.Max(0, float64(rank-neighborCount)))
	aboveCount := rank - aboveStart
	var aboveResults []redis.Z
	if aboveCount > 0 {
		aboveResults, err = r.rdb.ZRevRangeWithScores(ctx, zsetKey, aboveStart, rank-1).Result()
		if err != nil {
			return nil, nil, fmt.Errorf("查询上方邻居失败: %w", err)
		}
	}

	// 下方（排名更靠后，rank 更大）
	belowResults, err := r.rdb.ZRevRangeWithScores(ctx, zsetKey, rank+1, rank+neighborCount).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("查询下方邻居失败: %w", err)
	}

	above = r.buildBasicEntries(aboveResults)
	below = r.buildBasicEntries(belowResults)

	return above, below, nil
}

// GetPlayerInfo 获取某个玩家的附属信息（昵称、境界等）。
func (r *RankingRepo) GetPlayerInfo(ctx context.Context, rankingType model.RankingType, playerID uint64) (*model.RankingEntry, error) {
	infoKey := fmt.Sprintf(model.RedisKeyPlayerInfo, rankingType)

	data, err := r.rdb.HGet(ctx, infoKey, fmt.Sprintf("%d", playerID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("查询玩家信息失败: %w", err)
	}

	var entry model.RankingEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("解析玩家信息失败: %w", err)
	}
	return &entry, nil
}

// ---- 快照缓存 ----

// SetSnapshot 缓存排行榜快照（Top N）。
func (r *RankingRepo) SetSnapshot(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry, ttl time.Duration) error {
	key := fmt.Sprintf(model.RedisKeySnapshot, rankingType)
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("序列化快照失败: %w", err)
	}

	return r.rdb.Set(ctx, key, data, ttl).Err()
}

// GetSnapshot 获取缓存的排行榜快照。
func (r *RankingRepo) GetSnapshot(ctx context.Context, rankingType model.RankingType) ([]*model.RankingEntry, error) {
	key := fmt.Sprintf(model.RedisKeySnapshot, rankingType)

	data, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取快照失败: %w", err)
	}

	var entries []*model.RankingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("解析快照失败: %w", err)
	}
	return entries, nil
}

// ---- 分数衰减 ----

// GetInactivePlayers 获取指定时间之前没有活跃记录的玩家 ID 列表。
func (r *RankingRepo) GetInactivePlayers(ctx context.Context, rankingType model.RankingType, deadline time.Time) ([]uint64, error) {
	activeKey := fmt.Sprintf(model.RedisKeyLastActivity, rankingType)
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)

	// 获取所有活跃记录
	activeData, err := r.rdb.HGetAll(ctx, activeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取活跃记录失败: %w", err)
	}

	var inactiveIDs []uint64
	deadlineUnix := deadline.Unix()

	for playerIDStr, lastActiveStr := range activeData {
		lastActive, err := parseUnix(lastActiveStr)
		if err != nil {
			continue
		}
		if lastActive < deadlineUnix {
			playerID, err := parseUint64(playerIDStr)
			if err != nil {
				continue
			}

			// 确认玩家仍在排行榜中
			exists, err := r.rdb.ZScore(ctx, zsetKey, playerIDStr).Result()
			if err != nil || exists == 0 {
				_ = r.rdb.HDel(ctx, activeKey, playerIDStr)
				continue
			}

			inactiveIDs = append(inactiveIDs, playerID)
		}
	}

	return inactiveIDs, nil
}

// ApplyDecayToPlayer 对指定玩家应用分数衰减。
// 新分数 = 当前分数 * (1 - decayRate)
func (r *RankingRepo) ApplyDecayToPlayer(ctx context.Context, rankingType model.RankingType, playerID uint64, decayRate float64) error {
	zsetKey := fmt.Sprintf(model.RedisKeyScoreZSet, rankingType)

	// 使用 Lua 脚本原子执行衰减（防止并发覆盖）
	script := `
		local score = redis.call('ZSCORE', KEYS[1], ARGV[1])
		if score then
			local newScore = tonumber(score) * (1 - tonumber(ARGV[2]))
			redis.call('ZADD', KEYS[1], newScore, ARGV[1])
			return newScore
		end
		return nil
	`

	newScore, err := r.rdb.Eval(ctx, script, []string{zsetKey}, fmt.Sprintf("%d", playerID), decayRate).Result()
	if err != nil {
		r.log.ErrorContext(ctx, "玩家分数衰减失败",
			"type", rankingType, "player_id", playerID, "error", err)
		return fmt.Errorf("玩家分数衰减失败: %w", err)
	}
	r.log.DebugContext(ctx, "玩家分数衰减成功",
		"type", rankingType, "player_id", playerID, "new_score", newScore)
	return nil
}

// ---- 辅助方法 ----

// buildBasicEntries 仅基于 ZSet 结果构建条目（不含昵称等附属信息）。
func (r *RankingRepo) buildBasicEntries(results []redis.Z) []*model.RankingEntry {
	entries := make([]*model.RankingEntry, 0, len(results))
	for i, z := range results {
		playerID, err := parseUint64(fmt.Sprintf("%v", z.Member))
		if err != nil {
			continue
		}
		entries = append(entries, &model.RankingEntry{
			PlayerID: playerID,
			Score:    z.Score,
			Rank:     int32(i + 1),
		})
	}
	return entries
}

// buildEntriesWithInfo 合并 ZSet 结果和 Hash 中的附属信息。
func (r *RankingRepo) buildEntriesWithInfo(results []redis.Z, infoData []interface{}) []*model.RankingEntry {
	entries := make([]*model.RankingEntry, 0, len(results))
	for i, z := range results {
		playerID, err := parseUint64(fmt.Sprintf("%v", z.Member))
		if err != nil {
			continue
		}

		entry := &model.RankingEntry{
			PlayerID: playerID,
			Score:    z.Score,
			Rank:     int32(i + 1),
			UpdatedAt: time.Now().Unix(),
		}

		// 尝试从 Hash 中获取附属信息
		if i < len(infoData) && infoData[i] != nil {
			if dataStr, ok := infoData[i].(string); ok {
				var infoEntry model.RankingEntry
				if err := json.Unmarshal([]byte(dataStr), &infoEntry); err == nil {
					entry.Nickname = infoEntry.Nickname
					entry.RealmName = infoEntry.RealmName
				}
			}
		}

		entries = append(entries, entry)
	}
	return entries
}

// UpdateActivity 更新玩家活跃时间。
func (r *RankingRepo) UpdateActivity(ctx context.Context, rankingType model.RankingType, playerID uint64) error {
	activeKey := fmt.Sprintf(model.RedisKeyLastActivity, rankingType)
	return r.rdb.HSet(ctx, activeKey, fmt.Sprintf("%d", playerID), time.Now().Unix()).Err()
}

// ---- 工具函数 ----

func parseUint64(s string) (uint64, error) {
	var id uint64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}

func parseUnix(s string) (int64, error) {
	var t int64
	_, err := fmt.Sscanf(s, "%d", &t)
	return t, err
}
