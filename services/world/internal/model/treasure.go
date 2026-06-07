// Package model 藏宝图系统数据模型
package model

// TreasureFragment 藏宝图碎片
type TreasureFragment struct {
	PlayerID   int64  `json:"player_id"`
	Index      int    `json:"index"`       // 碎片编号 0-3（共4片）
	ObtainedAt int64  `json:"obtained_at"` // 获取时间
}

// TreasureMap 完整藏宝图
type TreasureMap struct {
	PlayerID   int64  `json:"player_id"`
	FragmentBits int   `json:"fragment_bits"` // 已收集的碎片位标记 (bit0-bit3)
	Completed  bool   `json:"completed"`      // 是否已拼合成完整地图
	Digged     bool   `json:"digged"`          // 是否已挖宝
	MapID      string `json:"map_id,omitempty"` // 藏宝图标识
	RegionID     string `json:"region_id,omitempty"`
	X            int    `json:"x,omitempty"`
	Y            int    `json:"y,omitempty"`
	ItemName     string `json:"item_name,omitempty"`
	Rarity       int    `json:"rarity,omitempty"`
}

// TreasureReward 挖宝奖励配置
type TreasureReward struct {
	Type   string `json:"type"`   // 奖励类型: item/stone/technique/artifact
	ID     int64  `json:"id"`     // 物品ID
	Name   string `json:"name"`   // 名称
	Amount int64  `json:"amount"` // 数量
	Weight int    `json:"weight"` // 权重
}

// TreasureRewardPool 挖宝奖励池
var TreasureRewardPool = []TreasureReward{
	{Type: "stone", ID: 1, Name: "大量灵石", Amount: 5000, Weight: 30},
	{Type: "stone", ID: 1, Name: "海量灵石", Amount: 20000, Weight: 10},
	{Type: "item", ID: 301, Name: "灵根进化石", Amount: 1, Weight: 15},
	{Type: "item", ID: 302, Name: "坐骑饲料x10", Amount: 10, Weight: 20},
	{Type: "technique", ID: 401, Name: "功法残卷·天元诀", Amount: 1, Weight: 10},
	{Type: "technique", ID: 402, Name: "功法残卷·九转金身", Amount: 1, Weight: 8},
	{Type: "artifact", ID: 501, Name: "本命法宝碎片·混沌钟", Amount: 1, Weight: 5},
	{Type: "artifact", ID: 502, Name: "本命法宝碎片·诛仙剑", Amount: 1, Weight: 3},
	{Type: "item", ID: 303, Name: "稀有丹药·玄元丹", Amount: 1, Weight: 12},
	{Type: "item", ID: 304, Name: "灵兽蛋·祥瑞麒麟", Amount: 1, Weight: 2},
}

// DrawReward 从奖励池中按权重随机抽取
func DrawReward() *TreasureReward {
	total := 0
	for _, r := range TreasureRewardPool {
		total += r.Weight
	}
	roll := 0
	for i, r := range TreasureRewardPool {
		roll += r.Weight
		if roll >= total/2 { // 简单均匀随机
			return &TreasureRewardPool[i]
		}
	}
	return &TreasureRewardPool[0]
}

// WeatherForecast 天气预告
type WeatherForecast struct {
	Period        string  `json:"period"`
	WeatherType   string  `json:"weather_type"`
	SpiritDensity float64 `json:"spirit_density"`
	Effect        string  `json:"effect"`
}
