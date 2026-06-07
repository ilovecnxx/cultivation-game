package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ActivityRepo 活动数据访问
type ActivityRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewActivityRepo 创建 ActivityRepo
func NewActivityRepo(db *sql.DB, log *zap.Logger) *ActivityRepo {
	return &ActivityRepo{db: db, log: log}
}

// ============================================================
// 限时活动
// ============================================================

// GetActiveEvents 获取当前进行中的活动
func (r *ActivityRepo) GetActiveEvents() ([]*model.LimitedEvent, error) {
	now := time.Now()
	query := `SELECT id, name, type, description, start_time, end_time, min_realm, created_at, updated_at
		FROM limited_events WHERE start_time <= ? AND end_time >= ? ORDER BY start_time ASC`
	rows, err := r.db.Query(query, now, now)
	if err != nil {
		return nil, fmt.Errorf("查询活动列表失败: %w", err)
	}
	defer rows.Close()

	var events []*model.LimitedEvent
	for rows.Next() {
		e := &model.LimitedEvent{}
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Description, &e.StartTime, &e.EndTime, &e.MinRealm, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描活动记录失败: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetEventByID 根据ID获取活动
func (r *ActivityRepo) GetEventByID(eventID string) (*model.LimitedEvent, error) {
	query := `SELECT id, name, type, description, start_time, end_time, min_realm, created_at, updated_at
		FROM limited_events WHERE id = ?`
	e := &model.LimitedEvent{}
	err := r.db.QueryRow(query, eventID).Scan(&e.ID, &e.Name, &e.Type, &e.Description, &e.StartTime, &e.EndTime, &e.MinRealm, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询活动失败: %w", err)
	}
	return e, nil
}

// GetEventRewards 获取活动奖励
func (r *ActivityRepo) GetEventRewards(eventID string) ([]*model.EventReward, error) {
	query := `SELECT id, event_id, item_id, item_name, quantity, probability, is_guaranteed FROM event_rewards WHERE event_id = ?`
	rows, err := r.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("查询活动奖励失败: %w", err)
	}
	defer rows.Close()

	var rewards []*model.EventReward
	for rows.Next() {
		rw := &model.EventReward{}
		if err := rows.Scan(&rw.ID, &rw.EventID, &rw.ItemID, &rw.ItemName, &rw.Quantity, &rw.Probability, &rw.IsGuaranteed); err != nil {
			return nil, fmt.Errorf("扫描活动奖励失败: %w", err)
		}
		rewards = append(rewards, rw)
	}
	return rewards, rows.Err()
}

// GetEventConditions 获取活动条件
func (r *ActivityRepo) GetEventConditions(eventID string) ([]*model.EventCondition, error) {
	query := `SELECT id, event_id, type, target, progress, priority FROM event_conditions WHERE event_id = ? ORDER BY priority ASC`
	rows, err := r.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("查询活动条件失败: %w", err)
	}
	defer rows.Close()

	var conds []*model.EventCondition
	for rows.Next() {
		c := &model.EventCondition{}
		if err := rows.Scan(&c.ID, &c.EventID, &c.Type, &c.Target, &c.Progress, &c.Priority); err != nil {
			return nil, fmt.Errorf("扫描活动条件失败: %w", err)
		}
		conds = append(conds, c)
	}
	return conds, rows.Err()
}

// GetEventProgress 获取玩家活动进度
func (r *ActivityRepo) GetEventProgress(playerID int64, eventID string) (*model.EventProgress, error) {
	query := `SELECT id, player_id, event_id, progress, claimed, created_at, updated_at
		FROM event_progress WHERE player_id = ? AND event_id = ?`
	ep := &model.EventProgress{}
	err := r.db.QueryRow(query, playerID, eventID).Scan(&ep.ID, &ep.PlayerID, &ep.EventID, &ep.Progress, &ep.Claimed, &ep.CreatedAt, &ep.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询活动进度失败: %w", err)
	}
	return ep, nil
}

