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

// DaoLvRepo 道侣数据仓储
type DaoLvRepo struct {
	relationColl *mongo.Collection
	proposalColl *mongo.Collection
	taskColl     *mongo.Collection
}

// NewDaoLvRepo 创建道侣仓储
func NewDaoLvRepo(db *mongo.Database) *DaoLvRepo {
	return &DaoLvRepo{
		relationColl: db.Collection("daolv_relations"),
		proposalColl: db.Collection("daolv_proposals"),
		taskColl:     db.Collection("daolv_tasks"),
	}
}

// ============================================================
// 道侣关系 CRUD
// ============================================================

// InsertRelation 插入道侣关系
func (r *DaoLvRepo) InsertRelation(ctx context.Context, rel *model.DaolvRelation) error {
	now := time.Now()
	rel.StartedAt = now
	rel.UpdatedAt = now
	_, err := r.relationColl.InsertOne(ctx, rel)
	return err
}

// FindRelationByPlayer 查找玩家当前的道侣关系(仅 normal)
func (r *DaoLvRepo) FindRelationByPlayer(ctx context.Context, playerID uint64) (*model.DaolvRelation, error) {
	var rel model.DaolvRelation
	filter := bson.M{
		"$or": []bson.M{
			{"player_a": playerID},
			{"player_b": playerID},
		},
		"status": "normal",
	}
	err := r.relationColl.FindOne(ctx, filter).Decode(&rel)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

// FindRelationByID 按 ID 查找道侣关系
func (r *DaoLvRepo) FindRelationByID(ctx context.Context, id string) (*model.DaolvRelation, error) {
	var rel model.DaolvRelation
	err := r.relationColl.FindOne(ctx, bson.M{"_id": id}).Decode(&rel)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

// FindRelationByPlayers 查找两名玩家之间的道侣关系
func (r *DaoLvRepo) FindRelationByPlayers(ctx context.Context, a, b uint64) (*model.DaolvRelation, error) {
	var rel model.DaolvRelation
	filter := bson.M{
		"$or": []bson.M{
			{"player_a": a, "player_b": b},
			{"player_a": b, "player_b": a},
		},
	}
	err := r.relationColl.FindOne(ctx, filter).Decode(&rel)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

// UpdateRelation 更新道侣关系(合并set与inc操作)
func (r *DaoLvRepo) UpdateRelation(ctx context.Context, id string, update bson.M) error {
	// Set updated_at automatically
	if set, ok := update["$set"]; ok {
		if setMap, ok2 := set.(bson.M); ok2 {
			setMap["updated_at"] = time.Now()
		}
	} else {
		update["$set"] = bson.M{"updated_at": time.Now()}
	}
	_, err := r.relationColl.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SetRelationField 设置单个字段
func (r *DaoLvRepo) SetRelationField(ctx context.Context, id, field string, value interface{}) error {
	return r.UpdateRelation(ctx, id, bson.M{"$set": bson.M{field: value}})
}

// IsPlayerInRelation 检查玩家是否已有道侣关系
func (r *DaoLvRepo) IsPlayerInRelation(ctx context.Context, playerID uint64) (bool, error) {
	count, err := r.relationColl.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"player_a": playerID},
			{"player_b": playerID},
		},
		"status": "normal",
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetPartnerID 获取道侣的另一方ID
func (r *DaoLvRepo) GetPartnerID(rel *model.DaolvRelation, playerID uint64) uint64 {
	if rel.PlayerA == playerID {
		return rel.PlayerB
	}
	return rel.PlayerA
}

// ============================================================
// 道侣申请 CRUD
// ============================================================

// InsertProposal 插入道侣申请
func (r *DaoLvRepo) InsertProposal(ctx context.Context, p *model.DaolvProposal) error {
	p.CreatedAt = time.Now()
	_, err := r.proposalColl.InsertOne(ctx, p)
	return err
}

// FindProposalByID 按 ID 查找申请
func (r *DaoLvRepo) FindProposalByID(ctx context.Context, id string) (*model.DaolvProposal, error) {
	var p model.DaolvProposal
	err := r.proposalColl.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProposalStatus 更新申请状态并记录处理时间
func (r *DaoLvRepo) UpdateProposalStatus(ctx context.Context, id, status string) error {
	_, err := r.proposalColl.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"status":    status,
			"handled_at": time.Now(),
		}},
	)
	return err
}

// FindPendingProposal 查找两名玩家之间的待处理申请
func (r *DaoLvRepo) FindPendingProposal(ctx context.Context, fromID, toID uint64) (*model.DaolvProposal, error) {
	var p model.DaolvProposal
	filter := bson.M{
		"$or": []bson.M{
			{"from_id": fromID, "to_id": toID},
			{"from_id": toID, "to_id": fromID},
		},
		"status": "pending",
	}
	err := r.proposalColl.FindOne(ctx, filter).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindProposalsByPlayer 查询玩家相关的所有申请(发出+收到)
func (r *DaoLvRepo) FindProposalsByPlayer(ctx context.Context, playerID uint64) ([]*model.DaolvProposal, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"from_id": playerID},
			{"to_id": playerID},
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.proposalColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var proposals []*model.DaolvProposal
	if err := cursor.All(ctx, &proposals); err != nil {
		return nil, err
	}
	return proposals, nil
}

// FindPendingProposalsByTarget 查询目标玩家的待处理申请
func (r *DaoLvRepo) FindPendingProposalsByTarget(ctx context.Context, toID uint64) ([]*model.DaolvProposal, error) {
	filter := bson.M{
		"to_id":  toID,
		"status": "pending",
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.proposalColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var proposals []*model.DaolvProposal
	if err := cursor.All(ctx, &proposals); err != nil {
		return nil, err
	}
	return proposals, nil
}

// FindRecentRejectedProposal 查找最近被拒绝的申请(用于冷却检查)
func (r *DaoLvRepo) FindRecentRejectedProposal(ctx context.Context, fromID uint64, since time.Time) (*model.DaolvProposal, error) {
	var p model.DaolvProposal
	filter := bson.M{
		"from_id":    fromID,
		"status":     "rejected",
		"handled_at": bson.M{"$gte": since},
	}
	err := r.proposalColl.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "handled_at", Value: -1}})).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ============================================================
