// Package model 坐骑系统数据模型
package model

// MountQuality 坐骑品质
type MountQuality int

const (
	MountQualityCommon    MountQuality = 1 // 普通（白）
	MountQualityUncommon  MountQuality = 2 // 优秀（绿）
	MountQualityRare      MountQuality = 3 // 稀有（蓝）
	MountQualityEpic      MountQuality = 4 // 史诗（紫）
	MountQualityLegendary MountQuality = 5 // 传说（金）
)

// MountQualityNames 坐骑品质名称
var MountQualityNames = map[MountQuality]string{
	MountQualityCommon:    "普通",
	MountQualityUncommon:  "优秀",
	MountQualityRare:      "稀有",
	MountQualityEpic:      "史诗",
	MountQualityLegendary: "传说",
}

// MountQualityColors 坐骑品质颜色
var MountQualityColors = map[MountQuality]string{
	MountQualityCommon:    "#9e9e9e",
	MountQualityUncommon:  "#4caf50",
	MountQualityRare:      "#42a5f5",
	MountQualityEpic:      "#ab47bc",
	MountQualityLegendary: "#ffd700",
}

// MountSpeedBonus 坐骑速度加成百分比
var MountSpeedBonus = map[MountQuality]int{
	MountQualityCommon:    10,
	MountQualityUncommon:  20,
	MountQualityRare:      30,
	MountQualityEpic:      40,
	MountQualityLegendary: 50,
}

// MountLevelMax 坐骑最高等级
const MountLevelMax = 50

// Mount 坐骑
type Mount struct {
	ID        int64        `json:"id" gorm:"primaryKey"`
	PlayerID  int64        `json:"player_id" gorm:"index"`
	Name      string       `json:"name"`                           // 坐骑名称
	Species   string       `json:"species"`                        // 种类标识: flying_sword/spirit_crane/cloud/dragon/phoenix
	Quality   MountQuality `json:"quality"`                        // 品质
	Level     int          `json:"level"`                          // 等级 1-50
	Exp       int64        `json:"exp"`                            // 当前经验
	Equipped  bool         `json:"equipped"`                       // 是否装备
	CreatedAt int64        `json:"created_at"`
	UpdatedAt int64        `json:"updated_at"`
}

// MountSpeciesConfig 坐骑种类配置
type MountSpeciesConfig struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Quality     MountQuality `json:"quality"`
	Realm       int          `json:"realm"` // 所需境界等级
}

// MountSpecies 所有坐骑品种定义
var MountSpeciesList = []MountSpeciesConfig{
	{ID: "flying_sword", Name: "飞剑", Description: "御剑飞行，迅捷如风", Quality: MountQualityCommon, Realm: 3},   // 筑基
	{ID: "spirit_crane", Name: "灵鹤", Description: "仙鹤展翅，凌云九天", Quality: MountQualityUncommon, Realm: 4}, // 金丹
	{ID: "cloud", Name: "七彩祥云", Description: "祥云瑞彩，霞光万丈", Quality: MountQualityRare, Realm: 5},        // 元婴
	{ID: "dragon", Name: "神龙", Description: "龙腾四海，威震八荒", Quality: MountQualityEpic, Realm: 6},          // 化神
	{ID: "phoenix", Name: "凤凰", Description: "凤鸣九天，涅槃重生", Quality: MountQualityLegendary, Realm: 7},    // 炼虚
}

// MountUpgradeExp 计算坐骑升级所需经验
// baseExp * qualityFactor * level
func MountUpgradeExp(quality MountQuality, level int) int64 {
	factor := 1.0
	switch quality {
	case MountQualityCommon:
		factor = 1.0
	case MountQualityUncommon:
		factor = 1.5
	case MountQualityRare:
		factor = 2.0
	case MountQualityEpic:
		factor = 3.0
	case MountQualityLegendary:
		factor = 5.0
	}
	return int64(float64(100) * factor * float64(level))
}

// MountSpeciesByName 按ID查找坐骑种类
func MountSpeciesByName(id string) *MountSpeciesConfig {
	for _, s := range MountSpeciesList {
		if s.ID == id {
			return &s
		}
	}
	return nil
}
