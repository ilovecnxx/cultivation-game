// Package repository 社交系统 MySQL 数据访问层 — 替代 MongoDB
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"cultivation-game/services/social/internal/model"
)

// SocialMySQLStore 社交系统统一 MySQL 存储
type SocialMySQLStore struct {
	db *sql.DB
}

// NewSocialMySQLStore 创建社交 MySQL 存储
func NewSocialMySQLStore(db *sql.DB) *SocialMySQLStore {
	return &SocialMySQLStore{db: db}
}

// ============================================================
// 聊天消息
// ============================================================

// InsertChatMessage 插入聊天消息
func (s *SocialMySQLStore) InsertChatMessage(ctx context.Context, msg *model.ChatMessage) error {
	msg.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages (channel, sender_id, sender_name, target_id, content, is_system, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.Channel, msg.SenderID, msg.SenderName, msg.TargetID, msg.Content, msg.IsSystem, msg.CreatedAt)
	return err
}

// FindChatByChannel 按频道查询消息
func (s *SocialMySQLStore) FindChatByChannel(ctx context.Context, channel model.ChatChannel, targetID string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	query := `SELECT id, channel, sender_id, sender_name, target_id, content, is_system, created_at
		FROM chat_messages WHERE channel = ?`
	args := []interface{}{channel}

	if channel == model.ChannelSect && targetID != "" {
		query += " AND target_id = ?"
		args = append(args, targetID)
	}
	if !before.IsZero() {
		query += " AND created_at < ?"
		args = append(args, before)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*model.ChatMessage
	for rows.Next() {
		m := &model.ChatMessage{}
		var id int64
		if err := rows.Scan(&id, &m.Channel, &m.SenderID, &m.SenderName, &m.TargetID, &m.Content, &m.IsSystem, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.ID = fmt.Sprintf("%d", id)
		msgs = append(msgs, m)
	}
	// 反转: 最早在前
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

// FindPrivateMessages 查询私聊历史
func (s *SocialMySQLStore) FindPrivateMessages(ctx context.Context, userA, userB string, limit int64, before time.Time) ([]*model.ChatMessage, error) {
	query := `SELECT id, channel, sender_id, sender_name, target_id, content, is_system, created_at
		FROM chat_messages WHERE channel = 'private' AND ((sender_id = ? AND target_id = ?) OR (sender_id = ? AND target_id = ?))`
	args := []interface{}{userA, userB, userB, userA}
	if !before.IsZero() {
		query += " AND created_at < ?"
		args = append(args, before)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*model.ChatMessage
	for rows.Next() {
		m := &model.ChatMessage{}
		var id int64
		if err := rows.Scan(&id, &m.Channel, &m.SenderID, &m.SenderName, &m.TargetID, &m.Content, &m.IsSystem, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.ID = fmt.Sprintf("%d", id)
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

// ============================================================
// 邮件
// ============================================================

// InsertMail 发送邮件
func (s *SocialMySQLStore) InsertMail(ctx context.Context, mail *model.Mail) error {
	mail.CreatedAt = time.Now()
	attachmentsJSON, _ := json.Marshal(mail.Attachments)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO social_mail (id, mail_type, title, content, sender_id, sender_name, receiver_id, attachments, is_read, is_claimed, created_at, expire_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mail.ID, mail.MailType, mail.Title, mail.Content, mail.SenderID, mail.SenderName, mail.ReceiverID,
		string(attachmentsJSON), mail.IsRead, mail.IsClaimed, mail.CreatedAt, toNullTime(mail.ExpireAt))
	return err
}

// FindMailByReceiver 分页查收件箱
func (s *SocialMySQLStore) FindMailByReceiver(ctx context.Context, receiverID string, page, pageSize int64) ([]*model.Mail, int64, error) {
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM social_mail WHERE receiver_id = ?`, receiverID).Scan(&total); err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, mail_type, title, content, sender_id, sender_name, receiver_id, attachments, is_read, is_claimed, created_at, expire_at
		FROM social_mail WHERE receiver_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, receiverID, pageSize, skip)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var mails []*model.Mail
	for rows.Next() {
		m := &model.Mail{}
		var attJSON string
		var expireAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.MailType, &m.Title, &m.Content, &m.SenderID, &m.SenderName, &m.ReceiverID,
			&attJSON, &m.IsRead, &m.IsClaimed, &m.CreatedAt, &expireAt); err != nil {
			return nil, 0, err
		}
		if attJSON != "" {
			json.Unmarshal([]byte(attJSON), &m.Attachments)
		}
		if expireAt.Valid {
			m.ExpireAt = expireAt.Time
		}
		mails = append(mails, m)
	}
	return mails, total, rows.Err()
}

// UpdateMailRead 标记已读
func (s *SocialMySQLStore) UpdateMailRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE social_mail SET is_read = 1 WHERE id = ?`, id)
	return err
}

// UpdateMailClaimed 标记附件已领取
func (s *SocialMySQLStore) UpdateMailClaimed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE social_mail SET is_claimed = 1 WHERE id = ?`, id)
	return err
}

// FindMailByID 按ID查找邮件
func (s *SocialMySQLStore) FindMailByID(ctx context.Context, id string) (*model.Mail, error) {
	m := &model.Mail{}
	var attJSON string
	var expireAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, mail_type, title, content, sender_id, sender_name, receiver_id, attachments, is_read, is_claimed, created_at, expire_at
		FROM social_mail WHERE id = ?`, id).Scan(
		&m.ID, &m.MailType, &m.Title, &m.Content, &m.SenderID, &m.SenderName, &m.ReceiverID,
		&attJSON, &m.IsRead, &m.IsClaimed, &m.CreatedAt, &expireAt)
	if err != nil {
		return nil, err
	}
	if attJSON != "" {
		json.Unmarshal([]byte(attJSON), &m.Attachments)
	}
	if expireAt.Valid {
		m.ExpireAt = expireAt.Time
	}
	return m, nil
}

// ============================================================
// 好友
// ============================================================

// UpsertFriend 添加或更新好友关系
func (s *SocialMySQLStore) UpsertFriend(ctx context.Context, userID, friendID string, status model.FriendStatus) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO friends (player_id, friend_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE status = VALUES(status), updated_at = VALUES(updated_at)`,
		userID, friendID, status, now, now)
	return err
}

// DeleteFriend 删除好友
func (s *SocialMySQLStore) DeleteFriend(ctx context.Context, userID, friendID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM friends WHERE player_id = ? AND friend_id = ?`, userID, friendID)
	return err
}

// ============================================================
// 道侣关系
// ============================================================

// InsertDaolvRelation 插入道侣关系
func (s *SocialMySQLStore) InsertDaolvRelation(ctx context.Context, rel *model.DaolvRelation) error {
	skillsJSON, _ := json.Marshal(rel.Skills)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO daolv_relations (id, player_a, player_b, intimacy, compatibility, level, skills,
		 daily_cultivated, daily_cultivate_date, gift_item_a, gift_item_b, last_propose_at, started_at, updated_at, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rel.ID, rel.PlayerA, rel.PlayerB, rel.Intimacy, rel.Compatibility, rel.Level, string(skillsJSON),
		rel.DailyCultivated, rel.DailyCultivateDate, rel.GiftItemA, rel.GiftItemB,
		toNullTime2(rel.LastProposeAt), rel.StartedAt, rel.UpdatedAt, rel.Status)
	return err
}

// FindDaolvRelation 查找玩家道侣关系
func (s *SocialMySQLStore) FindDaolvRelation(ctx context.Context, playerID uint64) (*model.DaolvRelation, error) {
	rel := &model.DaolvRelation{}
	var skillsJSON string
	var lastProposeAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, player_a, player_b, intimacy, compatibility, level, skills,
		 daily_cultivated, daily_cultivate_date, gift_item_a, gift_item_b, last_propose_at, started_at, updated_at, status
		FROM daolv_relations WHERE (player_a = ? OR player_b = ?) AND status = 'normal'`, playerID, playerID,
	).Scan(&rel.ID, &rel.PlayerA, &rel.PlayerB, &rel.Intimacy, &rel.Compatibility, &rel.Level, &skillsJSON,
		&rel.DailyCultivated, &rel.DailyCultivateDate, &rel.GiftItemA, &rel.GiftItemB,
		&lastProposeAt, &rel.StartedAt, &rel.UpdatedAt, &rel.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if skillsJSON != "" {
		json.Unmarshal([]byte(skillsJSON), &rel.Skills)
	}
	if lastProposeAt.Valid {
		rel.LastProposeAt = lastProposeAt.Time
	}
	return rel, nil
}

// UpdateDaolvRelation 更新道侣关系字段
func (s *SocialMySQLStore) UpdateDaolvRelation(ctx context.Context, id, field string, value interface{}) error {
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE daolv_relations SET %s = ?, updated_at = NOW() WHERE id = ?", field), value, id)
	return err
}

// InsertDaolvProposal 插入道侣申请
func (s *SocialMySQLStore) InsertDaolvProposal(ctx context.Context, p *model.DaolvProposal) error {
	p.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO daolv_proposals (id, from_id, from_name, to_id, to_name, message, gift_item_id, gift_item_name, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.FromID, p.FromName, p.ToID, p.ToName, p.Message, p.GiftItemID, p.GiftItemName, p.Status, p.CreatedAt)
	return err
}

// ============================================================
// 宗门
// ============================================================

// UpsertSect 插入或更新宗门
func (s *SocialMySQLStore) UpsertSect(ctx context.Context, sect *model.Sect) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sects (name, level, experience, member_count, max_members, leader_id, notice, created_at)
		 VALUES (?, 1, 0, 1, 50, ?, ?, NOW())`,
		sect.Name, sect.LeaderID, sect.Notice)
	return err
}

// FindSectByID 按ID查找宗门
func (s *SocialMySQLStore) FindSectByID(ctx context.Context, id string) (*model.Sect, error) {
	sect := &model.Sect{}
	var notice sql.NullString
	var territoryID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, level, experience, funds, member_count, max_members, leader_id, notice, created_at, reputation
		FROM sects WHERE id = ?`, id).Scan(
		&sect.ID, &sect.Name, &sect.Level, &sect.Experience, &sect.Funds, &sect.MemberCount, &sect.MaxMembers,
		&sect.LeaderID, &notice, &sect.CreatedAt, &sect.Reputation)
	if err != nil {
		return nil, err
	}
	if notice.Valid {
		sect.Notice = notice.String
	}
	return sect, nil
}

// ============================================================
// 辅助
// ============================================================

func toNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func toNullTime2(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
