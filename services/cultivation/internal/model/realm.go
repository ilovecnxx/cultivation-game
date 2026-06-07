// Package model 定义修仙游戏核心数据结构
package model

// SubStage 子境界（小层次）定义
type SubStage struct {
	Level        int    `json:"level"`        // 小境界等级（1-4或1-9）
	Name         string `json:"name"`         // 境界名称，如"练气一层"
	RequiredExp  int64  `json:"required_exp"`  // 突破所需修为值
	BaseAttack   int64  `json:"base_attack"`   // 基础攻击力
	BaseDefense  int64  `json:"base_defense"`  // 基础防御力
	BaseHP       int64  `json:"base_hp"`       // 基础生命值
}

// Realm 大境界定义
type Realm struct {
	ID              int         `json:"id"`               // 境界ID（1=练气, 2=筑基, ..., 9=渡劫）
	Name            string      `json:"name"`             // 境界名称
	SubStages       []SubStage  `json:"sub_stages"`       // 子境界列表
	HasTribulation  bool        `json:"has_tribulation"`  // 是否有天劫
	TribulationBaseRate float64 `json:"tribulation_base_rate"` // 天劫基础通过率
	TribulationDamage    float64 `json:"tribulation_damage"`   // 天劫失败损失比例
	ElementBonus    float64     `json:"element_bonus"`     // 元素属性加成比例
	BaseSpeed       float64     `json:"base_speed"`        // 基础修炼速度（修为/分钟）
}