// 道侣任务 CRUD
// ============================================================

// InsertTask 插入任务
func (r *DaoLvRepo) InsertTask(ctx context.Context, task *model.DaolvTask) error {
	task.CreatedAt = time.Now()
	_, err := r.taskColl.InsertOne(ctx, task)
	return err
}

// InsertTasks 批量插入任务
func (r *DaoLvRepo) InsertTasks(ctx context.Context, tasks []*model.DaolvTask) error {
	now := time.Now()
	docs := make([]interface{}, len(tasks))
	for i, t := range tasks {
		t.CreatedAt = now
		docs[i] = t
	}
	_, err := r.taskColl.InsertMany(ctx, docs)
	return err
}

// FindTasksByRelation 查找道侣关系的任务
func (r *DaoLvRepo) FindTasksByRelation(ctx context.Context, relationID string, period string) ([]*model.DaolvTask, error) {
	filter := bson.M{
		"relation_id": relationID,
	}
	if period != "" {
		filter["period"] = period
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.taskColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*model.DaolvTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// FindTaskByID 按ID查找任务
func (r *DaoLvRepo) FindTaskByID(ctx context.Context, id string) (*model.DaolvTask, error) {
	var t model.DaolvTask
	err := r.taskColl.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTask 更新任务
func (r *DaoLvRepo) UpdateTask(ctx context.Context, id string, update bson.M) error {
	_, err := r.taskColl.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

// UpdateTaskProgress 更新任务进度
func (r *DaoLvRepo) UpdateTaskProgress(ctx context.Context, id string, progress int64, completed bool) error {
	_, err := r.taskColl.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"progress":  progress,
			"completed": completed,
		},
	})
	return err
}

// FindTasksByPlayer 通过玩家ID查找其道侣关系的任务
func (r *DaoLvRepo) FindTasksByPlayer(ctx context.Context, playerID uint64) ([]*model.DaolvTask, error) {
	// 先找到玩家的道侣关系
	rel, err := r.FindRelationByPlayer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return r.FindTasksByRelation(ctx, rel.ID, "")
}

// DeleteTasksByRelation 删除道侣关系的所有任务
func (r *DaoLvRepo) DeleteTasksByRelation(ctx context.Context, relationID string) error {
	_, err := r.taskColl.DeleteMany(ctx, bson.M{"relation_id": relationID})
	return err
}
