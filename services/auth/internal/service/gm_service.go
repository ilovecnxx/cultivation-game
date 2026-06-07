// Package service 实现 GM 管理后台业务逻辑。
package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cultivation-game/services/auth/internal/config"
	"cultivation-game/services/auth/internal/model"
	"cultivation-game/services/auth/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// GM 错误定义。
var (
	ErrGMInvalidCredentials = errors.New("管理员用户名或密码错误")
	ErrGMNotFound           = errors.New("管理员不存在")
	ErrGMDisabled           = errors.New("管理员账号已被禁用")
	ErrGMPermissionDenied   = errors.New("权限不足")
	ErrGMPLayerNotFound     = errors.New("玩家不存在")
	ErrGMInvalidParams      = errors.New("参数错误")
)

// GMService GM 管理后台服务。
type GMService struct {
	gmRepo            *repository.GMRepo
	userRepo          *repository.UserRepo
	cfg               *config.Config
	log               *slog.Logger
	jwtSecret         string
	tokenExpire       time.Duration
	playerServiceAddr string
}

// NewGMService 创建 GMService。
func NewGMService(gmRepo *repository.GMRepo, userRepo *repository.UserRepo, cfg *config.Config, log *slog.Logger) *GMService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8080"
	if v := os.Getenv("PLAYER_SERVICE_ADDR"); v != "" {
		playerAddr = v
	}
	}
	return &GMService{
		gmRepo:            gmRepo,
		userRepo:          userRepo,
		cfg:               cfg,
		log:               log,
		playerServiceAddr: playerAddr,
		jwtSecret:         os.Getenv("GM_JWT_SECRET"),
		tokenExpire:       24 * time.Hour,
	}
}

// SetJWTSecret 设置 JWT 密钥（从配置文件加载）。
func (s *GMService) SetJWTSecret(secret string) {
	if secret != "" {
		s.jwtSecret = secret
	}
}

// ---- GM 认证 ----

// AuthenticateGM 验证 GM 管理员凭据，返回 JWT 令牌。
func (s *GMService) AuthenticateGM(ctx context.Context, username, password string) (*model.GMLoginResponse, error) {
	admin, err := s.gmRepo.GetAdminByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("查询管理员失败: %w", err)
	}
	if admin == nil {
		s.log.WarnContext(ctx, "GM 登录失败：管理员不存在", "username", username)
		return nil, ErrGMInvalidCredentials
	}

	if !admin.IsEnabled() {
		s.log.WarnContext(ctx, "GM 登录失败：账号已禁用", "username", username, "admin_id", admin.ID)
		return nil, ErrGMDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		s.log.WarnContext(ctx, "GM 登录失败：密码错误", "username", username)
		return nil, ErrGMInvalidCredentials
	}

	// 更新最后登录时间
	if err := s.gmRepo.UpdateAdminLoginTime(ctx, admin.ID); err != nil {
		s.log.WarnContext(ctx, "更新管理员登录时间失败", "error", err, "admin_id", admin.ID)
	}

	// 生成 JWT
	token, expiresAt, err := s.generateGMToken(admin)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	// 记录操作日志
	s.logOperation(ctx, admin.ID, model.GMActionLogin, model.GMTargetSystem, 0, map[string]interface{}{
		"username": username,
	})

	return &model.GMLoginResponse{
		Token:     token,
		AdminID:   admin.ID,
		Username:  admin.Username,
		Role:      admin.Role,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateGMToken 验证 GM JWT 令牌。
func (s *GMService) ValidateGMToken(tokenStr string) (*model.GMClaims, error) {
	// 解析 JWT
	parts, err := s.parseUnsignedJWT(tokenStr)
	if err != nil {
		return nil, errors.New("无效的令牌格式")
	}

	// 验证签名
	sig := s.signJWT(parts.Header + "." + parts.Payload)
	if !hmac.Equal([]byte(parts.Signature), []byte(sig)) {
		return nil, errors.New("令牌签名无效")
	}

	// 解析 claims
	claims := &model.GMClaims{}
	if err := json.Unmarshal([]byte(parts.Payload), claims); err != nil {
		return nil, errors.New("令牌载荷解析失败")
	}

	// 检查过期
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("令牌已过期")
	}

	return claims, nil
}

// jwtParts JWT 分段。
type jwtParts struct {
	Header    string
	Payload   string
	Signature string
}

// parseUnsignedJWT 解析 JWT 的三段。
func (s *GMService) parseUnsignedJWT(tokenStr string) (*jwtParts, error) {
	// 使用简单的 base64 解码
	parts := &jwtParts{}
	// 查找第一个和第二个点
	dot1 := -1
	dot2 := -1
	for i, c := range tokenStr {
		if c == '.' {
			if dot1 == -1 {
				dot1 = i
			} else if dot2 == -1 {
				dot2 = i
				break
			}
		}
	}
	if dot1 == -1 || dot2 == -1 {
		return nil, errors.New("无效的 JWT 格式")
	}

	parts.Header = tokenStr[:dot1]
	parts.Payload = tokenStr[dot1+1 : dot2]
	parts.Signature = tokenStr[dot2+1:]

	return parts, nil
}

