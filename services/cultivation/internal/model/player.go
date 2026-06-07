// Package model 定义修仙游戏核心数据结构
package model

// Player 玩家修炼状态（简化版，核心字段）
type Player struct {
	ID               uint64  `json:"id"`
	Name             string  `json:"name"`
	RealmID          int     `json:"realm_id"`          // 当前大境界ID
	RealmLevel       int     `json:"realm_level"`       // 当前小境界等级
	Experience       int64   `json:"experience"`        // 当前修为值
	BaseAttack       int64   `json:"base_attack"`
	BaseDefense      int64   `json:"base_defense"`
	BaseHP           int64   `json:"base_hp"`

	// 装备的功法
	TechniqueID      int     `json:"technique_id"`
	TechniqueLevel   int     `json:"technique_level"`   // 功法修炼等级

	// 灵根属性（金木水火土）
	SpiritRoots     map[string]float64 `json:"spirit_roots"` // 每种灵根亲和度0.0-1.0

	// 丹药/法宝加成
	PillBonuses     map[string]float64 `json:"pill_bonuses"` // 临时丹药加成
	ArtifactBonuses map[string]float64 `json:"artifact_bonuses"` // 法宝加成

	// 玩家状态（idle, cultivating, adventuring, exploring）
	Status          string  `json:"status"`             // 当前状态，默认"idle"

	// 修炼模式（online=在线, offline=离线，仅 status=cultivating 时有意义）
	CultivationMode string  `json:"cultivation_mode,omitempty"`

	// 离线闭关（已废弃，合并到 /api/v1/cultivate 统一处理，保留字段兼容）
	IsMeditating     bool  `json:"is_meditating"`
	MeditationStart  int64 `json:"meditation_start"`
	AccumulatedExp   int64 `json:"accumulated_exp"`

	// 在线挂机修炼
	IsCultivating    bool  `json:"is_cultivating"`
	CultivationStart int64 `json:"cultivation_start"`

	// V2 气运与业力
	Luck             int64 `json:"luck"`               // 气运值
	Karma            int64 `json:"karma"`              // 业力值
	MaxExpForLevel   int64 `json:"max_exp_for_level"`  // 当前等级最大修为（突破所需）
	DailyCheckInTime int64 `json:"daily_checkin_time"` // 每日签到时间戳

	// 当前区域灵气浓度（从世界服务获取）
	SpiritDensity float64 `json:"spirit_density"` // 0.5~5.0

	// 道心层数（突破失败累积，下次突破节点生成速度减慢）
	DaoXinStacks int `json:"dao_xin_stacks"` // 0~3层

	// 炼丹系统
	AlchemyLevel int        `json:"alchemy_level"` // 炼丹等级
	AlchemyExp   int64      `json:"alchemy_exp"`   // 炼丹经验
	Ingredients  map[int]int `json:"ingredients"`   // 已收集材料 ingredient_id -> 数量
	Pills        []Pill     `json:"pills"`          // 持有的丹药

	// 持久心魔系统
	HeartDemons      []PersistentHeartDemon `json:"heart_demons,omitempty"`       // 当前活跃的心魔
	SuppressionItems map[string]int         `json:"suppression_items,omitempty"`  // 压制道具 item_type -> count
	HasBodhiTechnique bool                  `json:"has_bodhi_technique"`          // 是否习得菩提心法
}

// SpiritRootMultiplier 计算灵根倍率（取最高灵根值映射）
// 天灵根(≥0.9)=2.0, 地灵根(≥0.7)=1.5, 人灵根(≥0.4)=1.0, 杂灵根(<0.4)=0.7
func (p *Player) SpiritRootMultiplier() float64 {
	highest := 0.0
	for _, v := range p.SpiritRoots {
		if v > highest {
			highest = v
		}
	}
	switch {
	case highest >= 0.9:
		return 2.0
	case highest >= 0.7:
		return 1.5
	case highest >= 0.4:
		return 1.0
	default:
		return 0.7
	}
}

// GetBreakthroughBonus 获取玩家所有突破加成总和
func (p *Player) GetBreakthroughBonus() float64 {
	total := 0.0
	for _, bonus := range p.PillBonuses {
		total += bonus
	}
	for _, bonus := range p.ArtifactBonuses {
		total += bonus
	}
	return total
}