// UpsertEventProgress 更新活动进度
func (r *ActivityRepo) UpsertEventProgress(ep *model.EventProgress) error {
	query := `INSERT INTO event_progress (id, player_id, event_id, progress, claimed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE progress = VALUES(progress), claimed = VALUES(claimed), updated_at = NOW()`
	_, err := r.db.Exec(query, ep.ID, ep.PlayerID, ep.EventID, ep.Progress, ep.Claimed)
	if err != nil {
		return fmt.Errorf("更新活动进度失败: %w", err)
	}
	return nil
}

// HasClaimedEventReward 检查是否已领取活动奖励
func (r *ActivityRepo) HasClaimedEventReward(playerID int64, eventID string) (bool, error) {
	query := `SELECT COUNT(*) FROM event_reward_records WHERE player_id = ? AND event_id = ?`
	var count int
	err := r.db.QueryRow(query, playerID, eventID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询奖励领取记录失败: %w", err)
	}
	return count > 0, nil
}

// ClaimEventReward 记录活动奖励领取
func (r *ActivityRepo) ClaimEventReward(playerID int64, eventID string, rewardID string) error {
	query := `INSERT INTO event_reward_records (id, player_id, event_id, reward_id, claimed_at)
		VALUES (?, ?, ?, ?, NOW())`
	_, err := r.db.Exec(query, fmt.Sprintf("%d_%s_%s", playerID, eventID, rewardID), playerID, eventID, rewardID)
	return err
}

// ============================================================
// 战令系统
// ============================================================

// GetActiveSeason 获取当前进行中的赛季
func (r *ActivityRepo) GetActiveSeason() (*model.BattlePassSeason, error) {
	now := time.Now()
	query := `SELECT season_id, season_name, start_time, end_time, premium_cost, created_at, updated_at
		FROM battle_pass_seasons WHERE start_time <= ? AND end_time >= ? ORDER BY start_time DESC LIMIT 1`
	s := &model.BattlePassSeason{}
	err := r.db.QueryRow(query, now, now).Scan(&s.SeasonID, &s.SeasonName, &s.StartTime, &s.EndTime, &s.PremiumCost, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询赛季失败: %w", err)
	}
	return s, nil
}

// GetBPTiers 获取战令等级列表
func (r *ActivityRepo) GetBPTiers(seasonID string) ([]*model.BPTier, error) {
	query := `SELECT id, season_id, level, exp_required, is_premium,
		reward_item_id, reward_name, reward_quantity, reward_type
		FROM bp_tiers WHERE season_id = ? ORDER BY level ASC`
	rows, err := r.db.Query(query, seasonID)
	if err != nil {
		return nil, fmt.Errorf("查询战令等级失败: %w", err)
	}
	defer rows.Close()

	var tiers []*model.BPTier
	for rows.Next() {
		t := &model.BPTier{}
		if err := rows.Scan(&t.ID, &t.SeasonID, &t.Level, &t.ExpRequired, &t.IsPremium,
			&t.RewardItemID, &t.RewardName, &t.RewardQuantity, &t.RewardType); err != nil {
			return nil, fmt.Errorf("扫描战令等级失败: %w", err)
		}
		tiers = append(tiers, t)
	}
	return tiers, rows.Err()
}

// GetBPProgress 获取玩家战令进度
func (r *ActivityRepo) GetBPProgress(playerID int64, seasonID string) (*model.BPProgress, error) {
	query := `SELECT id, player_id, season_id, current_level, current_exp, has_premium, claimed_levels, created_at, updated_at
		FROM bp_progress WHERE player_id = ? AND season_id = ?`
	p := &model.BPProgress{}
	err := r.db.QueryRow(query, playerID, seasonID).Scan(&p.ID, &p.PlayerID, &p.SeasonID, &p.CurrentLevel, &p.CurrentExp, &p.HasPremium, &p.ClaimedLevels, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询战令进度失败: %w", err)
	}
	return p, nil
}