// signJWT 对 JWT 载荷进行 HMAC-SHA256 签名。
func (s *GMService) signJWT(data string) string {
	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(data))
	return s.base64URLEncode(mac.Sum(nil)[:32])
}

// base64URLEncode 进行 Base64 URL 安全的编码（无补全）。
func (s *GMService) base64URLEncode(data []byte) string {
	encoded := hex.EncodeToString(data)
	// 使用标准的 base64url 编码
	return encoded
}

// generateGMToken 生成 GM 管理员的 JWT 令牌。
func (s *GMService) generateGMToken(admin *model.GMAdmin) (string, int64, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	headerB64 := s.base64URLEncode([]byte(header))

	expiresAt := time.Now().Add(s.tokenExpire).Unix()

	claims := model.GMClaims{
		AdminID:  admin.ID,
		Username: admin.Username,
		Role:     int8(admin.Role),
		Subject:  fmt.Sprintf("%d", admin.ID),
		Issuer:   "cultivation-game-gm",
		ExpiresAt: expiresAt,
		IssuedAt:  time.Now().Unix(),
	}

	payloadBytes, _ := json.Marshal(claims)
	payloadB64 := s.base64URLEncode(payloadBytes)

	signature := s.signJWT(headerB64 + "." + payloadB64)

	token := headerB64 + "." + payloadB64 + "." + signature
	return token, expiresAt, nil
}

// ---- 玩家管理 ----

// GetPlayerList 搜索玩家列表。
func (s *GMService) GetPlayerList(ctx context.Context, search string, page, limit int) (*model.GMPlayerSearchResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	url := fmt.Sprintf("%s/api/v1/player/list?search=%s&page=%d&limit=%d",
		s.playerServiceAddr, search, page, limit)

	resp, err := s.httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int                       `json:"code"`
		Data *model.GMPlayerSearchResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 || apiResp.Data == nil {
		return &model.GMPlayerSearchResponse{
			Total: 0,
			Page:  page,
			Items: []*model.GMPLayer{},
		}, nil
	}

	if apiResp.Data.Items == nil {
		apiResp.Data.Items = []*model.GMPLayer{}
	}
	return apiResp.Data, nil
}

