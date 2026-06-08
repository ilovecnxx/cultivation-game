// Package service 功法阁系统
//
// 功法阁分10个等级，弟子根据修为等级用贡献点兑换对应功法
// 每个境界有4本功法(攻/防/辅/秘)，每本5层
// 功法阁等级 = 宗门等级，决定可兑换的功法境界上限
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Technique 功法定义
type Technique struct {
	ID              string `bson:"_id" json:"id"`
	Name            string `bson:"name" json:"name"`
	Description     string `bson:"description" json:"description"`
	RealmRequired   int    `bson:"realm_required" json:"realm_required"`     // 所需境界(1-10)
	PavilionLevel   int    `bson:"pavilion_level" json:"pavilion_level"`     // 功法阁等级要求(1-10)
	Category        string `bson:"category" json:"category"`                 // attack/defense/support/secret
	MaxLevel        int    `bson:"max_level" json:"max_level"`               // 功法最大层数(5)
	CostContribute  int64  `bson:"cost_contribute" json:"cost_contribute"`   // 兑换所需贡献
	EffectType      string `bson:"effect_type" json:"effect_type"`           // 效果类型
	EffectValue     float64 `bson:"effect_value" json:"effect_value"`        // 基础效果值
	EffectPerLevel  float64 `bson:"effect_per_level" json:"effect_per_level"` // 每层递增效果
	Icon            string `bson:"icon" json:"icon"`
}

// MemberTechnique 成员已兑换功法
type MemberTechnique struct {
	ID          string    `bson:"_id" json:"id"`
	MemberID    string    `bson:"member_id" json:"member_id"`
	TechniqueID string    `bson:"technique_id" json:"technique_id"`
	Level       int       `bson:"level" json:"level"`         // 当前层数 1-5
	ObtainedAt  time.Time `bson:"obtained_at" json:"obtained_at"`
}

// SectTechniqueService 功法阁业务
type SectTechniqueService struct {
	db *mongo.Database
}

// NewSectTechniqueService 创建功法阁服务
func NewSectTechniqueService(db *mongo.Database) *SectTechniqueService {
	return &SectTechniqueService{db: db}
}

func (s *SectTechniqueService) techColl() *mongo.Collection        { return s.db.Collection("sect_techniques") }
func (s *SectTechniqueService) memberTechColl() *mongo.Collection  { return s.db.Collection("sect_member_techniques") }
func (s *SectTechniqueService) memberColl() *mongo.Collection      { return s.db.Collection("sect_members") }
func (s *SectTechniqueService) sectColl() *mongo.Collection        { return s.db.Collection("sects") }

