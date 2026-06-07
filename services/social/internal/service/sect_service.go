package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SectService 宗门业务逻辑
type SectService struct {
	db *mongo.Database
}

// NewSectService 创建宗门服务
func NewSectService(db *mongo.Database) *SectService {
	return &SectService{db: db}
}

func (s *SectService) sectColl() *mongo.Collection      { return s.db.Collection("sects") }
func (s *SectService) memberColl() *mongo.Collection    { return s.db.Collection("sect_members") }
func (s *SectService) skillColl() *mongo.Collection     { return s.db.Collection("sect_skills") }
func (s *SectService) memberSkillColl() *mongo.Collection { return s.db.Collection("sect_member_skills") }
func (s *SectService) applyColl() *mongo.Collection     { return s.db.Collection("sect_applies") }

// ============================================================
// 宗门基础管理
// ============================================================

// CreateSect 创建宗门
func (s *SectService) CreateSect(ctx context.Context, name, description, notice, leaderID, leaderName string) (*model.Sect, error) {
	// 检查宗门名是否已存在
	count, err := s.sectColl().CountDocuments(ctx, bson.M{"name": name})
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("宗门名 '%s' 已被占用", name)
	}

	// 检查玩家是否已有宗门
	mcount, err := s.memberColl().CountDocuments(ctx, bson.M{"user_id": leaderID})
	if err != nil {
		return nil, err
	}
	if mcount > 0 {
		return nil, fmt.Errorf("你已属于其他宗门")
	}

	now := time.Now()
	sect := &model.Sect{
		ID:          uuid.New().String(),
		Name:        name,
		Level:       1,
		Experience:  0,
		MemberCount: 1,
		MaxMembers:  20,
		LeaderID:    leaderID,
		LeaderName:  leaderName,
		Notice:      notice,
		Description: description,
		CreatedAt:   now,
	}

	// 创建宗门和宗主成员(事务)
	session, err := s.db.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if _, err := s.sectColl().InsertOne(sc, sect); err != nil {
			return nil, err
		}
		member := &model.SectMember{
			SectID:  sect.ID,
			UserID:  leaderID,
			Rank:    model.SectLeader,
			JoinedAt: now,
		}
		if _, err := s.memberColl().InsertOne(sc, member); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("创建宗门失败: %w", err)
	}

	// 初始化宗门技能
	s.initDefaultSkills(ctx, sect.ID)

	return sect, nil
}

// initDefaultSkills 创建宗门时为宗门初始化默认技能
func (s *SectService) initDefaultSkills(ctx context.Context, sectID string) error {
	defaultSkills := []struct {
		Name        string
		Description string
		EffectType  string
		EffectValue float64
		CostPerLevel int64
		MaxLevel    int
	}{
		{"聚灵诀", "提升全体成员修炼速度", "cultivation_bonus", 0.02, 1000, 10},
		{"金刚诀", "提升全体成员战斗防御", "combat_bonus", 0.015, 1500, 10},
		{"采药术", "提升采集效率", "gathering_bonus", 0.025, 800, 10},
	}

	for _, sk := range defaultSkills {
		skill := &model.SectSkill{
			ID:           uuid.New().String(),
			SectID:       sectID,
			Name:         sk.Name,
			Description:  sk.Description,
			Level:        1,
			MaxLevel:     sk.MaxLevel,
			CostPerLevel: sk.CostPerLevel,
			EffectType:   sk.EffectType,
			EffectValue:  sk.EffectValue,
		}
		if _, err := s.skillColl().InsertOne(ctx, skill); err != nil {
			return err
		}
	}
	return nil
}

// GetSect 获取宗门信息
func (s *SectService) GetSect(ctx context.Context, sectID string) (*model.Sect, error) {
	var sect model.Sect
	err := s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err != nil {
		return nil, fmt.Errorf("宗门不存在: %w", err)
	}
	return &sect, nil
}

// SearchSect 搜索宗门(按名称模糊匹配)
func (s *SectService) SearchSect(ctx context.Context, keyword string, page, pageSize int64) ([]*model.Sect, int64, error) {
	filter := bson.M{"name": bson.M{"$regex": keyword, "$options": "i"}}
	total, err := s.sectColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().SetSkip(skip).SetLimit(pageSize).SetSort(bson.D{{Key: "level", Value: -1}})
	cursor, err := s.sectColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var sects []*model.Sect
	if err := cursor.All(ctx, &sects); err != nil {
		return nil, 0, err
	}
	return sects, total, nil
}

