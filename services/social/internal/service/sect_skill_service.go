// Package service 宗门技能系统
//
// 技能树：四种加成类型，每类3级
//   - 修炼加成(cultivation_bonus): 5%/10%/15%
//   - 战斗加成(combat_bonus):    3%/6%/10%
//   - 采集加成(gathering_bonus): 10%/20%/30%
//   - 经济加成(economy_bonus):   5%/10%/15%
//
// 升级：宗主/长老消耗宗门资金+个人贡献提升技能等级
// 学习：成员消耗个人贡献学习已解锁技能
package service

import (
	"context"
	"fmt"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SectSkillService 宗门技能业务
type SectSkillService struct {
	db *mongo.Database
}

// NewSectSkillService 创建宗门技能服务
func NewSectSkillService(db *mongo.Database) *SectSkillService {
	return &SectSkillService{db: db}
}

func (s *SectSkillService) skillColl() *mongo.Collection       { return s.db.Collection("sect_skills") }
func (s *SectSkillService) memberSkillColl() *mongo.Collection { return s.db.Collection("sect_member_skills") }
func (s *SectSkillService) memberColl() *mongo.Collection      { return s.db.Collection("sect_members") }
func (s *SectSkillService) sectColl() *mongo.Collection        { return s.db.Collection("sects") }

// SkillTemplate 技能模板定义
type SkillTemplate struct {
	Name                string  // 技能名称
	Description         string  // 描述
	Category            string  // 效果分类
	PerLevelEffect      float64 // 每级效果值
	BaseCostContribution int64  // 每级个人贡献消耗基数
	BaseCostFunds       int64   // 每级宗门资金消耗基数
	MaxLevel            int     // 最高等级
}

// SkillTemplates 全技能模板库
var SkillTemplates = []SkillTemplate{
	{
		Name:                "聚灵诀",
		Description:         "提升全体成员修炼速度",
		Category:            "cultivation_bonus",
		PerLevelEffect:      0.05,
		BaseCostContribution: 1000,
		BaseCostFunds:       5000,
		MaxLevel:            3,
	},
	{
		Name:                "金刚诀",
		Description:         "提升全体成员战斗攻防",
		Category:            "combat_bonus",
		PerLevelEffect:      0.03,
		BaseCostContribution: 1500,
		BaseCostFunds:       8000,
		MaxLevel:            3,
	},
	{
		Name:                "采药术",
		Description:         "提升采集效率",
		Category:            "gathering_bonus",
		PerLevelEffect:      0.10,
		BaseCostContribution: 800,
		BaseCostFunds:       3000,
		MaxLevel:            3,
	},
	{
		Name:                "通商诀",
		Description:         "宗门商店折扣优惠",
		Category:            "economy_bonus",
		PerLevelEffect:      0.05,
		BaseCostContribution: 1200,
		BaseCostFunds:       6000,
		MaxLevel:            3,
	},
}

// GetSkillTree 获取宗门技能树（不存在则初始化）
func (s *SectSkillService) GetSkillTree(ctx context.Context, sectID string) ([]*model.SectSkill, error) {
	cursor, err := s.skillColl().Find(ctx, bson.M{"sect_id": sectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var skills []*model.SectSkill
	if err := cursor.All(ctx, &skills); err != nil {
		return nil, err
	}

	// 首次访问时初始化技能树
	if len(skills) == 0 {
		for _, tmpl := range SkillTemplates {
			skill := &model.SectSkill{
				ID:           uuid.New().String(),
				SectID:       sectID,
				Name:         tmpl.Name,
				Description:  tmpl.Description,
				Level:        0,
				MaxLevel:     tmpl.MaxLevel,
				CostPerLevel: tmpl.BaseCostContribution,
				EffectType:   tmpl.Category,
				EffectValue:  tmpl.PerLevelEffect,
			}
			if _, err := s.skillColl().InsertOne(ctx, skill); err != nil {
				return nil, err
			}
			skills = append(skills, skill)
		}
	}
	return skills, nil
}

// UpgradeSectSkill 升级宗门技能（仅宗主/长老），消耗宗门资金和操作者贡献
func (s *SectSkillService) UpgradeSectSkill(ctx context.Context, sectID, userID, skillID string) (*model.SectSkill, error) {
	// --- 权限校验 ---
	member, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return nil, err
	}
	if member.Rank != model.SectLeader && member.Rank != model.SectElder {
		return nil, fmt.Errorf("仅宗主和长老可升级宗门技能")
	}

	// --- 技能校验 ---
	var skill model.SectSkill
	err = s.skillColl().FindOne(ctx, bson.M{"_id": skillID, "sect_id": sectID}).Decode(&skill)
	if err != nil {
		return nil, fmt.Errorf("技能不存在")
	}
	if skill.Level >= skill.MaxLevel {
		return nil, fmt.Errorf("技能已达最高等级 %d", skill.MaxLevel)
	}

	// --- 计算消耗 ---
	newLevel := skill.Level + 1
	costContribution := skill.CostPerLevel * int64(newLevel)
	costFunds := int64(5000) * int64(newLevel)

	// --- 宗门资金校验 ---
	var sect model.Sect
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return nil, fmt.Errorf("宗门不存在")
	}
	if sect.Funds < costFunds {
		return nil, fmt.Errorf("宗门资金不足，需要 %d 灵石", costFunds)
	}
	if member.Contribution < costContribution {
		return nil, fmt.Errorf("你的个人贡献不足，需要 %d 贡献", costContribution)
	}

	// --- 事务: 扣费 + 升级 ---
		// 扣宗门资金
		if _, err := s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
			bson.M{"$inc": bson.M{"funds": -costFunds}}); err != nil {
			return nil, err
		}
		// 扣操作者贡献
		if _, err := s.memberColl().UpdateOne(ctx,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": -costContribution}}); err != nil {
			return nil, err
		}
		// 技能升级
		if _, err := s.skillColl().UpdateOne(ctx, bson.M{"_id": skillID},
			bson.M{"$set": bson.M{"level": newLevel}}); err != nil {
			return nil, err
		}

	skill.Level = newLevel
	return &skill, nil
}

