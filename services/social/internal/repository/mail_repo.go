package repository

import (
	"context"
	"time"

	"cultivation-game/services/social/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MailRepo 邮件仓储
type MailRepo struct {
	collection *mongo.Collection
}

// NewMailRepo 创建邮件仓储
func NewMailRepo(db *mongo.Database) *MailRepo {
	return &MailRepo{
		collection: db.Collection("mails"),
	}
}

// Insert 发送邮件
func (r *MailRepo) Insert(ctx context.Context, mail *model.Mail) error {
	mail.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, mail)
	return err
}

// FindByReceiver 查询收件箱，按时间倒序
func (r *MailRepo) FindByReceiver(ctx context.Context, receiverID string, page, pageSize int64) ([]*model.Mail, int64, error) {
	filter := bson.M{"receiver_id": receiverID}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(pageSize)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var mails []*model.Mail
	if err := cursor.All(ctx, &mails); err != nil {
		return nil, 0, err
	}
	return mails, total, nil
}

// FindByID 根据 ID 查找邮件
func (r *MailRepo) FindByID(ctx context.Context, mailID string) (*model.Mail, error) {
	var mail model.Mail
	err := r.collection.FindOne(ctx, bson.M{"_id": mailID}).Decode(&mail)
	if err != nil {
		return nil, err
	}
	return &mail, nil
}

// UpdateReadStatus 更新已读状态
func (r *MailRepo) UpdateReadStatus(ctx context.Context, mailID string, isRead bool) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": mailID},
		bson.M{"$set": bson.M{"is_read": isRead}},
	)
	return err
}

// MarkClaimed 标记附件已领取
func (r *MailRepo) MarkClaimed(ctx context.Context, mailID string) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": mailID},
		bson.M{"$set": bson.M{"is_claimed": true}},
	)
	return err
}

// DeleteByID 删除邮件
func (r *MailRepo) DeleteByID(ctx context.Context, mailID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": mailID})
	return err
}

// DeleteExpired 删除过期邮件(用于定时清理)
func (r *MailRepo) DeleteExpired(ctx context.Context) (int64, error) {
	// expire_at 字段存在且小于当前时间
	filter := bson.M{
		"expire_at": bson.M{
			"$lt": time.Now(),
		},
	}
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// CountUnread 统计未读邮件数
func (r *MailRepo) CountUnread(ctx context.Context, receiverID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"receiver_id": receiverID,
		"is_read":     false,
	})
}