// BreakthroughResult 突破结果
type BreakthroughResult struct {
	Success   bool    `json:"success"`     // 是否成功
	FinalRate float64 `json:"final_rate"`  // 最终概率
	NewRealmID   int  `json:"new_realm_id,omitempty"`
	NewRealmLevel int `json:"new_realm_level,omitempty"`
	ExpLoss    int64  `json:"exp_loss,omitempty"`    // 失败修为损失
	Dropped    bool   `json:"dropped,omitempty"`     // 是否掉境
	LuckCost   int64  `json:"luck_cost,omitempty"`   // 消耗的气运
	KarmaGained int64 `json:"karma_gained,omitempty"` // 获得的业力

	// 心魔
	HeartDemon *HeartDemon `json:"heart_demon,omitempty"`

	// 天劫相关
	Tribulation     *TribulationResult `json:"tribulation,omitempty"`
}

// TribulationResult 天劫结果
type TribulationResult struct {
	Triggered  bool    `json:"triggered"`           // 是否触发天劫
	Success    bool    `json:"success"`             // 是否通过
	Rate       float64 `json:"rate"`                // 通过概率
	Damage     int64   `json:"damage"`              // 天劫造成的伤害
	Survived   bool    `json:"survived"`            // 是否存活

	// V2 天劫
	ThunderCount  int            `json:"thunder_count"`   // 劫雷数量
	ThunderPassed int            `json:"thunder_passed"`  // 通过劫雷数
	HeartDemons   []*HeartDemon  `json:"heart_demons,omitempty"` // 心魔列表
	KarmaGained   int64          `json:"karma_gained"`    // 获得业力
	TribPower     *TribulationPower `json:"tribulation_power,omitempty"` // 劫力（渡劫期）
}

// HeartDemon 心魔（事件型，突破失败时的情景选择）
type HeartDemon struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Scenario   string   `json:"scenario"`
	Options    []string `json:"options"`
	KarmaCost  int64    `json:"karma_cost"`  // 破除心魔所需业力
	Damage     int64    `json:"damage"`      // 不破除时的伤害
}

// PersistentDemonType 五大心魔类型
type PersistentDemonType string

const (
	DemonGreed     PersistentDemonType = "greed"     // 贪 - reduced drop rates, increased spirit stone cost
	DemonWrath     PersistentDemonType = "wrath"     // 嗔 - reduced defense, increased damage taken
	DemonIgnorance PersistentDemonType = "ignor"     // 痴 - reduced exp gain, increased breakthrough difficulty
	DemonDoubt     PersistentDemonType = "doubt"     // 疑 - reduced crit rate, increased skill cooldown
	DemonSloth     PersistentDemonType = "sloth"     // 慢 - reduced cultivation speed, reduced movement speed
)

// PersistentHeartDemon 持久心魔（玩家长期携带的Debuff来源）
type PersistentHeartDemon struct {
	ID          string               `json:"id"`
	PlayerID    uint64               `json:"player_id"`
	DemonType   PersistentDemonType  `json:"demon_type"`
	Level       int                  `json:"level"`        // 1-10, higher = stronger debuff
	DebuffValue float64              `json:"debuff_value"` // percentage debuff
	CreatedAt   int64                `json:"created_at"`
	CreatedFrom string               `json:"created_from"` // "breakthrough_failure", "tribulation_failure", "curse"
	Defeated    bool                 `json:"defeated"`
	DefeatedAt  int64                `json:"defeated_at,omitempty"`
}

// DemonIllusionRecord 心魔幻境挑战记录
type DemonIllusionRecord struct {
	ID          string              `json:"id"`
	PlayerID    uint64              `json:"player_id"`
	DemonType   PersistentDemonType `json:"demon_type"`
	ChallengedAt int64              `json:"challenged_at"`
	Won         bool                `json:"won"`
	LevelBefore int                 `json:"level_before"`
	LevelAfter  int                 `json:"level_after"`
}

// SuppressionItem 心魔压制物品
type SuppressionItemType string

