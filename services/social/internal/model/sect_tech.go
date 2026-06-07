// Package model 宗门科技树数据模型
package model

// SectTechBranch 科技树分支定义
type SectTechBranch struct {
	ID          string `json:"id"`          // 分支标识
	Name        string `json:"name"`        // 分支名称
	Description string `json:"description"` // 描述
	Icon        string `json:"icon"`        // 图标
}

// SectTechBranches 全部科技分支
var SectTechBranches = []SectTechBranch{
	{ID: "cultivation", Name: "修炼加成", Description: "提升宗门成员修炼速度与突破效率", Icon: "&#x1F4A0;"},
	{ID: "combat", Name: "战斗加成", Description: "增强宗门成员战斗攻击与防御", Icon: "&#x2694;"},
	{ID: "gathering", Name: "采集加成", Description: "提高资源采集效率与产出", Icon: "&#x26CF;"},
	{ID: "economy", Name: "经济效益", Description: "降低消耗、提升灵石收益", Icon: "&#x1F4B0;"},
	{ID: "defense", Name: "防御阵地", Description: "加强宗门阵法防御与领地稳固", Icon: "&#x1F6E1;"},
}

// TechLevelConfig 科技等级配置
type TechLevelConfig struct {
	Level          int     `json:"level"`           // 等级 1-10
	EffectValue    float64 `json:"effect_value"`    // 效果值
	CostContribute int64   `json:"cost_contribute"` // 消耗宗门贡献
	CostFunds      int64   `json:"cost_funds"`      // 消耗宗门资金
}

// TechBranchConfig 科技分支完整配置（含各级数据）
type TechBranchConfig struct {
	BranchID string            `json:"branch_id"` // 对应 SectTechBranches.ID
	Levels   []TechLevelConfig `json:"levels"`
}