// GetUserSect 获取玩家所在宗门
func (s *SectService) GetUserSect(ctx context.Context, userID string) (*model.Sect, *model.SectMember, error) {
	var member model.SectMember
	err := s.memberColl().FindOne(ctx, bson.M{"user_id": userID}).Decode(&member)
	if err != nil {
		return nil, nil, fmt.Errorf("你未加入任何宗门")
	}
	sect, err := s.GetSect(ctx, member.SectID)
	if err != nil {
		return nil, nil, err
	}
	return sect, &member, nil
}

// ============================================================
// 成员管理
// ============================================================

// JoinSect 申请加入宗门
func (s *SectService) JoinSect(ctx context.Context, sectID, userID, userName, message string) error {
	// 检查是否已有宗门
	count, err := s.memberColl().CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("你已属于其他宗门")
	}

	// 检查是否已申请
	existing, _ := s.applyColl().CountDocuments(ctx, bson.M{
		"sect_id": sectID, "user_id": userID, "status": "pending",
	})
	if existing > 0 {
		return fmt.Errorf("已提交过申请，请等待审核")
	}

	apply := &model.SectApply{
		ID:        uuid.New().String(),
		SectID:    sectID,
		UserID:    userID,
		UserName:  userName,
		Message:   message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	_, err = s.applyColl().InsertOne(ctx, apply)
	return err
}

// HandleApply 处理宗门申请
func (s *SectService) HandleApply(ctx context.Context, applyID string, accept bool, operatorID string) error {
	// 验证操作者权限
	var apply model.SectApply
	err := s.applyColl().FindOne(ctx, bson.M{"_id": applyID, "status": "pending"}).Decode(&apply)
	if err != nil {
		return fmt.Errorf("申请不存在或已处理")
	}

	// 检查操作者是否有权限(宗主/长老)
	rank, err := s.getMemberRank(ctx, apply.SectID, operatorID)
	if err != nil || (rank != model.SectLeader && rank != model.SectElder) {
		return fmt.Errorf("无权处理申请")
	}

	if !accept {
		_, err = s.applyColl().UpdateOne(ctx, bson.M{"_id": applyID},
			bson.M{"$set": bson.M{"status": "rejected"}})
		return err
	}

	// 同意: 更新申请状态并添加成员(事务)
	session, err := s.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		// 更新申请
		if _, err := s.applyColl().UpdateOne(sc, bson.M{"_id": applyID},
			bson.M{"$set": bson.M{"status": "accepted"}}); err != nil {
			return nil, err
		}

		// 添加成员
		member := &model.SectMember{
			SectID:  apply.SectID,
			UserID:  apply.UserID,
			Rank:    model.SectRankMember,
			JoinedAt: time.Now(),
		}
		if _, err := s.memberColl().InsertOne(sc, member); err != nil {
			return nil, err
		}

		// 更新成员计数
		if _, err := s.sectColl().UpdateOne(sc, bson.M{"_id": apply.SectID},
			bson.M{"$inc": bson.M{"member_count": 1}}); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// LeaveSect 退出宗门
func (s *SectService) LeaveSect(ctx context.Context, sectID, userID string) error {
	// 检查是否宗主(宗主需先转让)
	member, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return err
	}
	if member.Rank == model.SectLeader {
		return fmt.Errorf("宗主无法退出宗门，请先转让宗主之位")
	}

	// 移除成员
	_, err = s.memberColl().DeleteOne(ctx, bson.M{"sect_id": sectID, "user_id": userID})
	if err != nil {
		return err
	}
	// 减少成员计数
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
		bson.M{"$inc": bson.M{"member_count": -1}})
	return nil
}

// KickMember 踢出成员(宗主/长老操作)
func (s *SectService) KickMember(ctx context.Context, sectID, operatorID, targetID string) error {
	// 验证权限
	opRank, err := s.getMemberRank(ctx, sectID, operatorID)
	if err != nil {
		return err
	}
	if opRank != model.SectLeader && opRank != model.SectElder {
		return fmt.Errorf("无权踢出成员")
	}

	targetRank, err := s.getMemberRank(ctx, sectID, targetID)
	if err != nil {
		return err
	}
	// 不能踢出比自己职位高的
	if rankWeight(targetRank) >= rankWeight(opRank) {
		return fmt.Errorf("无法踢出该成员")
	}

	_, err = s.memberColl().DeleteOne(ctx, bson.M{"sect_id": sectID, "user_id": targetID})
	if err != nil {
		return err
	}
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
		bson.M{"$inc": bson.M{"member_count": -1}})
	return nil
}