// UpsertBPProgress 更新战令进度
func (r *ActivityRepo) UpsertBPProgress(p *model.BPProgress) error {
	query := `INSERT INTO bp_progress (id, player_id, season_id, current_level, current_exp, has_premium, claimed_levels, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE current_level = VALUES(current_level), current_exp = VALUES(current_exp),
		has_premium = VALUES(has_premium), claimed_levels = VALUES(claimed_levels), updated_at = NOW()`
	_, err := r.db.Exec(query, p.ID, p.PlayerID, p.SeasonID, p.CurrentLevel, p.CurrentExp, p.HasPremium, p.ClaimedLevels)
	if err != nil {
		return fmt.Errorf("更新战令进度失败: %w", err)
	}
	return nil
}

// HasClaimedBPReward 检查战令奖励是否已领取
func (r *ActivityRepo) HasClaimedBPReward(playerID int64, seasonID string, level int) (bool, error) {
	query := `SELECT COUNT(*) FROM bp_reward_claim_log WHERE player_id = ? AND season_id = ? AND level = ?`
	var count int
	err := r.db.QueryRow(query, playerID, seasonID, level).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询战令领取记录失败: %w", err)
	}
	return count > 0, nil
}

// ClaimBPReward 记录战令奖励领取
func (r *ActivityRepo) ClaimBPReward(playerID int64, seasonID string, level int) error {
	query := `INSERT INTO bp_reward_claim_log (id, player_id, season_id, level, claimed_at)
		VALUES (?, ?, ?, ?, NOW())`
	id := fmt.Sprintf("%d_%s_%d", playerID, seasonID, level)
	_, err := r.db.Exec(query, id, playerID, seasonID, level)
	return err
}

// ClearBPSeason 清除过期赛季数据
func (r *ActivityRepo) ClearBPSeason(seasonID string) error {
	_, err := r.db.Exec(`DELETE FROM bp_reward_claim_log WHERE season_id = ?`, seasonID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`DELETE FROM bp_progress WHERE season_id = ?`, seasonID)
	return err
}

// ============================================================
// 签到增强
// ============================================================

// GetMonthlyCheckinDays 获取玩家本月签到情况
func (r *ActivityRepo) GetMonthlyCheckinDays(playerID int64, monthStr string) ([]int, int, error) {
	query := `SELECT checkin_day FROM monthly_checkin WHERE player_id = ? AND month_str = ?`
	rows, err := r.db.Query(query, playerID, monthStr)
	if err != nil {
		return nil, 0, fmt.Errorf("查询月签到记录失败: %w", err)
	}
	defer rows.Close()

	var days []int
	var makeupCount int
	for rows.Next() {
		var checkinDay int
		if err := rows.Scan(&checkinDay); err != nil {
			return nil, 0, err
		}
		days = append(days, checkinDay)
	}
	// 查询补签次数
	err = r.db.QueryRow(`SELECT COUNT(*) FROM monthly_checkin WHERE player_id = ? AND month_str = ? AND is_makeup = TRUE`, playerID, monthStr).Scan(&makeupCount)
	if err != nil {
		return nil, 0, err
	}
	return days, makeupCount, rows.Err()
}

// InsertMonthlyCheckin 插入月签到记录
func (r *ActivityRepo) InsertMonthlyCheckin(playerID int64, monthStr string, day int, isMakeup bool) error {
	query := `INSERT IGNORE INTO monthly_checkin (id, player_id, month_str, checkin_day, is_makeup, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())`
	id := fmt.Sprintf("%d_%s_%d", playerID, monthStr, day)
	_, err := r.db.Exec(query, id, playerID, monthStr, day, isMakeup)
	return err
}