// TechConfigs 所有科技分支的等级配置
var TechConfigs = []TechBranchConfig{
	{
		BranchID: "cultivation",
		Levels: []TechLevelConfig{
			{Level: 1, EffectValue: 0.05, CostContribute: 500, CostFunds: 2000},
			{Level: 2, EffectValue: 0.10, CostContribute: 800, CostFunds: 4000},
			{Level: 3, EffectValue: 0.15, CostContribute: 1200, CostFunds: 6000},
			{Level: 4, EffectValue: 0.20, CostContribute: 1800, CostFunds: 9000},
			{Level: 5, EffectValue: 0.25, CostContribute: 2500, CostFunds: 13000},
			{Level: 6, EffectValue: 0.30, CostContribute: 3500, CostFunds: 18000},
			{Level: 7, EffectValue: 0.35, CostContribute: 5000, CostFunds: 25000},
			{Level: 8, EffectValue: 0.40, CostContribute: 7000, CostFunds: 35000},
			{Level: 9, EffectValue: 0.45, CostContribute: 10000, CostFunds: 50000},
			{Level: 10, EffectValue: 0.50, CostContribute: 15000, CostFunds: 75000},
		},
	},
	{
		BranchID: "combat",
		Levels: []TechLevelConfig{
			{Level: 1, EffectValue: 0.03, CostContribute: 400, CostFunds: 2500},
			{Level: 2, EffectValue: 0.06, CostContribute: 700, CostFunds: 4500},
			{Level: 3, EffectValue: 0.10, CostContribute: 1100, CostFunds: 7000},
			{Level: 4, EffectValue: 0.14, CostContribute: 1600, CostFunds: 10000},
			{Level: 5, EffectValue: 0.18, CostContribute: 2200, CostFunds: 14000},
			{Level: 6, EffectValue: 0.22, CostContribute: 3000, CostFunds: 20000},
			{Level: 7, EffectValue: 0.26, CostContribute: 4200, CostFunds: 28000},
			{Level: 8, EffectValue: 0.30, CostContribute: 5800, CostFunds: 38000},
			{Level: 9, EffectValue: 0.35, CostContribute: 8000, CostFunds: 55000},
			{Level: 10, EffectValue: 0.40, CostContribute: 12000, CostFunds: 80000},
		},
	},
	{
		BranchID: "gathering",
		Levels: []TechLevelConfig{
			{Level: 1, EffectValue: 0.10, CostContribute: 300, CostFunds: 1500},
			{Level: 2, EffectValue: 0.20, CostContribute: 500, CostFunds: 3000},
			{Level: 3, EffectValue: 0.30, CostContribute: 800, CostFunds: 5000},
			{Level: 4, EffectValue: 0.40, CostContribute: 1200, CostFunds: 8000},
			{Level: 5, EffectValue: 0.50, CostContribute: 1800, CostFunds: 12000},
			{Level: 6, EffectValue: 0.60, CostContribute: 2500, CostFunds: 17000},
			{Level: 7, EffectValue: 0.70, CostContribute: 3500, CostFunds: 24000},
			{Level: 8, EffectValue: 0.80, CostContribute: 5000, CostFunds: 34000},
			{Level: 9, EffectValue: 0.90, CostContribute: 7000, CostFunds: 48000},
			{Level: 10, EffectValue: 1.00, CostContribute: 10000, CostFunds: 70000},
		},
	},
	{
		BranchID: "economy",
		Levels: []TechLevelConfig{
			{Level: 1, EffectValue: 0.05, CostContribute: 500, CostFunds: 2000},
			{Level: 2, EffectValue: 0.10, CostContribute: 800, CostFunds: 4000},
			{Level: 3, EffectValue: 0.15, CostContribute: 1200, CostFunds: 6500},
			{Level: 4, EffectValue: 0.20, CostContribute: 1700, CostFunds: 9500},
			{Level: 5, EffectValue: 0.25, CostContribute: 2400, CostFunds: 13500},
			{Level: 6, EffectValue: 0.30, CostContribute: 3300, CostFunds: 19000},
			{Level: 7, EffectValue: 0.35, CostContribute: 4800, CostFunds: 26000},
			{Level: 8, EffectValue: 0.40, CostContribute: 6800, CostFunds: 36000},
			{Level: 9, EffectValue: 0.45, CostContribute: 9500, CostFunds: 52000},
			{Level: 10, EffectValue: 0.50, CostContribute: 14000, CostFunds: 72000},
		},
	},
	{
		BranchID: "defense",
		Levels: []TechLevelConfig{
			{Level: 1, EffectValue: 0.05, CostContribute: 600, CostFunds: 3000},
			{Level: 2, EffectValue: 0.10, CostContribute: 900, CostFunds: 5500},
			{Level: 3, EffectValue: 0.15, CostContribute: 1400, CostFunds: 8500},
			{Level: 4, EffectValue: 0.20, CostContribute: 2000, CostFunds: 12000},
			{Level: 5, EffectValue: 0.25, CostContribute: 2800, CostFunds: 17000},
			{Level: 6, EffectValue: 0.30, CostContribute: 4000, CostFunds: 24000},
			{Level: 7, EffectValue: 0.35, CostContribute: 5500, CostFunds: 33000},
			{Level: 8, EffectValue: 0.40, CostContribute: 7500, CostFunds: 45000},
			{Level: 9, EffectValue: 0.45, CostContribute: 11000, CostFunds: 60000},
			{Level: 10, EffectValue: 0.50, CostContribute: 16000, CostFunds: 85000},
		},
	},
}

// GetTechConfig 获取指定分支的等级配置
func GetTechConfig(branchID string, level int) *TechLevelConfig {
	for _, bc := range TechConfigs {
		if bc.BranchID == branchID {
			for _, lc := range bc.Levels {
				if lc.Level == level {
					return &lc
				}
			}
		}
	}
	return nil
}

// NewSectTech 创建宗门科技对象
type NewSectTech struct {
	SectID  string `json:"sect_id"`
	Branch  string `json:"branch"`  // 分支ID
	Level   int    `json:"level"`   // 当前等级
	MaxTech int    `json:"max_tech"` // 最大等级
}