const (
	ItemCleansingPill      SuppressionItemType = "cleansing_pill"      // 清心丹: -1 level random demon
	ItemSuppressionTalisman SuppressionItemType = "suppression_talisman" // 镇魔符: -2 levels
	ItemBodhiTechnique     SuppressionItemType = "bodhi_technique"     // 菩提心法: passive prevention
)

// HeartDemonScenario 心魔情景模板
type HeartDemonScenario struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Scenario string   `json:"scenario"`
	OptionA  string   `json:"option_a"`
	OptionB  string   `json:"option_b"`
	OptionC  string   `json:"option_c"`
	KarmaCost int64   `json:"karma_cost"`
	Damage   int64    `json:"damage"`
}

// TribulationType 渡劫类型
type TribulationType int

const (
	Tribulation39 TribulationType = iota // 三九雷劫 (3 waves, minor realm breakthrough)
	Tribulation69                        // 六九雷劫 (6 waves, major realm breakthrough)
	Tribulation99                        // 九九雷劫 (9 waves, ascension breakthrough)
)

// TribulationSession 渡劫会话（交互式渡劫系统 V2）
type TribulationSession struct {
	PlayerID      string         `json:"player_id"`
	PlayerName    string         `json:"player_name"`
	Type          TribulationType `json:"type"`
	TypeName      string          `json:"type_name"`
	CurrentWave   int             `json:"current_wave"`
	TotalWaves    int             `json:"total_waves"`
	StrikesPerWave int            `json:"strikes_per_wave"`
	PlayerHP      int64           `json:"player_hp"`
	MaxHP         int64           `json:"max_hp"`
	DamageTaken   int64           `json:"damage_taken"`
	StartTime     int64           `json:"start_time"`
	Status        string          `json:"status"` // "active", "success", "failed"
	Guardians     []string        `json:"guardians"`
	RealmID       int             `json:"realm_id"`
	RealmLevel    int             `json:"realm_level"`
	BonusStats    *TribulationBonus `json:"bonus_stats,omitempty"`
}

// TribulationBonus 渡劫成功后获得的额外属性
type TribulationBonus struct {
	HPPermanent    int64   `json:"hp_permanent"`
	AttackBonus    float64 `json:"attack_bonus"`
	DefenseBonus   float64 `json:"defense_bonus"`
	SpeedBonus     float64 `json:"speed_bonus"`
	DaoXinRecover  int     `json:"dao_xin_recover"`
}

// WaveAction 玩家在每波雷劫中的行动选择
type WaveAction struct {
	Action   string `json:"action"`    // "endure", "dodge", "artifact"
	ItemID   string `json:"item_id,omitempty"`   // 使用法宝时的法宝ID
}

// WaveResult 每波雷劫的结果
type WaveResult struct {
	Wave         int    `json:"wave"`
	Strikes      int    `json:"strikes"`
	DamageBefore int64  `json:"damage_before"`
	DamageAfter  int64  `json:"damage_after"`
	DamageReduced int64 `json:"damage_reduced"`
	Dodged       bool   `json:"dodged"`
	Action       string `json:"action"`
	HPRemaining  int64  `json:"hp_remaining"`
	MaxHP        int64  `json:"max_hp"`
	Survived     bool   `json:"survived"`
	IsFinal      bool   `json:"is_final"`
}

// TribulationPower 劫力（渡劫期专用）
type TribulationPower struct {
	Current int64 `json:"current"` // 当前劫力
	Max     int64 `json:"max"`     // 劫力上限
}

// BreakthroughEvent 突破事件（供事件总线使用）
type BreakthroughEvent struct {
	PlayerID   uint64 `json:"player_id"`
	NewRealmID int    `json:"new_realm_id"`
}

// TribulationEvent 渡劫事件（供事件总线使用）
type TribulationEvent struct {
	PlayerID   string          `json:"player_id"`
	PlayerName string          `json:"player_name"`
	Type       TribulationType `json:"type"`
	TypeName   string          `json:"type_name"`
	Status     string          `json:"status"` // "started", "success", "failed"
	Wave       int             `json:"wave,omitempty"`
}

// EventBus 简单事件总线接口
type EventBus interface {
	Publish(event string, data interface{})
	Subscribe(event string, handler func(data interface{})) func() // 返回取消订阅函数
}