// HasMilestoneClaimed 检查里程碑奖励是否已领取
func (r *ActivityRepo) HasMilestoneClaimed(playerID int64, monthStr string, milestoneDay int) (bool, error) {
	query := `SELECT COUNT(*) FROM monthly_checkin_milestones WHERE player_id = ? AND month_str = ? AND milestone_day = ?`
	var count int
	err := r.db.QueryRow(query, playerID, monthStr, milestoneDay).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ClaimMilestone 领取里程碑奖励
func (r *ActivityRepo) ClaimMilestone(playerID int64, monthStr string, milestoneDay int) error {
	query := `INSERT INTO monthly_checkin_milestones (id, player_id, month_str, milestone_day, claimed_at)
		VALUES (?, ?, ?, ?, NOW())`
	id := fmt.Sprintf("%d_%s_%d", playerID, monthStr, milestoneDay)
	_, err := r.db.Exec(query, id, playerID, monthStr, milestoneDay)
	return err
}

// GetStreakBonus 获取连续签到倍率
func (r *ActivityRepo) GetStreakBonus(consecutiveDays int32) float64 {
	switch {
	case consecutiveDays >= 30:
		return 2.0
	case consecutiveDays >= 21:
		return 1.75
	case consecutiveDays >= 14:
		return 1.5
	case consecutiveDays >= 7:
		return 1.25
	default:
		return 1.0
	}
}

// ============================================================
// 成就系统增强
// ============================================================

// GetAllAchievements 获取所有成就定义
func (r *ActivityRepo) GetAllAchievements() ([]*model.AchievementReq, error) {
	query := `SELECT id, category, name, description, is_hidden, hint, icon, sort_order, created_at, updated_at
		FROM achievements ORDER BY sort_order ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询成就列表失败: %w", err)
	}
	defer rows.Close()

	var list []*model.AchievementReq
	for rows.Next() {
		a := &model.AchievementReq{}
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Description, &a.IsHidden, &a.Hint, &a.Icon, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描成就记录失败: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetAchievementByID 获取成就定义
func (r *ActivityRepo) GetAchievementByID(id string) (*model.AchievementReq, error) {
	query := `SELECT id, category, name, description, is_hidden, hint, icon, sort_order, created_at, updated_at
		FROM achievements WHERE id = ?`
	a := &model.AchievementReq{}
	err := r.db.QueryRow(query, id).Scan(&a.ID, &a.Category, &a.Name, &a.Description, &a.IsHidden, &a.Hint, &a.Icon, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询成就失败: %w", err)
	}
	return a, nil
}

// GetAchievementTiers 获取成就等级
func (r *ActivityRepo) GetAchievementTiers(achievementID string) ([]*model.AchievementTier, error) {
	query := `SELECT id, achievement_id, level, name, condition, title_id, reward_exp, reward_money
		FROM achievement_tiers WHERE achievement_id = ? ORDER BY level ASC`
	rows, err := r.db.Query(query, achievementID)
	if err != nil {
		return nil, fmt.Errorf("查询成就等级失败: %w", err)
	}
	defer rows.Close()

	var tiers []*model.AchievementTier
	for rows.Next() {
		t := &model.AchievementTier{}
		if err := rows.Scan(&t.ID, &t.AchievementID, &t.Level, &t.Name, &t.Condition, &t.TitleID, &t.RewardExp, &t.RewardMoney); err != nil {
			return nil, fmt.Errorf("扫描成就等级失败: %w", err)
		}
		tiers = append(tiers, t)
	}
	return tiers, rows.Err()
}

// GetPlayerAchievement 获取玩家成就进度
func (r *ActivityRepo) GetPlayerAchievement(playerID int64, achievementID string) (*model.PlayerAchievementTier, error) {
	query := `SELECT player_id, achievement_id, current_tier, progress, completed, claimed_tiers, completed_at, updated_at
		FROM player_achievements_tiers WHERE player_id = ? AND achievement_id = ?`
	pa := &model.PlayerAchievementTier{}
	err := r.db.QueryRow(query, playerID, achievementID).Scan(&pa.PlayerID, &pa.AchievementID, &pa.CurrentTier, &pa.Progress, &pa.Completed, &pa.ClaimedTiers, &pa.CompletedAt, &pa.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询玩家成就失败: %w", err)
	}
	return pa, nil
}

// UpsertPlayerAchievement 更新玩家成就进度
func (r *ActivityRepo) UpsertPlayerAchievement(pa *model.PlayerAchievementTier) error {
	query := `INSERT INTO player_achievements_tiers (player_id, achievement_id, current_tier, progress, completed, claimed_tiers, completed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE current_tier = VALUES(current_tier), progress = VALUES(progress),
		completed = VALUES(completed), claimed_tiers = VALUES(claimed_tiers), completed_at = VALUES(completed_at), updated_at = NOW()`
	_, err := r.db.Exec(query, pa.PlayerID, pa.AchievementID, pa.CurrentTier, pa.Progress, pa.Completed, pa.ClaimedTiers, pa.CompletedAt)
	return err
}

// BatchInitAchievements 初始化玩家成就
func (r *ActivityRepo) BatchInitAchievements(playerID int64, achievementIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	for _, aid := range achievementIDs {
		_, err := tx.Exec(`INSERT IGNORE INTO player_achievements_tiers (player_id, achievement_id, current_tier, progress, completed, claimed_tiers, completed_at, updated_at)
			VALUES (?, ?, 0, 0, FALSE, '', ?, NOW())`, playerID, aid, now)
		if err != nil {
			return fmt.Errorf("初始化成就记录失败 player=%d achievement=%s: %w", playerID, aid, err)
		}
	}
	return tx.Commit()
}

// ============================================================
// 称号系统
// ============================================================

// GetAllTitles 获取所有称号定义
func (r *ActivityRepo) GetAllTitles() ([]*model.Title, error) {
	query := `SELECT id, name, description, color, source, stat_bonus_hp, stat_bonus_attack, stat_bonus_defense, stat_bonus_speed, rarity, created_at, updated_at
		FROM titles ORDER BY rarity ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询称号列表失败: %w", err)
	}
	defer rows.Close()

	var titles []*model.Title
	for rows.Next() {
		t := &model.Title{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Color, &t.Source,
			&t.StatBonusHP, &t.StatBonusAttack, &t.StatBonusDefense, &t.StatBonusSpeed, &t.Rarity,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描称号记录失败: %w", err)
		}
		titles = append(titles, t)
	}
	return titles, rows.Err()
}

// GetPlayerTitles 获取玩家已获得的称号
func (r *ActivityRepo) GetPlayerTitles(playerID int64) ([]*model.PlayerTitleEnhanced, error) {
	query := `SELECT id, player_id, title_id, is_equipped, obtained_at
		FROM player_titles_enhanced WHERE player_id = ? ORDER BY obtained_at DESC`
	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家称号失败: %w", err)
	}
	defer rows.Close()

	var pts []*model.PlayerTitleEnhanced
	for rows.Next() {
		pt := &model.PlayerTitleEnhanced{}
		if err := rows.Scan(&pt.ID, &pt.PlayerID, &pt.TitleID, &pt.IsEquipped, &pt.ObtainedAt); err != nil {
			return nil, fmt.Errorf("扫描玩家称号失败: %w", err)
		}
		pts = append(pts, pt)
	}
	return pts, rows.Err()
}