// LearnMemberSkill 成员消耗个人贡献学习/升级已解锁技能
func (s *SectSkillService) LearnMemberSkill(ctx context.Context, sectID, userID, skillID string) (*model.SectMemberSkill, error) {
	// --- 技能校验 ---
	var skill model.SectSkill
	err := s.skillColl().FindOne(ctx, bson.M{"_id": skillID, "sect_id": sectID}).Decode(&skill)
	if err != nil {
		return nil, fmt.Errorf("技能不存在")
	}
	if skill.Level == 0 {
		return nil, fmt.Errorf("该技能尚未解锁，无法学习")
	}

	// --- 当前成员等级 ---
	var memberSkill model.SectMemberSkill
	err = s.memberSkillColl().FindOne(ctx, bson.M{
		"member_id": userID, "skill_id": skillID,
	}).Decode(&memberSkill)

	currentLevel := 0
	if err == nil {
		currentLevel = memberSkill.Level
	} else if err != mongo.ErrNoDocuments {
		return nil, err
	}

	if currentLevel >= skill.Level {
		return nil, fmt.Errorf("你的技能等级已达宗门当前上限(%d级)", skill.Level)
	}

	newLevel := currentLevel + 1
	cost := skill.CostPerLevel * int64(newLevel)

	// --- 个人贡献校验 ---
	var member model.SectMember
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员信息不存在")
	}
	if member.Contribution < cost {
		return nil, fmt.Errorf("个人贡献不足，需要 %d 贡献", cost)
	}

	// --- 扣贡献 + 升级 ---
	_, _ = s.memberColl().UpdateOne(ctx,
		bson.M{"sect_id": sectID, "user_id": userID},
		bson.M{"$inc": bson.M{"contribution": -cost}},
	)

	_, err = s.memberSkillColl().UpdateOne(ctx,
		bson.M{"member_id": userID, "skill_id": skillID},
		bson.M{
			"$set": bson.M{
				"member_id": userID,
				"skill_id":  skillID,
				"level":     newLevel,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return nil, err
	}

	return &model.SectMemberSkill{
		MemberID: userID,
		SkillID:  skillID,
		Level:    newLevel,
	}, nil
}

// GetMemberSkillBonuses 计算成员技能总加成（按效果类型聚合）
func (s *SectSkillService) GetMemberSkillBonuses(ctx context.Context, sectID, userID string) map[string]float64 {
	bonuses := make(map[string]float64)

	cursor, err := s.memberSkillColl().Find(ctx, bson.M{"member_id": userID})
	if err != nil {
		return bonuses
	}
	defer cursor.Close(ctx)

	var memberSkills []model.SectMemberSkill
	if err := cursor.All(ctx, &memberSkills); err != nil {
		return bonuses
	}

	for _, ms := range memberSkills {
		var skill model.SectSkill
		err := s.skillColl().FindOne(ctx, bson.M{"_id": ms.SkillID, "sect_id": sectID}).Decode(&skill)
		if err != nil {
			continue
		}
		bonuses[skill.EffectType] += skill.EffectValue * float64(ms.Level)
	}
	return bonuses
}

func (s *SectSkillService) getMember(ctx context.Context, sectID, userID string) (*model.SectMember, error) {
	var member model.SectMember
	err := s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	return &member, nil
}
