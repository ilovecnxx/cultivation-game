// Package service 宗门科技树
//
// 科技树: 修炼加成/战斗加成/采集加成/经济效益/防御阵地(5条路线)
// 每条路线10级
// 升级需要宗门贡献+宗门资金
// 长老以上可提议升级
package service

import (
	"context"
	"fmt"
	"time"

	"cultivation-game/services/social/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// SectTech 宗门科技数据（MongoDB文档）
type SectTech struct {
	ID        string    `bson:"_id" json:"id"`
	SectID    string    `bson:"sect_id" json:"sect_id"`
	Branch    string    `bson:"branch" json:"branch"`       // 分支ID: cultivation/combat/gathering/economy/defense
	Level     int       `bson:"level" json:"level"`         // 当前等级 0-10
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// SectTechService 宗门科技业务
type SectTechService struct {
	db *mongo.Database
}

// NewSectTechService 创建宗门科技服务
func NewSectTechService(db *mongo.Database) *SectTechService {
	return &SectTechService{db: db}
}

func (s *SectTechService) techColl() *mongo.Collection       { return s.db.Collection("sect_techs") }
func (s *SectTechService) memberColl() *mongo.Collection     { return s.db.Collection("sect_members") }
func (s *SectTechService) sectColl() *mongo.Collection        { return s.db.Collection("sects") }

// GetTechList 获取宗门科技列表
// GET /api/v1/sect/tech/list?sect_id=xxx
func (s *SectTechService) GetTechList(ctx context.Context, sectID string) ([]*SectTech, error) {
	cursor, err := s.techColl().Find(ctx, bson.M{"sect_id": sectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var techs []*SectTech
	if err := cursor.All(ctx, &techs); err != nil {
		return nil, err
	}

	// 首次访问时初始化所有科技分支
	if len(techs) == 0 {
		for _, branch := range model.SectTechBranches {
			tech := &SectTech{
				ID:        branch.ID + "_" + sectID,
				SectID:    sectID,
				Branch:    branch.ID,
				Level:     0,
				UpdatedAt: time.Now(),
			}
			if _, err := s.techColl().InsertOne(ctx, tech); err != nil {
				return nil, err
			}
			techs = append(techs, tech)
		}
	}
	return techs, nil
}

// UpgradeTech 升级宗门科技
// POST /api/v1/sect/tech/upgrade
// 仅长老及以上可操作
func (s *SectTechService) UpgradeTech(ctx context.Context, sectID, userID, branch string) (*SectTech, error) {
	// --- 权限校验：长老以上 ---
	member, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return nil, err
	}
	if member.Rank != model.SectLeader && member.Rank != model.SectElder {
		return nil, fmt.Errorf("仅宗主和长老可升级宗门科技")
	}

	// --- 查找当前科技等级 ---
	var tech SectTech
	err = s.techColl().FindOne(ctx, bson.M{"sect_id": sectID, "branch": branch}).Decode(&tech)
	if err != nil {
		return nil, fmt.Errorf("科技分支不存在")
	}

	if tech.Level >= 10 {
		return nil, fmt.Errorf("该科技分支已达最高等级(10级)")
	}

	nextLevel := tech.Level + 1
	levelCfg := model.GetTechConfig(branch, nextLevel)
	if levelCfg == nil {
		return nil, fmt.Errorf("配置错误：找不到等级 %d 的数据", nextLevel)
	}

	// --- 查询宗门 ---
	var sect struct {
		Funds int64 `bson:"funds"`
	}
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return nil, fmt.Errorf("宗门不存在")
	}

	// --- 资源校验 ---
	if sect.Funds < levelCfg.CostFunds {
		return nil, fmt.Errorf("宗门资金不足，需要 %d 灵石，当前 %d", levelCfg.CostFunds, sect.Funds)
	}
	if member.Contribution < levelCfg.CostContribute {
		return nil, fmt.Errorf("你的宗门贡献不足，需要 %d 贡献，当前 %d", levelCfg.CostContribute, member.Contribution)
	}

	// --- 事务: 扣费 + 升级 ---
		// 扣宗门资金
		if _, err := s.sectColl().UpdateOne(ctx,
			bson.M{"_id": sectID},
			bson.M{"$inc": bson.M{"funds": -levelCfg.CostFunds}}); err != nil {
			return nil, err
		}
		// 扣操作者贡献
		if _, err := s.memberColl().UpdateOne(ctx,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": -levelCfg.CostContribute}}); err != nil {
			return nil, err
		}
		// 科技升级
		if _, err := s.techColl().UpdateOne(ctx,
			bson.M{"sect_id": sectID, "branch": branch},
			bson.M{"$set": bson.M{"level": nextLevel, "updated_at": time.Now()}}); err != nil {
			return nil, err
		}

	tech.Level = nextLevel
	return &tech, nil
}

// GetTechBonuses 获取宗门科技对所有成员的加成效果
func (s *SectTechService) GetTechBonuses(ctx context.Context, sectID string) map[string]float64 {
	bonuses := make(map[string]float64)
	techs, err := s.GetTechList(ctx, sectID)
	if err != nil {
		return bonuses
	}
	for _, t := range techs {
		if t.Level > 0 {
			cfg := model.GetTechConfig(t.Branch, t.Level)
			if cfg != nil {
				bonuses[t.Branch] = cfg.EffectValue
			}
		}
	}
	return bonuses
}

func (s *SectTechService) getMember(ctx context.Context, sectID, userID string) (*model.SectMember, error) {
	var member model.SectMember
	err := s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("宗门成员不存在")
	}
	return &member, nil
}
