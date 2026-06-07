// Package models 定义修仙游戏的核心数据结构，包括玩家、物品、境界和战斗模型。
package models

import "time"

// PlayerAttribute 玩家基础属性。
type PlayerAttribute struct {
	Health     int64 // 气血
	Mana       int64 // 灵力
	Strength   int64 // 力量
	Agility    int64 // 身法
	Spirit     int64 // 神识
	Defense    int64 // 防御
	Critical   int64 // 暴击率（万分比，如 500 = 5%）
	Dodge      int64 // 闪避率（万分比）
}

// Player 玩家实体，对应数据库中的玩家记录。
type Player struct {
	ID            uint64          `json:"id"`            // 玩家唯一ID
	Nickname      string          `json:"nickname"`      // 昵称
	RealmID       uint32          `json:"realm_id"`      // 当前境界ID（如 1=练气, 2=筑基）
	RealmLevel    uint32          `json:"realm_level"`   // 当前境界层数（1-9层）
	Exp           uint64          `json:"exp"`           // 当前修为值
	SpiritRoot    string          `json:"spirit_root"`   // 灵根类型（如 "金灵根", "变异雷灵根"）
	BaseAttr      PlayerAttribute `json:"base_attr"`     // 基础属性（不含装备加成）
	EquipAttr     PlayerAttribute `json:"equip_attr"`    // 装备附加属性
	CreatedAt     time.Time       `json:"created_at"`    // 创建时间
	LastLoginAt   time.Time       `json:"last_login_at"` // 最后登录时间
	LastCultivate time.Time       `json:"last_cultivate"` // 最后修炼时间
}

// TotalAttr 返回玩家总属性（基础 + 装备）。
func (p *Player) TotalAttr() PlayerAttribute {
	return PlayerAttribute{
		Health:   p.BaseAttr.Health + p.EquipAttr.Health,
		Mana:     p.BaseAttr.Mana + p.EquipAttr.Mana,
		Strength: p.BaseAttr.Strength + p.EquipAttr.Strength,
		Agility:  p.BaseAttr.Agility + p.EquipAttr.Agility,
		Spirit:   p.BaseAttr.Spirit + p.EquipAttr.Spirit,
		Defense:  p.BaseAttr.Defense + p.EquipAttr.Defense,
		Critical: p.BaseAttr.Critical + p.EquipAttr.Critical,
		Dodge:    p.BaseAttr.Dodge + p.EquipAttr.Dodge,
	}
}

// CanBreakthrough 判断当前是否满足突破条件（经验值达到 100% 且层数未满）。
func (p *Player) CanBreakthrough(realmCfg *RealmConfig) bool {
	levelMax := int(realmCfg.LevelCount)
	if int(p.RealmLevel) >= levelMax {
		return false // 已达当前境界最高层，需突破大境界
	}
	// 每层所需经验 = 基础值 * 层数系数
	neededExp := realmCfg.BaseExpPerLevel * uint64(p.RealmLevel)
	return p.Exp >= neededExp
}
