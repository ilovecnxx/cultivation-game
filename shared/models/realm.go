package models

// RealmConfig 境界配置，定义一个大境界（如练气、筑基）的完整属性。
type RealmConfig struct {
	ID              uint32 `json:"id"`                // 境界ID
	Name            string `json:"name"`              // 境界名称
	LevelCount      uint32 `json:"level_count"`       // 层数（通常9层）
	BaseExpPerLevel uint64 `json:"base_exp_per_level"` // 每层基础所需经验
	ExpMultiplier   float64 `json:"exp_multiplier"`   // 层数经验倍率（每层累乘）
	BaseHealth      int64  `json:"base_health"`       // 基础气血加成
	BaseMana        int64  `json:"base_mana"`         // 基础灵力加成
	BaseAttack      int64  `json:"base_attack"`       // 基础攻击加成
	BaseDefense     int64  `json:"base_defense"`      // 基础防御加成
	NextRealmID     uint32 `json:"next_realm_id"`     // 下一境界ID（0表示最高境界）
	Icon            string `json:"icon"`              // 境界图标路径
	Description     string `json:"description"`       // 境界描述
}

// ExpForLevel 计算指定层数所需的累计经验值。
func (r *RealmConfig) ExpForLevel(level uint32) uint64 {
	if level == 0 || level > r.LevelCount {
		return 0
	}
	// 经验值 = 基础值 * (倍率 ^ (层数-1))
	multiplier := 1.0
	for i := uint32(1); i < level; i++ {
		multiplier *= r.ExpMultiplier
	}
	return uint64(float64(r.BaseExpPerLevel) * multiplier)
}

// BreakthroughConfig 大境界突破配置。
// 突破是从当前大境界的顶层晋升到下一个大境界的底层。
type BreakthroughConfig struct {
	FromRealmID uint32  `json:"from_realm_id"` // 当前境界ID
	ToRealmID   uint32  `json:"to_realm_id"`   // 目标境界ID
	BaseRate    float64 `json:"base_rate"`      // 基础成功率（0.0-1.0）
	// 成功率修正因素
	SpiritRootBonus   map[string]float64 `json:"spirit_root_bonus"`   // 灵根类型 -> 成功率加成
	ItemBonus         map[uint32]float64 `json:"item_bonus"`          // 辅助道具ID -> 成功率加成
	MaxRate           float64            `json:"max_rate"`             // 最大成功率上限
	FailedDropLevels  uint32             `json:"failed_drop_levels"`  // 失败后掉落的层数（如 1 = 掉1层）
	FailedCooldownSec uint32             `json:"failed_cooldown_sec"` // 失败后冷却时间（秒）
}

// RealmProgress 玩家的境界进度汇总。
type RealmProgress struct {
	PlayerID      uint64 `json:"player_id"`
	CurrentRealm  uint32 `json:"current_realm"`   // 当前境界ID
	CurrentLevel  uint32 `json:"current_level"`   // 当前层数
	CurrentExp    uint64 `json:"current_exp"`     // 当前经验值
	Breakthroughs uint32 `json:"breakthroughs"`   // 累计突破次数
	FailedCount   uint32 `json:"failed_count"`    // 累计突破失败次数
}

// ExpToNextLevel 返回升到下一层所需经验值；若已满层则返回 0。
func (rp *RealmProgress) ExpToNextLevel(cfg *RealmConfig) uint64 {
	if rp.CurrentLevel >= cfg.LevelCount {
		return 0
	}
	needed := cfg.ExpForLevel(rp.CurrentLevel + 1)
	if rp.CurrentExp >= needed {
		return 0
	}
	return needed - rp.CurrentExp
}
