// Package model 黑市系统数据模型
package model

// ============================================================================
// 物品类型常量
// ============================================================================

const (
	ItemTypePill            = "pill"
	ItemTypeArtifactFragment = "artifact_fragment"
	ItemTypePetEgg          = "pet_egg"
	ItemTypeTechnique       = "technique"
	ItemTypeMaterial        = "material"
	ItemTypeEquipment       = "equipment"
	ItemTypeTreasure        = "treasure"
	ItemTypeMysteryBox      = "mystery_box"
)

// ============================================================================
// 事件系统
// ============================================================================

// BlackMarketEventType 特殊事件类型
type BlackMarketEventType string

const (
	EventDoubleStock BlackMarketEventType = "double_stock" // 双倍库存
	EventAllDiscount  BlackMarketEventType = "all_discount" // 全场折扣
	EventRareItem    BlackMarketEventType = "rare_item"    // 稀有物品出现
)

// BlackMarketEvent 黑市特殊事件
type BlackMarketEvent struct {
	Type        BlackMarketEventType `json:"type"`
	Description string              `json:"description"`
	Discount    int                 `json:"discount,omitempty"`     // 额外折扣百分比（all_discount）
	ExtraItems  []BlackMarketItem   `json:"extra_items,omitempty"` // 额外稀有物品（rare_item）
}

// ============================================================================
// 基础模型
// ============================================================================

// BlackMarketItem 黑市物品
type BlackMarketItem struct {
	ID            string `json:"id" bson:"_id"`
	ItemID        int64  `json:"item_id"`         // 物品ID
	Name          string `json:"name"`             // 物品名称
	Type          string `json:"type"`             // 物品类型
	Description   string `json:"description"`      // 描述
	PriceStone    int64  `json:"price_stone"`      // 灵石价格（随机浮动后）
	PriceJade     int64  `json:"price_jade"`       // 仙玉价格（随机浮动后）
	Stock         int    `json:"stock"`             // 库存
	OriginalPrice int64  `json:"original_price"`   // 原价（灵石）
	Discount      int    `json:"discount"`          // 折扣百分比 70-150（<100=打折 >100=溢价）
	ExpiresAt     int64  `json:"expires_at"`        // 过期时间
	MaxPerPlayer  int    `json:"max_per_player,omitempty"` // 每位玩家限购数量（0=不限）
	Rarity        string `json:"rarity,omitempty"`  // 稀有度: common/rare/epic/legendary
}

// BlackMarketPurchaseRecord 玩家购买记录
type BlackMarketPurchaseRecord struct {
	PlayerID   uint64 `json:"player_id"`
	ItemBMID   string `json:"item_bm_id"`   // 黑市物品ID (bm_xxx)
	ItemID     int64  `json:"item_id"`       // 模板物品ID
	ItemName   string `json:"item_name"`
	ItemType   string `json:"item_type"`
	Quantity   int    `json:"quantity"`
	TotalStone int64  `json:"total_stone"`
	TotalJade  int64  `json:"total_jade"`
	Discount   int    `json:"discount"`    // 随机折扣
	VipBonus   int    `json:"vip_bonus"`   // VIP额外折扣
	Timestamp  int64  `json:"timestamp"`
}

// ============================================================================
// 刷新配置
// ============================================================================

// BlackMarketRefreshTimes 每日刷新时间点（小时），每6小时刷新一次
var BlackMarketRefreshTimes = []int{0, 6, 12, 18}

// ============================================================================
// 物品池
// ============================================================================

