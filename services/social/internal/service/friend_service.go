package service

import (
	"context"
	"fmt"
	"time"

	"cultivation-game/services/social/internal/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FriendService 好友业务逻辑
type FriendService struct {
	db    *mongo.Database
	redis *redis.Client
}

// NewFriendService 创建好友服务
func NewFriendService(db *mongo.Database, rdb *redis.Client) *FriendService {
	return &FriendService{
		db:    db,
		redis: rdb,
	}
}

// friendColl 好友集合
func (s *FriendService) friendColl() *mongo.Collection {
	return s.db.Collection("friends")
}

// applyColl 好友申请集合
func (s *FriendService) applyColl() *mongo.Collection {
	return s.db.Collection("friend_applies")
}

// ============================================================
// 好友管理
// ============================================================

// GetFriendList 获取好友列表
func (s *FriendService) GetFriendList(ctx context.Context, userID string) ([]*model.Friend, error) {
	cursor, err := s.friendColl().Find(ctx, bson.M{
		"user_id": userID,
		"status":  model.FriendStatusNormal,
	})
	if err != nil {
		return nil, fmt.Errorf("查询好友列表失败: %w", err)
	}
	defer cursor.Close(ctx)

	var friends []*model.Friend
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, err
	}
	return friends, nil
}

// GetBlacklist 获取黑名单
func (s *FriendService) GetBlacklist(ctx context.Context, userID string) ([]*model.Friend, error) {
	cursor, err := s.friendColl().Find(ctx, bson.M{
		"user_id": userID,
		"status":  model.FriendStatusBlacked,
	})
	if err != nil {
		return nil, fmt.Errorf("查询黑名单失败: %w", err)
	}
	defer cursor.Close(ctx)

	var friends []*model.Friend
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, err
	}
	return friends, nil
}

// AddFriend 添加好友(需要先有通过的好友申请)
func (s *FriendService) AddFriend(ctx context.Context, userID, friendID, remark string) error {
	now := time.Now()

	// 双向添加好友关系
	for _, pair := range [][2]string{{userID, friendID}, {friendID, userID}} {
		_, err := s.friendColl().UpdateOne(
			ctx,
			bson.M{"user_id": pair[0], "friend_id": pair[1]},
			bson.M{
				"$set": bson.M{
					"user_id":    pair[0],
					"friend_id":  pair[1],
					"status":     model.FriendStatusNormal,
					"created_at": now,
					"updated_at": now,
					"remark":     remark,
				},
			},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("添加好友失败: %w", err)
		}
	}
	return nil
}

// RemoveFriend 删除好友
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	// 双向删除
	_, err := s.friendColl().DeleteMany(ctx, bson.M{
		"$or": []bson.M{
			{"user_id": userID, "friend_id": friendID},
			{"user_id": friendID, "friend_id": userID},
		},
	})
	if err != nil {
		return fmt.Errorf("删除好友失败: %w", err)
	}
	return nil
}

// BlockUser 拉黑用户
func (s *FriendService) BlockUser(ctx context.Context, userID, blockID string) error {
	now := time.Now()
	_, err := s.friendColl().UpdateOne(
		ctx,
		bson.M{"user_id": userID, "friend_id": blockID},
		bson.M{
			"$set": bson.M{
				"user_id":    userID,
				"friend_id":  blockID,
				"status":     model.FriendStatusBlacked,
				"updated_at": now,
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// UnblockUser 取消拉黑
func (s *FriendService) UnblockUser(ctx context.Context, userID, blockID string) error {
	_, err := s.friendColl().DeleteOne(ctx, bson.M{
		"user_id":   userID,
		"friend_id": blockID,
		"status":    model.FriendStatusBlacked,
	})
	return err
}

// IsBlocked 检查是否已被拉黑
func (s *FriendService) IsBlocked(ctx context.Context, userID, targetID string) (bool, error) {
	count, err := s.friendColl().CountDocuments(ctx, bson.M{
		"user_id":   targetID,
		"friend_id": userID,
		"status":    model.FriendStatusBlacked,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ============================================================
// 好友申请
// ============================================================

// ApplyFriend 发送好友申请
func (s *FriendService) ApplyFriend(ctx context.Context, fromID, fromName, toID, message string) error {
	// 检查是否已被对方拉黑
	blocked, err := s.IsBlocked(ctx, fromID, toID)
	if err != nil {
		return err
	}
	if blocked {
		return fmt.Errorf("无法发送申请: 你已被对方拉黑")
	}

	apply := &model.FriendApply{
		ID:        uuid.New().String(),
		FromID:    fromID,
		FromName:  fromName,
		ToID:      toID,
		Message:   message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	_, err = s.applyColl().InsertOne(ctx, apply)
	return err
}

// HandleApply 处理好友申请(同意/拒绝)
func (s *FriendService) HandleApply(ctx context.Context, applyID string, accept bool) error {
	status := "rejected"
	if accept {
		status = "accepted"
	}

	var apply model.FriendApply
	err := s.applyColl().FindOneAndUpdate(
		ctx,
		bson.M{"_id": applyID, "status": "pending"},
		bson.M{"$set": bson.M{"status": status, "handled_at": time.Now()}},
	).Decode(&apply)
	if err != nil {
		return fmt.Errorf("处理申请失败: %w", err)
	}

	// 如果同意，添加好友关系
	if accept {
		return s.AddFriend(ctx, apply.FromID, apply.ToID, "")
	}
	return nil
}

// GetPendingApplies 获取待处理的申请列表
func (s *FriendService) GetPendingApplies(ctx context.Context, userID string) ([]*model.FriendApply, error) {
	cursor, err := s.applyColl().Find(ctx, bson.M{
		"to_id":  userID,
		"status": "pending",
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var applies []*model.FriendApply
	if err := cursor.All(ctx, &applies); err != nil {
		return nil, err
	}
	return applies, nil
}