// GetPlayerDetail 获取玩家详细信息。
func (s *GMService) GetPlayerDetail(ctx context.Context, playerID uint64) (*model.GMPLayer, error) {
	url := fmt.Sprintf("%s/api/v1/player/%d", s.playerServiceAddr, playerID)

	resp, err := s.httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int           `json:"code"`
		Data *model.GMPLayer `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 || apiResp.Data == nil {
		return nil, ErrGMPLayerNotFound
	}

	return apiResp.Data, nil
}

// EditPlayerAttribute 修改玩家属性。
func (s *GMService) EditPlayerAttribute(ctx context.Context, playerID uint64, adminID uint64, field string, value interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"field": field,
		"value": value,
	})

	url := fmt.Sprintf("%s/api/v1/player/%d/attribute", s.playerServiceAddr, playerID)
	resp, err := s.httpPost(ctx, url, body)
	if err != nil {
		return fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 {
		return fmt.Errorf("修改属性失败: %s", apiResp.Msg)
	}

	// 记录操作日志
	s.logOperation(ctx, adminID, model.GMActionEditAttribute, model.GMTargetPlayer, playerID, map[string]interface{}{
		"field": field,
		"value": value,
	})

	return nil
}

// ---- 封禁 ----

// BanPlayer 封禁玩家。
func (s *GMService) BanPlayer(ctx context.Context, playerID, adminID uint64, reason string, banType int8, durationMinutes int) error {
	admin, err := s.gmRepo.GetAdminByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if admin == nil {
		return ErrGMNotFound
	}
	if !admin.CanWrite() {
		return ErrGMPermissionDenied
	}

	var endsAt *time.Time
	if banType != int8(model.GMBanPermanent) && durationMinutes > 0 {
		t := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
		endsAt = &t
	}

	ban := &model.GMBan{
		PlayerID: playerID,
		AdminID:  adminID,
		Reason:   reason,
		BanType:  model.GMBanType(banType),
		EndsAt:   endsAt,
		Status:   model.GMBanActive,
	}

	if err := s.gmRepo.InsertBan(ctx, ban); err != nil {
		return fmt.Errorf("创建封禁记录失败: %w", err)
	}

	// 通知 Player 服务封禁
	banBody, _ := json.Marshal(map[string]interface{}{
		"ban_type": banType,
		"reason":   reason,
		"ends_at":  endsAt,
	})
	url := fmt.Sprintf("%s/api/v1/player/%d/ban", s.playerServiceAddr, playerID)
	resp, err := s.httpPost(ctx, url, banBody)
	if err != nil {
		s.log.WarnContext(ctx, "通知 Player 服务封禁失败", "error", err, "player_id", playerID)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// 同时更新用户状态
	if err := s.userRepo.UpdateStatus(ctx, playerID, model.UserStatusBanned); err != nil {
		s.log.WarnContext(ctx, "更新用户状态失败", "error", err, "player_id", playerID)
	}

	// 记录日志
	s.logOperation(ctx, adminID, model.GMActionBanPlayer, model.GMTargetPlayer, playerID, map[string]interface{}{
		"reason":         reason,
		"ban_type":       banType,
		"duration_minutes": durationMinutes,
	})

	return nil
}

// UnbanPlayer 解封玩家。
func (s *GMService) UnbanPlayer(ctx context.Context, playerID, adminID uint64) error {
	admin, err := s.gmRepo.GetAdminByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if admin == nil {
		return ErrGMNotFound
	}
	if !admin.CanWrite() {
		return ErrGMPermissionDenied
	}

	if err := s.gmRepo.DeactivateBan(ctx, playerID); err != nil {
		return fmt.Errorf("解除封禁失败: %w", err)
	}

	// 通知 Player 服务解封
	url := fmt.Sprintf("%s/api/v1/player/%d/unban", s.playerServiceAddr, playerID)
	resp, err := s.httpPost(ctx, url, nil)
	if err != nil {
		s.log.WarnContext(ctx, "通知 Player 服务解封失败", "error", err, "player_id", playerID)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// 恢复用户状态
	if err := s.userRepo.UpdateStatus(ctx, playerID, model.UserStatusNormal); err != nil {
		s.log.WarnContext(ctx, "更新用户状态失败", "error", err, "player_id", playerID)
	}

	// 记录日志
	s.logOperation(ctx, adminID, model.GMActionUnbanPlayer, model.GMTargetPlayer, playerID, map[string]interface{}{
		"action": "unban",
	})

	return nil
}

// ---- 公告 ----

// SendAnnouncement 发送公告。
func (s *GMService) SendAnnouncement(ctx context.Context, adminID uint64, title, content string, annType int8, targetPlayerID uint64) error {
	admin, err := s.gmRepo.GetAdminByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if admin == nil {
		return ErrGMNotFound
	}
	if !admin.CanWrite() {
		return ErrGMPermissionDenied
	}

	if title == "" || content == "" {
		return ErrGMInvalidParams
	}

	var targetPtr *uint64
	if targetPlayerID > 0 {
		targetPtr = &targetPlayerID
	}

	announcement := &model.GMAnnouncement{
		AdminID:        adminID,
		Title:          title,
		Content:        content,
		Type:           model.GMAnnouncementType(annType),
		TargetPlayerID: targetPtr,
	}

	if err := s.gmRepo.InsertAnnouncement(ctx, announcement); err != nil {
		return fmt.Errorf("创建公告失败: %w", err)
	}

	// 如果类型是世界公告，通知所有在线玩家
	if annType == int8(model.GMAnnouncementWorld) {
		s.broadcastAnnouncement(ctx, title, content)
	}

	// 记录日志
	s.logOperation(ctx, adminID, model.GMActionAnnouncement, model.GMTargetAnnouncement, announcement.ID, map[string]interface{}{
		"title":   title,
		"type":    annType,
		"content": content,
	})

	return nil
}

// broadcastAnnouncement 广播世界公告。
func (s *GMService) broadcastAnnouncement(ctx context.Context, title, content string) {
	body, _ := json.Marshal(map[string]string{
		"title":   title,
		"content": content,
		"type":    "world",
	})
	// 发送到世界服务或网关进行广播
	url := os.Getenv("GATEWAY_BROADCAST_URL")
	if url == "" {
		url = "http://127.0.0.1:8080/api/v1/gateway/broadcast"
	}
	resp, err := s.httpPost(ctx, url, body)
	if err != nil {
		s.log.WarnContext(ctx, "广播公告失败", "error", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
}

// ---- 物品发放 ----

// SendItem 给玩家发送物品。
func (s *GMService) SendItem(ctx context.Context, playerID, adminID uint64, itemID string, quantity int) error {
	admin, err := s.gmRepo.GetAdminByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if admin == nil {
		return ErrGMNotFound
	}
	if !admin.CanWrite() {
		return ErrGMPermissionDenied
	}

	if itemID == "" || quantity <= 0 {
		return ErrGMInvalidParams
	}

	body, _ := json.Marshal(map[string]interface{}{
		"item_id":  itemID,
		"quantity": quantity,
	})

	url := fmt.Sprintf("%s/api/v1/player/%d/inventory/add", s.playerServiceAddr, playerID)
	resp, err := s.httpPost(ctx, url, body)
	if err != nil {
		return fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 {
		return fmt.Errorf("发放物品失败: %s", apiResp.Msg)
	}

	// 记录日志
	s.logOperation(ctx, adminID, model.GMActionSendItem, model.GMTargetPlayer, playerID, map[string]interface{}{
		"item_id":  itemID,
		"quantity": quantity,
	})

	return nil
}

// ---- 服务器统计 ----

// GetServerStats 获取服务器统计信息。
func (s *GMService) GetServerStats(ctx context.Context) (*model.GMServerStats, error) {
	stats := &model.GMServerStats{
		Version: "1.0.0",
		Uptime:  "0天0小时0分钟",
	}

	// 获取总玩家数
	totalPlayers, err := s.fetchTotalPlayers(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "获取总玩家数失败", "error", err)
	} else {
		stats.TotalPlayers = totalPlayers
	}

	// 获取 DAU
	todayDAU, peakDAU, err := s.fetchDAU(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "获取 DAU 失败", "error", err)
	} else {
		stats.TodayDAU = todayDAU
		stats.PeakDAU = peakDAU
	}

	// 获取在线玩家数
	online, err := s.fetchOnlineCount(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "获取在线人数失败", "error", err)
	} else {
		stats.OnlinePlayers = online
	}

	return stats, nil
}

// fetchTotalPlayers 从玩家服务获取总玩家数。
func (s *GMService) fetchTotalPlayers(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/player/stats/total", s.playerServiceAddr)
	resp, err := s.httpGet(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}
	return apiResp.Data.Total, nil
}

// fetchDAU 获取 DAU 数据。
func (s *GMService) fetchDAU(ctx context.Context) (today, peak int64, err error) {
	url := fmt.Sprintf("%s/api/v1/player/stats/dau", s.playerServiceAddr)
	resp, err := s.httpGet(ctx, url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			Today int64 `json:"today"`
			Peak  int64 `json:"peak"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, 0, err
	}
	return apiResp.Data.Today, apiResp.Data.Peak, nil
}

