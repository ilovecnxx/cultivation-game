package model

// Dungeon 秘境副本配置
type Dungeon struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	RealmReq   int     `json:"realm_req"`   // 需要境界ID
	Floors     []Floor `json:"floors"`
	DailyLimit int     `json:"daily_limit"` // 每日限制次数
	EntryFee   int64   `json:"entry_fee"`   // 入场费(灵石)
}

// Floor 秘境层
type Floor struct {
	Level    int              `json:"level"`
	Monsters []DungeonMonster `json:"monsters"` // 普通怪物(可能多个)
	Boss     DungeonMonster   `json:"boss"`     // 本层Boss
	Rewards  FloorReward      `json:"rewards"`  // 通关本层奖励
}

// DungeonMonster 秘境怪物
type DungeonMonster struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	HP       int64   `json:"hp"`
	Atk      int64   `json:"atk"`
	Def      int64   `json:"def"`
	Skills   []int   `json:"skills"`    // 技能ID列表
	DropRate float64 `json:"drop_rate"` // 额外掉落率(0~1)
}

// FloorReward 层奖励
type FloorReward struct {
	Exp   int64 `json:"exp"`   // 修为
	Money int64 `json:"money"` // 灵石
	Items []int `json:"items"` // 可能掉落物品ID列表(随机掉落其中一部分)
}

// DungeonSession 玩家秘境进度(运行时)
type DungeonSession struct {
	PlayerID       string   `json:"player_id"`
	DungeonID      int      `json:"dungeon_id"`
	CurrentFloor   int      `json:"current_floor"`   // 当前所在层(从1开始)
	ClearedFloors  []int    `json:"cleared_floors"`   // 已通关层数列表
	Team           []string `json:"team"`             // 组队成员ID列表(含队长)
	EnteredAt      int64    `json:"entered_at"`       // 进入时间(unix时间戳)
	Completed      bool     `json:"completed"`        // 是否已通关全部层
	RewardsClaimed bool     `json:"rewards_claimed"`  // 是否已领取通关奖励
}

// DungeonDailyRecord 每日挑战记录
type DungeonDailyRecord struct {
	PlayerID  string `json:"player_id"`
	DungeonID int    `json:"dungeon_id"`
	Count     int    `json:"count"`     // 今日已挑战次数
	Date      string `json:"date"`      // 记录日期 YYYY-MM-DD
}