// TransferLeader 转让宗主
func (s *SectService) TransferLeader(ctx context.Context, sectID, currentLeaderID, newLeaderID string) error {
	// 验证当前身份
	member, err := s.getMember(ctx, sectID, currentLeaderID)
	if err != nil {
		return err
	}
	if member.Rank != model.SectLeader {
		return fmt.Errorf("仅宗主可转让")
	}

	// 验证新宗主是否在宗门
	newMember, err := s.getMember(ctx, sectID, newLeaderID)
	if err != nil {
		return fmt.Errorf("目标玩家不在宗门内")
	}

	now := time.Now()
	// 转让: 原宗主降为长老, 新宗主升为宗主
	session, err := s.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		_, _ = s.memberColl().UpdateOne(sc,
			bson.M{"sect_id": sectID, "user_id": currentLeaderID},
			bson.M{"$set": bson.M{"rank": model.SectElder, "updated_at": now}})
		_, _ = s.memberColl().UpdateOne(sc,
			bson.M{"sect_id": sectID, "user_id": newLeaderID},
			bson.M{"$set": bson.M{"rank": model.SectLeader, "updated_at": now}})
		_, _ = s.sectColl().UpdateOne(sc, bson.M{"_id": sectID},
			bson.M{"$set": bson.M{"leader_id": newLeaderID, "leader_name": newMember.UserID}})
		return nil, nil
	})
	return err
}

// SetMemberRank 设置成员职位(仅宗主可操作)
func (s *SectService) SetMemberRank(ctx context.Context, sectID, operatorID, targetID string, newRank model.SectRank) error {
	opRank, err := s.getMemberRank(ctx, sectID, operatorID)
	if err != nil {
		return err
	}
	if opRank != model.SectLeader {
		return fmt.Errorf("仅宗主可设置职位")
	}

	_, _ = s.memberColl().UpdateOne(ctx,
		bson.M{"sect_id": sectID, "user_id": targetID},
		bson.M{"$set": bson.M{"rank": newRank}})
	return nil
}

// ============================================================
// 宗门贡献
// ============================================================

// AddContribution 增加成员贡献
func (s *SectService) AddContribution(ctx context.Context, sectID, userID string, amount int64) (int64, error) {
	result, err := s.memberColl().UpdateOne(ctx,
		bson.M{"sect_id": sectID, "user_id": userID},
		bson.M{"$inc": bson.M{"contribution": amount}},
	)
	if err != nil {
		return 0, err
	}
	if result.MatchedCount == 0 {
		return 0, fmt.Errorf("成员不存在")
	}

	// 增加宗门经验(贡献额的10%)
	expGain := int64(float64(amount) * 0.1)
	s.addSectExperience(ctx, sectID, expGain)

	return amount, nil
}

// GetContributionRank 获取宗门贡献排行
func (s *SectService) GetContributionRank(ctx context.Context, sectID string, limit int64) ([]*model.SectMember, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "contribution", Value: -1}}).
		SetLimit(limit)
	cursor, err := s.memberColl().Find(ctx, bson.M{"sect_id": sectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*model.SectMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// addSectExperience 增加宗门经验并检查升级
func (s *SectService) addSectExperience(ctx context.Context, sectID string, exp int64) {
	sect, err := s.GetSect(ctx, sectID)
	if err != nil {
		return
	}
	sect.Experience += exp

	// 经验升级公式: 每级所需经验 = 1000 * level^1.5
	levelUpExp := int64(1000 * math.Pow(float64(sect.Level), 1.5))
	if sect.Experience >= levelUpExp && sect.Level < 50 {
		sect.Level++
		sect.Experience -= levelUpExp
		_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
			bson.M{"$set": bson.M{
				"level":      sect.Level,
				"experience": sect.Experience,
				"max_members": 20 + (sect.Level-1)*5, // 每级+5成员上限
			}},
		)
	} else {
		_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
			bson.M{"$set": bson.M{"experience": sect.Experience}})
	}
}

// ============================================================
// 宗门技能
// ============================================================