// 功法模板库 - 每个境界4本功法 x 10境界 = 40本
var techniqueTemplates = []Technique{
	// ====== 锻体期(1) ======
	{Name: "铁布衫", Description: "外功基础，锤炼血肉之躯", RealmRequired: 1, PavilionLevel: 1, Category: "defense", MaxLevel: 5, CostContribute: 200, EffectType: "defense_bonus", EffectValue: 5, EffectPerLevel: 2, Icon: "🛡️"},
	{Name: "破石拳", Description: "凝力于拳，碎石断金", RealmRequired: 1, PavilionLevel: 1, Category: "attack", MaxLevel: 5, CostContribute: 200, EffectType: "attack_bonus", EffectValue: 5, EffectPerLevel: 2, Icon: "👊"},
	{Name: "吐纳术", Description: "调息养气，固本培元", RealmRequired: 1, PavilionLevel: 1, Category: "support", MaxLevel: 5, CostContribute: 150, EffectType: "hp_bonus", EffectValue: 20, EffectPerLevel: 10, Icon: "💨"},
	{Name: "通脉诀", Description: "打通经脉，加速灵气流转", RealmRequired: 1, PavilionLevel: 1, Category: "secret", MaxLevel: 5, CostContribute: 300, EffectType: "cultivation_speed", EffectValue: 0.05, EffectPerLevel: 0.02, Icon: "🌀"},

	// ====== 练气期(2) ======
	{Name: "金刚体", Description: "引灵气入体，铸金刚之躯", RealmRequired: 2, PavilionLevel: 2, Category: "defense", MaxLevel: 5, CostContribute: 500, EffectType: "defense_bonus", EffectValue: 12, EffectPerLevel: 5, Icon: "💎"},
	{Name: "烈火掌", Description: "火灵化掌，焚尽万物", RealmRequired: 2, PavilionLevel: 2, Category: "attack", MaxLevel: 5, CostContribute: 500, EffectType: "attack_bonus", EffectValue: 12, EffectPerLevel: 5, Icon: "🔥"},
	{Name: "回春术", Description: "灵力滋养，伤愈如初", RealmRequired: 2, PavilionLevel: 2, Category: "support", MaxLevel: 5, CostContribute: 400, EffectType: "hp_regen", EffectValue: 0.5, EffectPerLevel: 0.2, Icon: "🌿"},
	{Name: "气旋诀", Description: "以气化旋，聚灵提速", RealmRequired: 2, PavilionLevel: 2, Category: "secret", MaxLevel: 5, CostContribute: 700, EffectType: "cultivation_speed", EffectValue: 0.08, EffectPerLevel: 0.03, Icon: "🌪️"},

	// ====== 筑基期(3) ======
	{Name: "玄铁甲", Description: "灵气凝铠，刀枪不入", RealmRequired: 3, PavilionLevel: 3, Category: "defense", MaxLevel: 5, CostContribute: 1000, EffectType: "defense_bonus", EffectValue: 25, EffectPerLevel: 10, Icon: "🛡️"},
	{Name: "冰魄剑", Description: "寒气成剑，冰封千里", RealmRequired: 3, PavilionLevel: 3, Category: "attack", MaxLevel: 5, CostContribute: 1000, EffectType: "attack_bonus", EffectValue: 25, EffectPerLevel: 10, Icon: "❄️"},
	{Name: "定神诀", Description: "安神定心，神识清明", RealmRequired: 3, PavilionLevel: 3, Category: "support", MaxLevel: 5, CostContribute: 800, EffectType: "mp_bonus", EffectValue: 50, EffectPerLevel: 25, Icon: "🧠"},
	{Name: "筑基秘法", Description: "上古筑基秘术，突破助力", RealmRequired: 3, PavilionLevel: 3, Category: "secret", MaxLevel: 5, CostContribute: 1500, EffectType: "breakthrough_bonus", EffectValue: 0.05, EffectPerLevel: 0.02, Icon: "📜"},

	// ====== 金丹期(4) ======
	{Name: "金钟罩", Description: "金光护体，万物不侵", RealmRequired: 4, PavilionLevel: 4, Category: "defense", MaxLevel: 5, CostContribute: 2000, EffectType: "defense_bonus", EffectValue: 50, EffectPerLevel: 20, Icon: "🔔"},
	{Name: "天雷引", Description: "引雷入剑，天罚降世", RealmRequired: 4, PavilionLevel: 4, Category: "attack", MaxLevel: 5, CostContribute: 2000, EffectType: "attack_bonus", EffectValue: 50, EffectPerLevel: 20, Icon: "⚡"},
	{Name: "丹元诀", Description: "金丹运转，生生不息", RealmRequired: 4, PavilionLevel: 4, Category: "support", MaxLevel: 5, CostContribute: 1500, EffectType: "all_stats", EffectValue: 0.03, EffectPerLevel: 0.02, Icon: "✨"},
	{Name: "金丹秘术", Description: "金丹九转，造化无穷", RealmRequired: 4, PavilionLevel: 4, Category: "secret", MaxLevel: 5, CostContribute: 3000, EffectType: "crit_damage", EffectValue: 0.1, EffectPerLevel: 0.05, Icon: "💛"},

	// ====== 元婴期(5) ======
	{Name: "不灭金身", Description: "元婴护体，万法不破", RealmRequired: 5, PavilionLevel: 5, Category: "defense", MaxLevel: 5, CostContribute: 4000, EffectType: "defense_bonus", EffectValue: 100, EffectPerLevel: 40, Icon: "🟡"},
	{Name: "焚天诀", Description: "烈焰焚天，神魔辟易", RealmRequired: 5, PavilionLevel: 5, Category: "attack", MaxLevel: 5, CostContribute: 4000, EffectType: "attack_bonus", EffectValue: 100, EffectPerLevel: 40, Icon: "🔥"},
	{Name: "元婴道胎", Description: "元婴滋养，寿元绵长", RealmRequired: 5, PavilionLevel: 5, Category: "support", MaxLevel: 5, CostContribute: 3000, EffectType: "lifespan_bonus", EffectValue: 50, EffectPerLevel: 25, Icon: "👶"},
	{Name: "化婴大法", Description: "元婴出窍，神识暴涨", RealmRequired: 5, PavilionLevel: 5, Category: "secret", MaxLevel: 5, CostContribute: 6000, EffectType: "spirit_sense", EffectValue: 50, EffectPerLevel: 30, Icon: "👁️"},

	// ====== 化神期(6) ======
	{Name: "虚空遁", Description: "化身虚空，万法不沾", RealmRequired: 6, PavilionLevel: 6, Category: "defense", MaxLevel: 5, CostContribute: 8000, EffectType: "dodge_bonus", EffectValue: 0.03, EffectPerLevel: 0.02, Icon: "🌌"},
	{Name: "星辰剑诀", Description: "星辰之力，凝为剑意", RealmRequired: 6, PavilionLevel: 6, Category: "attack", MaxLevel: 5, CostContribute: 8000, EffectType: "attack_bonus", EffectValue: 200, EffectPerLevel: 80, Icon: "⭐"},
	{Name: "化神领域", Description: "神识外放，领域自成", RealmRequired: 6, PavilionLevel: 6, Category: "support", MaxLevel: 5, CostContribute: 6000, EffectType: "all_stats", EffectValue: 0.05, EffectPerLevel: 0.03, Icon: "🌐"},
	{Name: "神游太虚", Description: "神识遨游，洞察先机", RealmRequired: 6, PavilionLevel: 6, Category: "secret", MaxLevel: 5, CostContribute: 12000, EffectType: "crit_rate", EffectValue: 0.03, EffectPerLevel: 0.02, Icon: "🔮"},

	// ====== 炼虚期(7) ======
	{Name: "虚空壁垒", Description: "虚空为盾，万劫不毁", RealmRequired: 7, PavilionLevel: 7, Category: "defense", MaxLevel: 5, CostContribute: 15000, EffectType: "defense_bonus", EffectValue: 400, EffectPerLevel: 150, Icon: "🛡️"},
	{Name: "虚无破灭", Description: "虚无一击，归于混沌", RealmRequired: 7, PavilionLevel: 7, Category: "attack", MaxLevel: 5, CostContribute: 15000, EffectType: "attack_bonus", EffectValue: 400, EffectPerLevel: 150, Icon: "💥"},
	{Name: "炼虚合道", Description: "炼虚归真，道法自然", RealmRequired: 7, PavilionLevel: 7, Category: "support", MaxLevel: 5, CostContribute: 12000, EffectType: "mp_regen", EffectValue: 0.02, EffectPerLevel: 0.01, Icon: "☯️"},
	{Name: "虚空穿梭", Description: "破开虚空，瞬移千里", RealmRequired: 7, PavilionLevel: 7, Category: "secret", MaxLevel: 5, CostContribute: 22000, EffectType: "speed_bonus", EffectValue: 20, EffectPerLevel: 10, Icon: "⚡"},

	// ====== 合体期(8) ======
	{Name: "天地法相", Description: "天地为体，法相自成", RealmRequired: 8, PavilionLevel: 8, Category: "defense", MaxLevel: 5, CostContribute: 30000, EffectType: "defense_bonus", EffectValue: 800, EffectPerLevel: 300, Icon: "🗿"},
	{Name: "乾坤一击", Description: "逆转乾坤，一击灭世", RealmRequired: 8, PavilionLevel: 8, Category: "attack", MaxLevel: 5, CostContribute: 30000, EffectType: "attack_bonus", EffectValue: 800, EffectPerLevel: 300, Icon: "💢"},
	{Name: "万象归元", Description: "万法归一，生生不息", RealmRequired: 8, PavilionLevel: 8, Category: "support", MaxLevel: 5, CostContribute: 25000, EffectType: "all_stats", EffectValue: 0.08, EffectPerLevel: 0.04, Icon: "🔄"},
	{Name: "合体秘典", Description: "天人合一，窥见大道", RealmRequired: 8, PavilionLevel: 8, Category: "secret", MaxLevel: 5, CostContribute: 45000, EffectType: "breakthrough_bonus", EffectValue: 0.08, EffectPerLevel: 0.03, Icon: "📖"},

	// ====== 大乘期(9) ======
	{Name: "大乘金光", Description: "大乘金光护体，金仙难伤", RealmRequired: 9, PavilionLevel: 9, Category: "defense", MaxLevel: 5, CostContribute: 60000, EffectType: "defense_bonus", EffectValue: 1500, EffectPerLevel: 600, Icon: "✨"},
	{Name: "大罗天罚", Description: "引动天罚，灭仙弑神", RealmRequired: 9, PavilionLevel: 9, Category: "attack", MaxLevel: 5, CostContribute: 60000, EffectType: "attack_bonus", EffectValue: 1500, EffectPerLevel: 600, Icon: "⚡"},
	{Name: "大乘道果", Description: "道果初成，法力无边", RealmRequired: 9, PavilionLevel: 9, Category: "support", MaxLevel: 5, CostContribute: 50000, EffectType: "all_stats", EffectValue: 0.12, EffectPerLevel: 0.06, Icon: "🍎"},
	{Name: "大乘秘藏", Description: "大乘秘法，直指飞升", RealmRequired: 9, PavilionLevel: 9, Category: "secret", MaxLevel: 5, CostContribute: 90000, EffectType: "cultivation_speed", EffectValue: 0.2, EffectPerLevel: 0.08, Icon: "📜"},

	// ====== 渡劫期(10) ======
	{Name: "不灭道体", Description: "渡劫不灭，万古长存", RealmRequired: 10, PavilionLevel: 10, Category: "defense", MaxLevel: 5, CostContribute: 120000, EffectType: "defense_bonus", EffectValue: 3000, EffectPerLevel: 1200, Icon: "🌟"},
	{Name: "混沌开天", Description: "混沌之力，开天辟地", RealmRequired: 10, PavilionLevel: 10, Category: "attack", MaxLevel: 5, CostContribute: 120000, EffectType: "attack_bonus", EffectValue: 3000, EffectPerLevel: 1200, Icon: "💫"},
	{Name: "飞升道种", Description: "凝聚道种，为飞升筑基", RealmRequired: 10, PavilionLevel: 10, Category: "support", MaxLevel: 5, CostContribute: 100000, EffectType: "all_stats", EffectValue: 0.18, EffectPerLevel: 0.08, Icon: "🌱"},
	{Name: "飞升秘典", Description: "飞升之道，尽在其中", RealmRequired: 10, PavilionLevel: 10, Category: "secret", MaxLevel: 5, CostContribute: 180000, EffectType: "breakthrough_bonus", EffectValue: 0.15, EffectPerLevel: 0.05, Icon: "📕"},
}

