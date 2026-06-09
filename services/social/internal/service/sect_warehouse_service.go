// Package service 宗门仓库系统
//
// 成员捐献物品获得贡献(按市场价)，其他成员用灵石或贡献购买
// 灵石购买 → 灵石进宗门资金，贡献购买 → 贡献扣除
// 宗门资金 = 成员捐献 + 仓库物品销售收入
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

// WarehouseItem 仓库物品
type WarehouseItem struct {
	ID               string    `bson:"_id" json:"id"`
	SectID           string    `bson:"sect_id" json:"sect_id"`
	DonorID          string    `bson:"donor_id" json:"donor_id"`
	DonorName        string    `bson:"donor_name" json:"donor_name"`
	ItemName         string    `bson:"item_name" json:"item_name"`
	ItemType         string    `bson:"item_type" json:"item_type"` // weapon/robe/headgear/boots/necklace/ring/pill/material/other
	ItemIcon         string    `bson:"item_icon" json:"item_icon"`
	Quantity         int       `bson:"quantity" json:"quantity"`
	PriceSpirit      int64     `bson:"price_spirit" json:"price_spirit"`           // 灵石价格
	PriceContribution int64    `bson:"price_contribution" json:"price_contribution"` // 贡献价格
	MarketValue      int64     `bson:"market_value" json:"market_value"`           // 市场价值(捐献时获得贡献)
	Status           string    `bson:"status" json:"status"`                       // available / sold
	CreatedAt        time.Time `bson:"created_at" json:"created_at"`
}

// SectWarehouseService 宗门仓库业务
type SectWarehouseService struct {
	db *mongo.Database
}

// NewSectWarehouseService 创建宗门仓库服务
func NewSectWarehouseService(db *mongo.Database) *SectWarehouseService {
	return &SectWarehouseService{db: db}
}

func (s *SectWarehouseService) whColl() *mongo.Collection   { return s.db.Collection("sect_warehouse") }
func (s *SectWarehouseService) memberColl() *mongo.Collection { return s.db.Collection("sect_members") }
func (s *SectWarehouseService) sectColl() *mongo.Collection   { return s.db.Collection("sects") }

// DonateItem 捐献物品到仓库
// 捐献者获得与市场价等值的宗门贡献
// POST /api/v1/sect/warehouse/donate
func (s *SectWarehouseService) DonateItem(ctx context.Context, sectID, userID, userName, itemName, itemType, itemIcon string, quantity int, marketValue int64) (*WarehouseItem, error) {
	// 验证成员身份
	member, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return nil, err
	}

	item := &WarehouseItem{
		ID:                uuid.New().String(),
		SectID:            sectID,
		DonorID:           userID,
		DonorName:         userName,
		ItemName:          itemName,
		ItemType:          itemType,
		ItemIcon:          itemIcon,
		Quantity:          quantity,
		PriceSpirit:       marketValue,           // 灵石售价 = 市场价
		PriceContribution: marketValue / 2,        // 贡献售价 = 市场价的一半
		MarketValue:       marketValue,
		Status:            "available",
		CreatedAt:         time.Now(),
	}

	// 事务: 插入物品 + 增加捐献者贡献
		if _, err := s.whColl().InsertOne(ctx, item); err != nil {
			return nil, err
		}
		// 捐献者获得等值贡献
		contributionGain := marketValue * int64(quantity)
		if _, err := s.memberColl().UpdateOne(ctx,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": contributionGain}},
		); err != nil {
			return nil, err
		}

	// 增加宗门经验(捐献额的10%)
	_ = member // member already validated above
	s.addSectExp(ctx, sectID, marketValue*int64(quantity)/10)

	return item, nil
}

