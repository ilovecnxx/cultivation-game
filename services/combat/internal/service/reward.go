package service

import (
	"math"
	"math/rand"

	"cultivation-game/services/combat/internal/engine"
	"cultivation-game/services/combat/internal/model"
)

// DropTable 掉落表
type DropTable struct {
	Items    []DropTableEntry `json:"items"`
	GoldMin  int              `json:"gold_min"`
	GoldMax  int              `json:"gold_max"`
	ExpBase  int              `json:"exp_base"`
}

// DropTableEntry 掉落条目
type DropTableEntry struct {
	ItemID   string  `json:"item_id"`
	ItemName string  `json:"item_name"`
	Rate     float64 `json:"rate"`     // 掉落概率(0~1)
	Quantity int     `json:"quantity"` // 掉落数量
	Rarity   string  `json:"rarity"`  // 品质
}

// RewardCalculator 奖励计算器
type RewardCalculator struct {
	DropTables    map[string]*DropTable `json:"drop_tables"`    // 怪物ID -> 掉落表
	ExpPerLevel   int                   `json:"exp_per_level"`  // 每级基础经验
	PartyBonus    float64               `json:"party_bonus"`    // 每多一人经验加成
}

// NewRewardCalculator 创建奖励计算器
func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{
		DropTables:  make(map[string]*DropTable),
		ExpPerLevel: 50,
		PartyBonus:  0.1,
	}
}

// RegisterDropTable 注册掉落表
func (rc *RewardCalculator) RegisterDropTable(monsterID string, table *DropTable) {
	rc.DropTables[monsterID] = table
}

// CalculateRewards 计算战斗奖励
//
//   - 经验: 每个怪物等级 * ExpPerLevel, 组队加成
//   - 金币: 掉落表随机
//   - 物品: 按概率独立判定
func (rc *RewardCalculator) CalculateRewards(
	battleResult *engine.BattleResult,
	playerLevel int,
	partySize int,
	luck int,
) *engine.BattleRewards {
	rewards := &engine.BattleRewards{
		Exp:   0,
		Gold:  0,
		Items: make([]*engine.DropItem, 0),
	}

	if battleResult.State != engine.BattleStatePlayerWin {
		return rewards
	}

	// 计算经验
	for _, enemy := range battleResult.EnemyTeam {
		if enemy.Type == model.FighterTypeMonster {
			exp := rc.calculateExp(enemy, playerLevel, partySize)
			rewards.Exp += exp

			// 计算掉落
			rc.calculateDrops(enemy, luck, rewards)
		}
	}

	return rewards
}

// calculateExp 计算单个怪物的经验
func (rc *RewardCalculator) calculateExp(monster *model.Fighter, playerLevel int, partySize int) int {
	baseExp := monster.Level * rc.ExpPerLevel

	// 等级差修正: 怪物等级高于玩家, 经验增加; 低则减少
	levelDiff := monster.Level - playerLevel
	levelModifier := 1.0 + float64(levelDiff)*0.1
	if levelModifier < 0.5 {
		levelModifier = 0.5
	}
	if levelModifier > 2.0 {
		levelModifier = 2.0
	}

	baseExp = int(float64(baseExp) * levelModifier)

	// 组队加成
	partyBonus := 1.0 + float64(partySize-1)*rc.PartyBonus
	return int(math.Round(float64(baseExp) * partyBonus))
}

// calculateDrops 计算掉落
func (rc *RewardCalculator) calculateDrops(monster *model.Fighter, luck int, rewards *engine.BattleRewards) {
	table, ok := rc.DropTables[monster.ID]
	if !ok {
		// 没有专属掉落表, 使用通用掉落
		table = rc.getGenericDropTable(monster)
	}
	if table == nil {
		return
	}

	// 金币掉落
	if table.GoldMax > 0 {
		gold := table.GoldMin + rand.Intn(table.GoldMax-table.GoldMin+1)
		rewards.Gold += gold
	}

	// 物品掉落
	for _, entry := range table.Items {
		dropRate := engine.CalculateDropRate(entry.Rate, luck)
		if rand.Float64() < dropRate {
			item := &engine.DropItem{
				ID:       entry.ItemID,
				Name:     entry.ItemName,
				Quantity: entry.Quantity,
				Rarity:   entry.Rarity,
			}
			rewards.Items = append(rewards.Items, item)
		}
	}
}

// getGenericDropTable 获取通用掉落表
func (rc *RewardCalculator) getGenericDropTable(monster *model.Fighter) *DropTable {
	// 根据怪物等级生成通用掉落
	return &DropTable{
		GoldMin: monster.Level * 5,
		GoldMax: monster.Level * 15,
		ExpBase: monster.Level * rc.ExpPerLevel,
		Items: []DropTableEntry{
			{
				ItemID:   "coin_gold",
				ItemName: "金币",
				Rate:     0.5,
				Quantity: monster.Level * 2,
				Rarity:   "common",
			},
			{
				ItemID:   "spirit_stone",
				ItemName: "灵石",
				Rate:     0.3,
				Quantity: monster.Level,
				Rarity:   "uncommon",
			},
		},
	}
}

// DistributeExpToParty 分配经验给队伍
//
//   - 存活者均分经验
//   - 经验值受到每个成员等级修正
func DistributeExpToParty(totalExp int, party []*model.Fighter) map[string]int {
	alive := make([]*model.Fighter, 0)
	for _, p := range party {
		if p.IsAlive() {
			alive = append(alive, p)
		}
	}

	if len(alive) == 0 {
		return nil
	}

	// 基础每人平均
	basePerPerson := totalExp / len(alive)
	distribution := make(map[string]int)

	for _, p := range alive {
		distribution[p.ID] = basePerPerson
	}

	return distribution
}