// InitTechniques 初始化功法阁(首次访问时调用)
func (s *SectTechniqueService) InitTechniques(ctx context.Context, sectID string) error {
	// 检查是否已初始化
	count, err := s.techColl().CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // 已初始化，功法全局共享
	}

	docs := make([]interface{}, len(techniqueTemplates))
	for i, t := range techniqueTemplates {
		t.ID = uuid.New().String()
		docs[i] = t
	}
	_, err = s.techColl().InsertMany(ctx, docs)
	return err
}

// GetTechniquesByRealm 获取指定境界可兑换的功法列表
// GET /api/v1/sect/technique/list?sect_id=xxx&realm=3
func (s *SectTechniqueService) GetTechniquesByRealm(ctx context.Context, sectID string, realm int) ([]*Technique, error) {
	// 确保功法已初始化
	_ = s.InitTechniques(ctx, sectID)

	// 获取宗门信息以确定功法阁等级
	var sect struct {
		Level int `bson:"level"`
	}
	err := s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	pavilionLevel := 1
	if err == nil && sect.Level > 0 {
		pavilionLevel = sect.Level
	}
	if pavilionLevel > 10 {
		pavilionLevel = 10
	}

	filter := bson.M{
		"realm_required": bson.M{"$lte": realm},
		"pavilion_level": bson.M{"$lte": pavilionLevel},
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "realm_required", Value: -1},
		{Key: "category", Value: 1},
	})
	cursor, err := s.techColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var techniques []*Technique
	if err := cursor.All(ctx, &techniques); err != nil {
		return nil, err
	}
	return techniques, nil
}