// GetWarehouseItems 获取仓库物品列表
// GET /api/v1/sect/warehouse/list?sect_id=xxx
func (s *SectWarehouseService) GetWarehouseItems(ctx context.Context, sectID string, page, pageSize int64) ([]*WarehouseItem, int64, error) {
	filter := bson.M{"sect_id": sectID, "status": "available"}
	total, err := s.whColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().SetSkip(skip).SetLimit(pageSize).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.whColl().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*WarehouseItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// BuyItem 购买仓库物品
// currency: "spirit" 用灵石购买 / "contribution" 用贡献购买
// POST /api/v1/sect/warehouse/buy
func (s *SectWarehouseService) BuyItem(ctx context.Context, itemID, sectID, userID, currency string) error {
	// 查找物品
	var item WarehouseItem
	err := s.whColl().FindOne(ctx, bson.M{"_id": itemID, "sect_id": sectID, "status": "available"}).Decode(&item)
	if err != nil {
		return fmt.Errorf("物品不存在或已售出")
	}

	// 不能买自己捐献的
	if item.DonorID == userID {
		return fmt.Errorf("不能购买自己捐献的物品")
	}

	// 查找购买者
	buyer, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return err
	}

		switch currency {
		case "spirit":
			// 灵石购买: 灵石进入宗门资金
			cost := item.PriceSpirit * int64(item.Quantity)
			// 这里应该调用player服务扣灵石，简化处理：增加宗门资金
			if _, err := s.sectColl().UpdateOne(ctx,
				bson.M{"_id": sectID},
				bson.M{"$inc": bson.M{"funds": cost}},
			); err != nil {
				return err
			}
			// 扣除购买者的灵石(由调用方通过player服务处理)
		case "contribution":
			// 贡献购买: 扣除买家贡献
			cost := item.PriceContribution * int64(item.Quantity)
			if buyer.Contribution < cost {
				return fmt.Errorf("贡献不足，需要 %d 贡献，当前 %d", cost, buyer.Contribution)
			}
			if _, err := s.memberColl().UpdateOne(ctx,
				bson.M{"sect_id": sectID, "user_id": userID},
				bson.M{"$inc": bson.M{"contribution": -cost}},
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("不支持的货币类型: %s", currency)
		}

		// 标记物品为已售出(或减少数量)
		if item.Quantity <= 1 {
			if _, err := s.whColl().UpdateOne(ctx,
				bson.M{"_id": itemID},
				bson.M{"$set": bson.M{"status": "sold"}},
			); err != nil {
				return err
			}
		} else {
			if _, err := s.whColl().UpdateOne(ctx,
				bson.M{"_id": itemID},
				bson.M{"$inc": bson.M{"quantity": -1}},
			); err != nil {
				return err
			}
		}

	return nil
}

// DonateFunds 捐献灵石给宗门
// POST /api/v1/sect/warehouse/donate-funds
func (s *SectWarehouseService) DonateFunds(ctx context.Context, sectID, userID string, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("捐献金额必须大于0")
	}

	// 验证成员
	_, err := s.getMember(ctx, sectID, userID)
	if err != nil {
		return 0, err
	}

		// 增加宗门资金
		if _, err := s.sectColl().UpdateOne(ctx,
			bson.M{"_id": sectID},
			bson.M{"$inc": bson.M{"funds": amount}},
		); err != nil {
			return 0, err
		}
		// 捐献者获得贡献(10%的灵石价值)
		contribution := amount / 10
		if _, err := s.memberColl().UpdateOne(ctx,
			bson.M{"sect_id": sectID, "user_id": userID},
			bson.M{"$inc": bson.M{"contribution": contribution}},
		); err != nil {
			return 0, err
		}

	s.addSectExp(ctx, sectID, amount/10)
	return amount, nil
}

func (s *SectWarehouseService) getMember(ctx context.Context, sectID, userID string) (*SectMember, error) {
	var member SectMember
	err := s.memberColl().FindOne(ctx, bson.M{"sect_id": sectID, "user_id": userID}).Decode(&member)
	if err != nil {
		return nil, fmt.Errorf("成员不存在")
	}
	return &member, nil
}

// SectMember 仓库专用的简化成员结构
type SectMember struct {
	UserID       string `bson:"user_id" json:"user_id"`
	Contribution int64  `bson:"contribution" json:"contribution"`
	Rank         string `bson:"rank" json:"rank"`
}

func (s *SectWarehouseService) addSectExp(ctx context.Context, sectID string, exp int64) {
	_, _ = s.sectColl().UpdateOne(ctx, bson.M{"_id": sectID},
		bson.M{"$inc": bson.M{"experience": exp}})
}
