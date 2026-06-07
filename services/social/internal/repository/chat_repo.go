// Package repository 提供 MongoDB 数据访问层
package repository

import (
	"context"
	"time"

	"cultivation-game/services/social/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatRepo 聊天消息仓储
type ChatRepo struct {
	collection *mongo.Collection
}

// NewChatRepo 创建聊天仓储
func NewChatRepo(db *mongo.Database) *ChatRepo {
	return &ChatRepo{
		collection: db.Collection("chat_messages"),
	}
}

// Insert 插入一条聊天消息
func (r *ChatRepo) Insert(ctx context.Context, msg *model.ChatMessage) error {
	msg.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, msg)
	return err
}

// FindByChannel 按频道查询消息，按时间倒序
func (r *ChatRepo) FindByChannel(ctx context.Context, channel model.ChatChannel, targetID string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	filter := bson.M{"channel": channel}
	if channel == model.ChannelSect && targetID != "" {
		filter["target_id"] = targetID
	}
	if !before.IsZero() {
		filter["created_at"] = bson.M{"$lt": before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*model.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	// 反转顺序，让最早的在前
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// FindPrivateMessages 查询私聊历史
func (r *ChatRepo) FindPrivateMessages(ctx context.Context, userA, userB string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	filter := bson.M{
		"channel": model.ChannelPrivate,
		"$or": []bson.M{
			{"sender_id": userA, "target_id": userB},
			{"sender_id": userB, "target_id": userA},
		},
	}
	if !before.IsZero() {
		filter["created_at"] = bson.M{"$lt": before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*model.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}