// ExchangeTechnique 兑换功法
// POST /api/v1/sect/technique/exchange
func (s *SectTechniqueService) ExchangeTechnique(ctx context.Context, sectID, userID, techniqueID string) (*MemberTechnique, error) {
	// 查找功法
	var tech Technique
	err := s.techColl().FindOne(ctx, bson.M{"_id": techniqueID}).Decode(&tech)
	if err != nil {
		return nil, fmt.Errorf("功法不存在")
	}

	// 检查功法阁等级
	var sect struct {
		Level int `bson:"level"`
	}
	err = s.sectColl().FindOne(ctx, bson.M{"_id": sectID}).Decode(&sect)
	if err == nil && sect.Level < tech.PavilionLevel {
		return nil, fmt.Errorf("功法阁等级不足(需要%d级，当前%d级)", tech.PavilionLevel, sect.Level)
	}

	// 检查是否已兑换
	existing, _ := s.memberTechColl().CountDocuments(ctx, bson.M{
		"member_id": userID, "technique_id": techniqueID,
	})
	if existing > 0 {
		return nil, fmt.Errorf("已拥有该功法，请前往技能页面升级")
	}

	// 检查贡献
	var member struct {
		Contribution int64 `bson:"contribution"`
	}
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	if member.Contribution < tech.CostContribute {
		return nil, fmt.Errorf("贡献不足，需要 %d 贡献，当前 %d", tech.CostContribute, member.Contribution)
	}

	// 事务: 扣贡献 + 获得功法
	session, err := s.db.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	mt := &MemberTechnique{
		ID:          uuid.New().String(),
		MemberID:    userID,
		TechniqueID: techniqueID,
		Level:       1,
		ObtainedAt:  time.Now(),
	}

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if _, err := s.memberColl().UpdateOne(sc,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": -tech.CostContribute}},
		); err != nil {
			return nil, err
		}
		if _, err := s.memberTechColl().InsertOne(sc, mt); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("兑换失败: %w", err)
	}

	return mt, nil
}