// GetSectSkills 获取宗门技能列表
func (s *SectService) GetSectSkills(ctx context.Context, sectID string) ([]*model.SectSkill, error) {
	cursor, err := s.skillColl().Find(ctx, bson.M{"sect_id": sectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var skills []*model.SectSkill
	if err := cursor.All(ctx, &skills); err != nil {
		return nil, err
	}
	return skills, nil
}

// LearnSkill 学习/升级宗门技能
func (s *SectService) LearnSkill(ctx context.Context, sectID, userID, skillID string) error {
	var skill model.SectSkill
	err := s.skillColl().FindOne(ctx, bson.M{"_id": skillID, "sect_id": sectID}).Decode(&skill)
	if err != nil {
		return fmt.Errorf("技能不存在")
	}

	// 获取成员的当前技能等级
	var memberSkill model.SectMemberSkill
	err = s.memberSkillColl().FindOne(ctx, bson.M{
		"member_id": userID, "skill_id": skillID,
	}).Decode(&memberSkill)

	currentLevel := memberSkill.Level
	if err == mongo.ErrNoDocuments {
		currentLevel = 0
	} else if err != nil {
		return err
	}

	if currentLevel >= skill.MaxLevel {
		return fmt.Errorf("技能已达最高等级")
	}

	newLevel := currentLevel + 1
	cost := skill.CostPerLevel * int64(newLevel)

	// 检查贡献
	var member model.SectMember
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return fmt.Errorf("成员信息不存在")
	}
	if member.Contribution < cost {
		return fmt.Errorf("贡献不足，需要 %d", cost)
	}

	// 扣除贡献并升级技能
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
	return err
}

// GetMemberSkills 获取成员已学技能
func (s *SectService) GetMemberSkills(ctx context.Context, userID string) ([]*model.SectMemberSkill, error) {
	cursor, err := s.memberSkillColl().Find(ctx, bson.M{"member_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var skills []*model.SectMemberSkill
	if err := cursor.All(ctx, &skills); err != nil {
		return nil, err
	}
	return skills, nil
}

// GetMemberSkillBonuses 计算成员技能总加成(按类型聚合)
func (s *SectService) GetMemberSkillBonuses(ctx context.Context, sectID, userID string) map[string]float64 {
	bonuses := make(map[string]float64)

	skills, err := s.GetMemberSkills(ctx, userID)
	if err != nil {
		return bonuses
	}

	for _, ms := range skills {
		var skill model.SectSkill
		err := s.skillColl().FindOne(ctx, bson.M{"_id": ms.SkillID, "sect_id": sectID}).Decode(&skill)
		if err != nil {
			continue
		}
		bonuses[skill.EffectType] += skill.EffectValue * float64(ms.Level)
	}
	return bonuses
}

// ============================================================
// 宗门战(简化设计)
// ============================================================

// SectWar 宗门战记录
type SectWar struct {
	ID          string    `bson:"_id" json:"id"`
	SectA       string    `bson:"sect_a" json:"sect_a"`
	SectB       string    `bson:"sect_b" json:"sect_b"`
	WinnerSect  string    `bson:"winner_sect,omitempty" json:"winner_sect,omitempty"`
	Status      string    `bson:"status" json:"status"` // pending / active / finished
	ScheduledAt time.Time `bson:"scheduled_at" json:"scheduled_at"`
	FinishedAt  time.Time `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
}

// DeclareWar 宣战
func (s *SectService) DeclareWar(ctx context.Context, sectAID, sectBID, operatorID string) (*SectWar, error) {
	// 仅宗主可宣战
	rank, err := s.getMemberRank(ctx, sectAID, operatorID)
	if err != nil || rank != model.SectLeader {
		return nil, fmt.Errorf("仅宗主可宣战")
	}

	war := &SectWar{
		ID:          uuid.New().String(),
		SectA:       sectAID,
		SectB:       sectBID,
		Status:      "pending",
		ScheduledAt: time.Now().Add(24 * time.Hour), // 24小时后开战
	}
	// 存储到 sect_wars 集合
	_, err = s.db.Collection("sect_wars").InsertOne(ctx, war)
	if err != nil {
		return nil, err
	}
	return war, nil
}

// ============================================================
// 辅助方法
// ============================================================

func (s *SectService) getMember(ctx context.Context, sectID, userID string) (*model.SectMember, error) {
	var member model.SectMember
	err := s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	return &member, nil
}

func (s *SectService) getMemberRank(ctx context.Context, sectID, userID string) (model.SectRank, error) {
	member, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return "", err
	}
	return member.Rank, nil
}

func rankWeight(rank model.SectRank) int {
	switch rank {
	case model.SectLeader:
		return 4
	case model.SectElder:
		return 3
	case model.SectElite:
		return 2
	default:
		return 1
	}
}
