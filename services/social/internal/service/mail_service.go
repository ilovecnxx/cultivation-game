package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/repository"

	"github.com/google/uuid"
)

// MailService 邮件业务逻辑
type MailService struct {
	repo              *repository.MailRepo
	friendSvc         *FriendService // 用于检查黑名单
	playerServiceAddr string         // Player 服务 HTTP 地址（用于发放附件）
}

// NewMailService 创建邮件服务
func NewMailService(repo *repository.MailRepo, friendSvc *FriendService) *MailService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &MailService{
		repo:              repo,
		friendSvc:         friendSvc,
		playerServiceAddr: playerAddr,
	}
}

// SendSystemMail 发送系统邮件(可带附件)
func (s *MailService) SendSystemMail(ctx context.Context, receiverID, title, content string, attachments []model.MailAttachment, expireDuration time.Duration) (*model.Mail, error) {
	mail := &model.Mail{
		ID:          uuid.New().String(),
		MailType:    model.MailSystem,
		Title:       title,
		Content:     content,
		SenderID:    "system",
		SenderName:  "系统",
		ReceiverID:  receiverID,
		Attachments: attachments,
		IsRead:      false,
		IsClaimed:   false,
		ExpireAt:    time.Now().Add(expireDuration),
	}

	if err := s.repo.Insert(ctx, mail); err != nil {
		return nil, fmt.Errorf("发送系统邮件失败: %w", err)
	}
	return mail, nil
}

// SendPlayerMail 发送玩家邮件
func (s *MailService) SendPlayerMail(ctx context.Context, senderID, senderName, receiverID, title, content string, attachments []model.MailAttachment) (*model.Mail, error) {
	// 检查是否被对方拉黑
	blocked, err := s.friendSvc.IsBlocked(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, fmt.Errorf("无法发送邮件: 你已被对方拉黑")
	}

	mail := &model.Mail{
		ID:          uuid.New().String(),
		MailType:    model.MailPlayer,
		Title:       title,
		Content:     content,
		SenderID:    senderID,
		SenderName:  senderName,
		ReceiverID:  receiverID,
		Attachments: attachments,
		IsRead:      false,
		IsClaimed:   false,
	}

	if err := s.repo.Insert(ctx, mail); err != nil {
		return nil, fmt.Errorf("发送玩家邮件失败: %w", err)
	}
	return mail, nil
}

// GetInbox 获取收件箱
func (s *MailService) GetInbox(ctx context.Context, receiverID string, page, pageSize int64) ([]*model.Mail, int64, error) {
	return s.repo.FindByReceiver(ctx, receiverID, page, pageSize)
}

// ReadMail 阅读邮件(标记已读)
func (s *MailService) ReadMail(ctx context.Context, mailID, userID string) (*model.Mail, error) {
	mail, err := s.repo.FindByID(ctx, mailID)
	if err != nil {
		return nil, fmt.Errorf("邮件不存在: %w", err)
	}
	if mail.ReceiverID != userID {
		return nil, fmt.Errorf("无权访问该邮件")
	}
	if !mail.IsRead {
		if err := s.repo.UpdateReadStatus(ctx, mailID, true); err != nil {
			return nil, err
		}
		mail.IsRead = true
	}
	return mail, nil
}

// ClaimAttachment 领取邮件附件
func (s *MailService) ClaimAttachment(ctx context.Context, mailID, userID string) (*model.Mail, error) {
	mail, err := s.repo.FindByID(ctx, mailID)
	if err != nil {
		return nil, fmt.Errorf("邮件不存在: %w", err)
	}
	if mail.ReceiverID != userID {
		return nil, fmt.Errorf("无权操作该邮件")
	}
	if mail.IsClaimed {
		return nil, fmt.Errorf("附件已领取")
	}
	if len(mail.Attachments) == 0 {
		return nil, fmt.Errorf("该邮件无附件")
	}

	// 发放附件：调用 Player 服务增加物品/货币
	for _, att := range mail.Attachments {
		if att.CoinType != "" && att.CoinAmount > 0 {
			s.grantCurrency(ctx, mail.ReceiverID, att.CoinType, att.CoinAmount)
		}
		if att.ItemID != "" && att.Quantity > 0 {
			s.grantItem(ctx, mail.ReceiverID, att.ItemID, att.Quantity)
		}
	}

	// 标记已领取
	if err := s.repo.MarkClaimed(ctx, mailID); err != nil {
		return nil, err
	}
	mail.IsClaimed = true

	return mail, nil
}

// DeleteMail 删除邮件
func (s *MailService) DeleteMail(ctx context.Context, mailID, userID string) error {
	mail, err := s.repo.FindByID(ctx, mailID)
	if err != nil {
		return fmt.Errorf("邮件不存在: %w", err)
	}
	if mail.ReceiverID != userID {
		return fmt.Errorf("无权删除该邮件")
	}
	// 如果包含未领取附件，不允许删除
	if !mail.IsClaimed && len(mail.Attachments) > 0 {
		return fmt.Errorf("请先领取附件后再删除")
	}
	return s.repo.DeleteByID(ctx, mailID)
}

// CountUnread 统计未读邮件
func (s *MailService) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

// CleanExpiredMails 清理过期邮件(定时任务调用)
func (s *MailService) CleanExpiredMails(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}

// grantItem 调用 Player 服务添加物品到背包。
func (s *MailService) grantItem(ctx context.Context, playerID, itemID string, quantity int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"item_id":  itemID,
		"quantity": quantity,
	})
	s.postToPlayer(fmt.Sprintf("%s/api/v1/player/%s/inventory/add", s.playerServiceAddr, playerID), body)
}

// grantCurrency 调用 Player 服务增加货币。
func (s *MailService) grantCurrency(ctx context.Context, playerID, currencyType string, amount int64) {
	body, _ := json.Marshal(map[string]interface{}{
		"currency_type": currencyType,
		"amount":        amount,
	})
	s.postToPlayer(fmt.Sprintf("%s/api/v1/player/%s/currency", s.playerServiceAddr, playerID), body)
}

// postToPlayer HTTP POST 工具方法。
func (s *MailService) postToPlayer(url string, body []byte) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}