// UpgradeTechnique 升级功法层数
// POST /api/v1/sect/technique/upgrade
func (s *SectTechniqueService) UpgradeTechnique(ctx context.Context, sectID, userID, memberTechID string) (*MemberTechnique, error) {
	var mt MemberTechnique
	err := s.memberTechColl().FindOne(ctx, bson.M{"_id": memberTechID, "member_id": userID}).Decode(&mt)
	if err != nil {
		return nil, fmt.Errorf("功法记录不存在")
	}

	var tech Technique
	err = s.techColl().FindOne(ctx, bson.M{"_id": mt.TechniqueID}).Decode(&tech)
	if err != nil {
		return nil, fmt.Errorf("功法不存在")
	}

	if mt.Level >= tech.MaxLevel {
		return nil, fmt.Errorf("功法已达最高层数(%d层)", tech.MaxLevel)
	}

	// 升级消耗 = 兑换消耗的50% * 新等级
	upgradeCost := tech.CostContribute / 2 * int64(mt.Level+1)

	var member struct {
		Contribution int64 `bson:"contribution"`
	}
	err = s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	if member.Contribution < upgradeCost {
		return nil, fmt.Errorf("贡献不足，需要 %d 贡献", upgradeCost)
	}

	_, _ = s.memberColl().UpdateOne(ctx,
		bson.M{"sect_id": sectID, "user_id": userID},
		bson.M{"$inc": bson.M{"contribution": -upgradeCost}},
	)

	newLevel := mt.Level + 1
	_, _ = s.memberTechColl().UpdateOne(ctx,
		bson.M{"_id": memberTechID},
		bson.M{"$set": bson.M{"level": newLevel}},
	)
	mt.Level = newLevel

	return &mt, nil
}

// GetMyTechniques 获取成员已兑换功法
// GET /api/v1/sect/technique/my?user_id=xxx
func (s *SectTechniqueService) GetMyTechniques(ctx context.Context, userID string) ([]*MemberTechnique, error) {
	cursor, err := s.memberTechColl().Find(ctx, bson.M{"member_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var techniques []*MemberTechnique
	if err := cursor.All(ctx, &techniques); err != nil {
		return nil, err
	}
	return techniques, nil
}