// AddPlayerTitle 添加玩家称号
func (r *ActivityRepo) AddPlayerTitle(playerID int64, titleID string) error {
	id := fmt.Sprintf("%d_%s", playerID, titleID)
	query := `INSERT IGNORE INTO player_titles_enhanced (id, player_id, title_id, is_equipped, obtained_at)
		VALUES (?, ?, ?, FALSE, NOW())`
	_, err := r.db.Exec(query, id, playerID, titleID)
	return err
}

// EquipPlayerTitle 装备/卸下称号
func (r *ActivityRepo) EquipPlayerTitle(playerID int64, titleID string, equip bool) error {
	// 首先取消所有称号的装备状态
	if equip {
		_, err := r.db.Exec(`UPDATE player_titles_enhanced SET is_equipped = FALSE WHERE player_id = ?`, playerID)
		if err != nil {
			return fmt.Errorf("取消称号装备失败: %w", err)
		}
	}
	_, err := r.db.Exec(`UPDATE player_titles_enhanced SET is_equipped = ? WHERE player_id = ? AND title_id = ?`, equip, playerID, titleID)
	return err
}

// GetEquippedTitle 获取玩家当前装备的称号
func (r *ActivityRepo) GetEquippedTitle(playerID int64) (*model.PlayerTitleEnhanced, error) {
	query := `SELECT id, player_id, title_id, is_equipped, obtained_at
		FROM player_titles_enhanced WHERE player_id = ? AND is_equipped = TRUE LIMIT 1`
	pt := &model.PlayerTitleEnhanced{}
	err := r.db.QueryRow(query, playerID).Scan(&pt.ID, &pt.PlayerID, &pt.TitleID, &pt.IsEquipped, &pt.ObtainedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询装备称号失败: %w", err)
	}
	return pt, nil
}

