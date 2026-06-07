package model

// ArenaPlayer 竞技场玩家数据
type ArenaPlayer struct {
	PlayerID       string `json:"player_id"`
	Score          int    `json:"score"`            // 当前积分
	Rank           string `json:"rank"`             // 段位: bronze/silver/gold/diamond/legend
	Tier           int    `json:"tier"`             // 子段位: 1-3 (1最高, 3最低)
	SeasonWin      int    `json:"season_win"`       // 本赛季胜场
	SeasonLose     int    `json:"season_lose"`      // 本赛季负场
	Streak         int    `json:"streak"`            // 连胜次数(>=0)
	LastSeasonRank string `json:"last_season_rank"` // 上赛季段位
	DailyWinCount  int    `json:"daily_win_count"`  // 今日胜场
	LastDailyDate  string `json:"last_daily_date"`  // 最后记录日期 YYYY-MM-DD
}

// MatchRecord 对战记录
type MatchRecord struct {
	ID           string `json:"id"`
	PlayerA      string `json:"player_a"`
	PlayerB      string `json:"player_b"`
	Winner       string `json:"winner"`        // 胜者ID, "draw"表示平局
	ScoreChangeA int    `json:"score_change_a"` // A的积分变化
	ScoreChangeB int    `json:"score_change_b"` // B的积分变化
	RankA        string `json:"rank_a"`         // 对战时A的段位
	RankB        string `json:"rank_b"`
	TierA        int    `json:"tier_a"`
	TierB        int    `json:"tier_b"`
	PlayerAScore int    `json:"player_a_score"` // 对战时A的积分
	PlayerBScore int    `json:"player_b_score"`
	Rounds       int    `json:"rounds"`          // 回合数
	Timestamp    int64  `json:"timestamp"`       // Unix时间戳
}

// SeasonInfo 赛季信息
type SeasonInfo struct {
	SeasonID  int    `json:"season_id"`
	Name      string `json:"name"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status"` // active / ended
}