// BlackMarketItemPool 黑市物品池条目
type BlackMarketItemPool struct {
	ItemID         int64  `json:"item_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	BasePriceStone int64  `json:"base_price_stone"` // 基础灵石价格
	BasePriceJade  int64  `json:"base_price_jade"`  // 基础仙玉价格
	MinStock       int    `json:"min_stock"`
	MaxStock       int    `json:"max_stock"`
	Weight         int    `json:"weight"`           // 出现权重
	MaxPerPlayer   int    `json:"max_per_player"`   // 每位玩家限购（0=不限）
	Rarity         string `json:"rarity"`           // 稀有度
}

// BlackMarketPool 黑市常规物品池
var BlackMarketPool = []BlackMarketItemPool{
	// ---- 丹药 (pill) ----
	{ItemID: 1001, Name: "凝气丹", Type: ItemTypePill, Description: "提升修炼速度(小)", BasePriceStone: 500, BasePriceJade: 0, MinStock: 5, MaxStock: 20, Weight: 20, MaxPerPlayer: 0, Rarity: "common"},
	{ItemID: 1002, Name: "筑基丹", Type: ItemTypePill, Description: "筑基突破辅助丹药", BasePriceStone: 2000, BasePriceJade: 0, MinStock: 1, MaxStock: 5, Weight: 15, MaxPerPlayer: 3, Rarity: "rare"},
	{ItemID: 1003, Name: "玄元丹", Type: ItemTypePill, Description: "大量提升修为", BasePriceStone: 5000, BasePriceJade: 50, MinStock: 1, MaxStock: 3, Weight: 10, MaxPerPlayer: 2, Rarity: "epic"},
	{ItemID: 1004, Name: "洗髓丹", Type: ItemTypePill, Description: "洗髓伐脉，提升根骨", BasePriceStone: 8000, BasePriceJade: 80, MinStock: 1, MaxStock: 2, Weight: 5, MaxPerPlayer: 1, Rarity: "epic"},

	// ---- 法宝碎片 (artifact_fragment) ----
	{ItemID: 2001, Name: "法宝碎片·紫绶仙衣", Type: ItemTypeArtifactFragment, Description: "凑齐5片可合成", BasePriceStone: 3000, BasePriceJade: 30, MinStock: 1, MaxStock: 3, Weight: 12, MaxPerPlayer: 5, Rarity: "rare"},
	{ItemID: 2002, Name: "法宝碎片·番天印", Type: ItemTypeArtifactFragment, Description: "凑齐5片可合成", BasePriceStone: 3500, BasePriceJade: 40, MinStock: 1, MaxStock: 3, Weight: 10, MaxPerPlayer: 5, Rarity: "rare"},
	{ItemID: 2003, Name: "法宝碎片·捆仙绳", Type: ItemTypeArtifactFragment, Description: "凑齐5片可合成", BasePriceStone: 2800, BasePriceJade: 25, MinStock: 1, MaxStock: 3, Weight: 10, MaxPerPlayer: 5, Rarity: "rare"},

	// ---- 灵兽蛋 (pet_egg) ----
	{ItemID: 3001, Name: "灵兽蛋·霜月狼", Type: ItemTypePetEgg, Description: "孵化获得稀有灵兽", BasePriceStone: 8000, BasePriceJade: 100, MinStock: 1, MaxStock: 2, Weight: 5, MaxPerPlayer: 1, Rarity: "epic"},
	{ItemID: 3002, Name: "灵兽蛋·赤炎虎", Type: ItemTypePetEgg, Description: "孵化获得稀有灵兽", BasePriceStone: 10000, BasePriceJade: 120, MinStock: 1, MaxStock: 1, Weight: 3, MaxPerPlayer: 1, Rarity: "legendary"},

	// ---- 功法残卷 (technique) ----
	{ItemID: 4001, Name: "功法残卷·混元诀", Type: ItemTypeTechnique, Description: "记载了上古功法的一页", BasePriceStone: 6000, BasePriceJade: 60, MinStock: 1, MaxStock: 2, Weight: 8, MaxPerPlayer: 1, Rarity: "epic"},
	{ItemID: 4002, Name: "功法残卷·太虚剑经", Type: ItemTypeTechnique, Description: "记载了上古剑法的一页", BasePriceStone: 7000, BasePriceJade: 70, MinStock: 1, MaxStock: 2, Weight: 7, MaxPerPlayer: 1, Rarity: "epic"},

	// ---- 材料 (material) ----
	{ItemID: 6001, Name: "千年玄铁", Type: ItemTypeMaterial, Description: "锻造神器的上等材料", BasePriceStone: 1500, BasePriceJade: 0, MinStock: 3, MaxStock: 10, Weight: 18, MaxPerPlayer: 0, Rarity: "common"},
	{ItemID: 6002, Name: "天蚕丝", Type: ItemTypeMaterial, Description: "编织法衣的稀有材料", BasePriceStone: 2500, BasePriceJade: 20, MinStock: 2, MaxStock: 8, Weight: 14, MaxPerPlayer: 0, Rarity: "rare"},
	{ItemID: 6003, Name: "星辰砂", Type: ItemTypeMaterial, Description: "蕴含星辰之力的珍稀材料", BasePriceStone: 4500, BasePriceJade: 40, MinStock: 1, MaxStock: 4, Weight: 8, MaxPerPlayer: 3, Rarity: "epic"},

	// ---- 装备 (equipment) ----
	{ItemID: 7001, Name: "玄铁重剑", Type: ItemTypeEquipment, Description: "重剑无锋，大巧不工", BasePriceStone: 3500, BasePriceJade: 0, MinStock: 1, MaxStock: 3, Weight: 12, MaxPerPlayer: 1, Rarity: "rare"},
	{ItemID: 7002, Name: "金蚕丝甲", Type: ItemTypeEquipment, Description: "刀枪不入的宝甲", BasePriceStone: 4500, BasePriceJade: 30, MinStock: 1, MaxStock: 2, Weight: 10, MaxPerPlayer: 1, Rarity: "rare"},
	{ItemID: 7003, Name: "凌云靴", Type: ItemTypeEquipment, Description: "日行千里的宝靴", BasePriceStone: 3000, BasePriceJade: 20, MinStock: 1, MaxStock: 2, Weight: 10, MaxPerPlayer: 1, Rarity: "rare"},

	// ---- 宝物 (treasure) ----
	{ItemID: 8001, Name: "乾坤袋", Type: ItemTypeTreasure, Description: "内有乾坤，储物无限", BasePriceStone: 10000, BasePriceJade: 150, MinStock: 1, MaxStock: 1, Weight: 3, MaxPerPlayer: 1, Rarity: "legendary"},
	{ItemID: 8002, Name: "聚宝盆", Type: ItemTypeTreasure, Description: "每日产出灵石", BasePriceStone: 12000, BasePriceJade: 200, MinStock: 1, MaxStock: 1, Weight: 2, MaxPerPlayer: 1, Rarity: "legendary"},

	// ---- 神秘宝箱 (mystery_box) ----
	{ItemID: 9001, Name: "神秘宝箱·铜", Type: ItemTypeMysteryBox, Description: "打开获得随机道具", BasePriceStone: 1000, BasePriceJade: 10, MinStock: 2, MaxStock: 5, Weight: 18, MaxPerPlayer: 0, Rarity: "common"},
	{ItemID: 9002, Name: "神秘宝箱·银", Type: ItemTypeMysteryBox, Description: "打开获得稀有道具", BasePriceStone: 3000, BasePriceJade: 30, MinStock: 1, MaxStock: 3, Weight: 12, MaxPerPlayer: 3, Rarity: "rare"},
	{ItemID: 9003, Name: "神秘宝箱·金", Type: ItemTypeMysteryBox, Description: "打开获得极品道具", BasePriceStone: 8000, BasePriceJade: 100, MinStock: 1, MaxStock: 1, Weight: 4, MaxPerPlayer: 1, Rarity: "epic"},
}

// BlackMarketRarePool 稀有物品池（仅 rare_item 事件时出现）
var BlackMarketRarePool = []BlackMarketItemPool{
	{ItemID: 1005, Name: "九转金丹", Type: ItemTypePill, Description: "逆天改命之神丹", BasePriceStone: 50000, BasePriceJade: 500, MinStock: 1, MaxStock: 1, Weight: 10, MaxPerPlayer: 1, Rarity: "legendary"},
	{ItemID: 2004, Name: "先天至宝碎片", Type: ItemTypeArtifactFragment, Description: "凑齐3片可合成先天至宝", BasePriceStone: 20000, BasePriceJade: 300, MinStock: 1, MaxStock: 2, Weight: 10, MaxPerPlayer: 3, Rarity: "legendary"},
	{ItemID: 3003, Name: "神兽蛋·青龙", Type: ItemTypePetEgg, Description: "孵化获得上古神兽", BasePriceStone: 50000, BasePriceJade: 800, MinStock: 1, MaxStock: 1, Weight: 5, MaxPerPlayer: 1, Rarity: "legendary"},
	{ItemID: 4003, Name: "至尊功法·开天诀", Type: ItemTypeTechnique, Description: "完整的上古神功", BasePriceStone: 30000, BasePriceJade: 400, MinStock: 1, MaxStock: 1, Weight: 8, MaxPerPlayer: 1, Rarity: "legendary"},
	{ItemID: 7004, Name: "混沌战甲", Type: ItemTypeEquipment, Description: "上古魔神遗留的战甲", BasePriceStone: 40000, BasePriceJade: 600, MinStock: 1, MaxStock: 1, Weight: 5, MaxPerPlayer: 1, Rarity: "legendary"},
	{ItemID: 8003, Name: "昆仑镜", Type: ItemTypeTreasure, Description: "上古十大神器之一", BasePriceStone: 80000, BasePriceJade: 1000, MinStock: 1, MaxStock: 1, Weight: 2, MaxPerPlayer: 1, Rarity: "legendary"},
}