// fetchOnlineCount 获取在线玩家数。
func (s *GMService) fetchOnlineCount(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/api/v1/player/stats/online", s.playerServiceAddr)
	resp, err := s.httpGet(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			Online int `json:"online"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}
	return apiResp.Data.Online, nil
}

// ---- 操作日志 ----

// GetOperationLogs 获取操作日志。
func (s *GMService) GetOperationLogs(ctx context.Context, page, limit int) (*model.GMOperationLogResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, total, err := s.gmRepo.GetOperationLogs(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return &model.GMOperationLogResponse{
		Total: total,
		Page:  page,
		Items: items,
	}, nil
}

// ---- 内部辅助 ----

// logOperation 记录 GM 操作日志。
func (s *GMService) logOperation(ctx context.Context, adminID uint64, actionType model.GMActionType, targetType model.GMAuditTargetType, targetID uint64, detail map[string]interface{}) {
	detailBytes, _ := json.Marshal(detail)

	opLog := &model.GMOperationLog{
		AdminID:    adminID,
		ActionType: actionType,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detailBytes,
		IPAddress:  getClientIP(ctx),
	}

	if err := s.gmRepo.InsertOperationLog(ctx, opLog); err != nil {
		s.log.WarnContext(ctx, "记录操作日志失败", "error", err)
	}
}

// getClientIP 从上下文中获取客户端 IP。
func getClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return ""
}

// httpGet 执行 HTTP GET 请求。
func (s *GMService) httpGet(ctx context.Context, url string) (*http.Response, error) {
	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

// httpPost 执行 HTTP POST 请求。
func (s *GMService) httpPost(ctx context.Context, url string, body []byte) (*http.Response, error) {
	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

// ---- 种子数据 ----

// SeedDefaultAdmin 创建默认管理员账号（如果不存在）。
func (s *GMService) SeedDefaultAdmin(ctx context.Context) error {
	existing, err := s.gmRepo.GetAdminByUsername(ctx, "admin")
	if err != nil {
		return fmt.Errorf("查询默认管理员失败: %w", err)
	}
	if existing != nil {
		return nil // 已存在
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	admin := &model.GMAdmin{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         model.GMAdminRoleSuperAdmin,
		Status:       model.GMAdminStatusEnabled,
	}

	if err := s.gmRepo.CreateAdmin(ctx, admin); err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}

	s.log.InfoContext(ctx, "已创建默认 GM 管理员账号", "username", "admin", "password", "admin123")
	return nil
}