// GetTitleByID 获取称号定义
func (r *ActivityRepo) GetTitleByID(titleID string) (*model.Title, error) {
	query := `SELECT id, name, description, color, source, stat_bonus_hp, stat_bonus_attack, stat_bonus_defense, stat_bonus_speed, rarity, created_at, updated_at
		FROM titles WHERE id = ?`
	t := &model.Title{}
	err := r.db.QueryRow(query, titleID).Scan(&t.ID, &t.Name, &t.Description, &t.Color, &t.Source,
		&t.StatBonusHP, &t.StatBonusAttack, &t.StatBonusDefense, &t.StatBonusSpeed, &t.Rarity,
		&t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询称号定义失败: %w", err)
	}
	return t, nil
}

// GetTitlesByIDs 批量获取称号定义
func (r *ActivityRepo) GetTitlesByIDs(ids []string) (map[string]*model.Title, error) {
	if len(ids) == 0 {
		return make(map[string]*model.Title), nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT id, name, description, color, source, stat_bonus_hp, stat_bonus_attack, stat_bonus_defense, stat_bonus_speed, rarity, created_at, updated_at
		FROM titles WHERE id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询称号失败: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*model.Title)
	for rows.Next() {
		t := &model.Title{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Color, &t.Source,
			&t.StatBonusHP, &t.StatBonusAttack, &t.StatBonusDefense, &t.StatBonusSpeed, &t.Rarity,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描称号记录失败: %w", err)
		}
		result[t.ID] = t
	}
	return result, rows.Err()
}

// GetPlayerAchievementPoints 获取玩家成就点总数
func (r *ActivityRepo) GetPlayerAchievementPoints(playerID int64) (int, error) {
	query := `SELECT COALESCE(SUM(current_tier * 10), 0) FROM player_achievements_tiers WHERE player_id = ?`
	var points int
	err := r.db.QueryRow(query, playerID).Scan(&points)
	if err != nil {
		return 0, err
	}
	return points, nil
}

// ============================================================
// 活动数据序列化工具
// ============================================================

// MarshalClaimedLevels 序列化已领取等级
func MarshalClaimedLevels(levels []int) string {
	if len(levels) == 0 {
		return ""
	}
	parts := make([]string, len(levels))
	for i, l := range levels {
		parts[i] = fmt.Sprintf("%d", l)
	}
	return strings.Join(parts, ",")
}

// UnmarshalClaimedLevels 反序列化已领取等级
func UnmarshalClaimedLevels(data string) []int {
	if data == "" {
		return nil
	}
	parts := strings.Split(data, ",")
	levels := make([]int, 0, len(parts))
	for _, p := range parts {
		var l int
		if _, err := fmt.Sscanf(p, "%d", &l); err == nil {
			levels = append(levels, l)
		}
	}
	return levels
}

// MarshalEventRewards 序列化事件奖励
func MarshalEventRewards(rewards []*model.EventReward) string {
	data, _ := json.Marshal(rewards)
	return string(data)
}

// UnmarshalEventRewards 反序列化事件奖励
func UnmarshalEventRewards(data string) []*model.EventReward {
	if data == "" {
		return nil
	}
	var rewards []*model.EventReward
	json.Unmarshal([]byte(data), &rewards)
	return rewards
}
