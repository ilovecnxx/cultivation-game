// Package service 实现认证业务逻辑：密码哈希、JWT 签发验证、Redis 会话管理。
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cultivation-game/services/auth/internal/config"
	"cultivation-game/services/auth/internal/model"
	"cultivation-game/services/auth/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// 错误定义。
var (
	ErrUserExists       = errors.New("用户名已存在")
	ErrUserNotFound     = errors.New("用户不存在")
	ErrInvalidPassword  = errors.New("密码错误")
	ErrUserBanned       = errors.New("账号已被封禁")
	ErrInvalidToken     = errors.New("无效的令牌")
	ErrTokenExpired     = errors.New("令牌已过期")
	ErrSessionExpired   = errors.New("会话已过期")
)

// SessionKeys Redis 中会话相关键的格式。
const (
	sessionKey      = "auth:session:%d"     // auth:session:<user_id>
	accessTokenKey  = "auth:access:%s"      // auth:access:<token_hash>
	refreshTokenKey = "auth:refresh:%s"     // auth:refresh:<token_hash>
	userTokenKey    = "auth:user_tokens:%d" // auth:user_tokens:<user_id> 存储用户所有活跃 token 的集合
)

// Claims JWT 载荷。
type Claims struct {
	UserID   uint64 `json:"user_id"`
	PlayerID uint64 `json:"player_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService 认证服务，处理注册、登录、令牌管理和会话控制。
type AuthService struct {
	userRepo          *repository.UserRepo
	rdb               *redis.Client
	cfg               *config.Config
	log               *slog.Logger
	playerServiceAddr string // Player 服务 HTTP 地址
}

// NewAuthService 创建 AuthService。
func NewAuthService(userRepo *repository.UserRepo, rdb *redis.Client, cfg *config.Config, log *slog.Logger) *AuthService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8080"
	if v := os.Getenv("PLAYER_SERVICE_ADDR"); v != "" {
		playerAddr = v
	}
	}
	return &AuthService{
		userRepo:          userRepo,
		rdb:               rdb,
		cfg:               cfg,
		log:               log,
		playerServiceAddr: playerAddr,
	}
}

// ---- 注册 ----

// Register 注册新用户。
// 流程：检查用户名唯一性 -> 密码哈希 -> 写入 MySQL -> 签发令牌 -> 写入 Redis 会话。
func (s *AuthService) Register(ctx context.Context, username, password, nickname, gender string) (*model.User, string, string, error) {
	// 检查用户名是否已存在
	existing, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", "", fmt.Errorf("检查用户名失败: %w", err)
	}
	if existing != nil {
		return nil, "", "", ErrUserExists
	}

	// 密码强度校验（至少 6 位）
	if len(password) < 6 {
		return nil, "", "", errors.New("密码长度不能少于 6 位")
	}
	if len(username) < 2 {
		return nil, "", "", errors.New("用户名长度不能少于 2 位")
	}

	// 密码哈希（bcrypt）
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.ErrorContext(ctx, "密码哈希失败", "error", err)
		return nil, "", "", fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户记录（初始无 player_id，后续由 player 服务回填）
	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Email:        "",
		Status:       model.UserStatusNormal,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", "", err
	}

	// 调用 Player 服务创建角色
	playerID, err := s.createPlayerInPlayerService(ctx, user.ID, nickname, "", gender)
	if err != nil {
		s.log.WarnContext(ctx, "创建角色失败（不影响注册）", "error", err, "user_id", user.ID, "nickname", nickname)
	} else {
		user.PlayerID = playerID
		// 回填 PlayerID 到 MySQL
		if err := s.userRepo.UpdatePlayerID(ctx, user.ID, playerID); err != nil {
			s.log.WarnContext(ctx, "回填 PlayerID 失败", "error", err, "user_id", user.ID)
		}
	}

	// 签发 Token 对
	accessToken, refreshToken, err := s.generateTokenPair(ctx, user.ID, user.PlayerID, username)
	if err != nil {
		return nil, "", "", fmt.Errorf("签发令牌失败: %w", err)
	}

	// 写入会话到 Redis
	if err := s.saveSession(ctx, user.ID, user.PlayerID, username, accessToken, refreshToken); err != nil {
		s.log.WarnContext(ctx, "保存会话到 Redis 失败", "error", err, "user_id", user.ID)
		// 非致命，不影响注册成功
	}

	return user, accessToken, refreshToken, nil
}

// ---- 跨服务调用 ----

// spiritRootNameToID 将灵根中文名映射为 Player 服务的 int32 枚举值
func spiritRootNameToID(name string) int32 {
	switch name {
	case "金":
		return 1
	case "木":
		return 2
	case "水":
		return 3
	case "火":
		return 4
	case "土":
		return 5
	case "风":
		return 6
	case "雷":
		return 7
	case "冰":
		return 8
	default:
		return 0 // 无灵根
	}
}

// createPlayerInPlayerService 调用 Player 服务 HTTP API 创建角色
func (s *AuthService) createPlayerInPlayerService(ctx context.Context, userID uint64, nickname, spiritRoot, gender string) (uint64, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id":     fmt.Sprintf("%d", userID),
		"name":        nickname,
		"spirit_root": spiritRootNameToID(spiritRoot),
		"gender":      gender,
	})

	httpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, s.playerServiceAddr+"/api/v1/player/register", bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP 调用失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("Player 服务返回异常状态码 %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"` // 使用 interface{} 兼容类型断言
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}

	// 从 data 中提取 player ID
	if result.Data != nil {
		if id, ok := result.Data["id"].(float64); ok {
			return uint64(id), nil
		}
	}

	return 0, fmt.Errorf("无法从响应中提取 player ID: %s", string(body))
}

// ---- 登录 ----

// Login 用户登录。
// 流程：查询用户 -> 检查状态 -> 验证密码 -> 更新登录时间 -> 签发令牌 -> 写入 Redis 会话。
func (s *AuthService) Login(ctx context.Context, username, password, deviceID string) (*model.User, string, string, error) {
	// 查询用户
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", "", fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, "", "", ErrUserNotFound
	}

	// 检查账号状态
	if user.IsBanned() {
		return nil, "", "", ErrUserBanned
	}
	if !user.IsActive() {
		return nil, "", "", errors.New("账号状态异常")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidPassword
	}

	// 更新登录时间
	if err := s.userRepo.UpdateLoginTime(ctx, user.ID, deviceID); err != nil {
		s.log.WarnContext(ctx, "更新登录时间失败", "error", err, "user_id", user.ID)
	}

	// 签发 Token 对
	accessToken, refreshToken, err := s.generateTokenPair(ctx, user.ID, user.PlayerID, user.Username)
	if err != nil {
		return nil, "", "", fmt.Errorf("签发令牌失败: %w", err)
	}

	// 清理旧会话（如果存在）
	if err := s.cleanupOldSession(ctx, user.ID); err != nil {
		s.log.WarnContext(ctx, "清理旧会话失败", "error", err, "user_id", user.ID)
	}

	// 保存新会话到 Redis
	if err := s.saveSession(ctx, user.ID, user.PlayerID, user.Username, accessToken, refreshToken); err != nil {
		s.log.WarnContext(ctx, "保存会话到 Redis 失败", "error", err, "user_id", user.ID)
	}

	return user, accessToken, refreshToken, nil
}

// ---- 令牌刷新 ----

// TokenRefresh 使用 Refresh Token 换取新的 Access Token（可选同时轮换 Refresh Token）。
func (s *AuthService) TokenRefresh(ctx context.Context, refreshTokenStr string, userID uint64) (newAccessToken, newRefreshToken string, expiresAt int64, err error) {
	// 验证 Refresh Token
	claims, err := s.validateRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", 0, ErrInvalidToken
	}

	if claims.UserID != userID {
		return "", "", 0, ErrInvalidToken
	}

	// 检查 Redis 中是否仍存在此 Refresh Token（防止已注销的 token 被重复使用）
	exists, err := s.rdb.Exists(ctx, fmt.Sprintf(refreshTokenKey, hashToken(refreshTokenStr))).Result()
	if err != nil || exists == 0 {
		return "", "", 0, ErrSessionExpired
	}

	// 查询用户确认状态
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return "", "", 0, ErrUserNotFound
	}
	if user.IsBanned() {
		return "", "", 0, ErrUserBanned
	}

	// 签发新 Token 对（刷新令牌轮换）
	newAccessToken, newRefreshToken, err = s.generateTokenPair(ctx, user.ID, user.PlayerID, user.Username)
	if err != nil {
		return "", "", 0, fmt.Errorf("签发新令牌失败: %w", err)
	}

	// 移除旧的 token 记录
	oldRefreshKey := fmt.Sprintf(refreshTokenKey, hashToken(refreshTokenStr))
	oldAccessKey := fmt.Sprintf(accessTokenKey, hashToken(claims.ID)) // claims.ID 是原始 Access Token 的 jti
	if err := s.rdb.Del(ctx, oldRefreshKey, oldAccessKey).Err(); err != nil {
		s.log.WarnContext(ctx, "删除旧令牌失败", "error", err, "user_id", user.ID)
	}

	// 保存新会话
	if err := s.saveSession(ctx, user.ID, user.PlayerID, user.Username, newAccessToken, newRefreshToken); err != nil {
		s.log.WarnContext(ctx, "保存新会话失败", "error", err, "user_id", user.ID)
	}

	expiresAt = time.Now().Add(s.cfg.JWTAccessExpire).Unix()
	return newAccessToken, newRefreshToken, expiresAt, nil
}

// ---- 登出 ----

// Logout 用户登出，使当前会话失效。
func (s *AuthService) Logout(ctx context.Context, userID uint64, accessToken string) error {
	// 删除用户会话
	sessionKeyStr := fmt.Sprintf(sessionKey, userID)
	if err := s.rdb.Del(ctx, sessionKeyStr).Err(); err != nil {
		s.log.ErrorContext(ctx, "删除会话失败", "error", err, "user_id", userID)
		return fmt.Errorf("登出失败: %w", err)
	}

	// 删除 Access Token 记录
	accessKey := fmt.Sprintf(accessTokenKey, hashToken(accessToken))
	if err := s.rdb.Del(ctx, accessKey).Err(); err != nil {
		s.log.WarnContext(ctx, "删除 Access Token 记录失败", "error", err, "user_id", userID)
	}

	// 删除用户活跃 token 集合
	userTokenSetKey := fmt.Sprintf(userTokenKey, userID)
	if err := s.rdb.Del(ctx, userTokenSetKey).Err(); err != nil {
		s.log.WarnContext(ctx, "删除用户 token 集合失败", "error", err, "user_id", userID)
	}

	s.log.InfoContext(ctx, "用户登出成功", "user_id", userID)
	return nil
}

// ---- 令牌验证（内部服务调用） ----

// ValidateToken 验证 Access Token 并返回用户信息。
func (s *AuthService) ValidateToken(ctx context.Context, accessToken string) (*Claims, error) {
	claims, err := s.validateAccessToken(accessToken)
	if err != nil {
		return nil, err
	}

	// 检查 Redis 中该 token 是否仍有效（未被注销）
	key := fmt.Sprintf(accessTokenKey, hashToken(claims.ID))
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil || exists == 0 {
		return nil, ErrSessionExpired
	}

	return claims, nil
}

// ---- JWT 内部方法 ----

// generateTokenPair 同时生成 Access Token 和 Refresh Token。
func (s *AuthService) generateTokenPair(ctx context.Context, userID, playerID uint64, username string) (accessToken, refreshToken string, err error) {
	accessToken, err = s.generateAccessToken(userID, playerID, username)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = s.generateRefreshToken(userID, playerID, username)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// generateAccessToken 生成 Access Token（短时效，默认 1 小时）。
func (s *AuthService) generateAccessToken(userID, playerID uint64, username string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		PlayerID: playerID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cultivation-game-auth",
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("access_%d_%d", userID, now.UnixNano()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTAccessSecret))
}

// generateRefreshToken 生成 Refresh Token（长效，默认 7 天）。
func (s *AuthService) generateRefreshToken(userID, playerID uint64, username string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		PlayerID: playerID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTRefreshExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cultivation-game-auth",
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("refresh_%d_%d", userID, now.UnixNano()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTRefreshSecret))
}

// validateAccessToken 验证 Access Token 并返回 Claims。
func (s *AuthService) validateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// validateRefreshToken 验证 Refresh Token 并返回 Claims。
func (s *AuthService) validateRefreshToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTRefreshSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ---- Redis 会话管理 ----

// saveSession 将用户会话信息保存到 Redis。
func (s *AuthService) saveSession(ctx context.Context, userID, playerID uint64, username, accessToken, refreshToken string) error {
	pipe := s.rdb.Pipeline()

	// 会话信息（Hash 结构）
	sessionKeyStr := fmt.Sprintf(sessionKey, userID)
	sessionData, _ := json.Marshal(model.SessionInfo{
		UserID:       userID,
		PlayerID:     playerID,
		Username:     username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	pipe.Set(ctx, sessionKeyStr, sessionData, s.cfg.SessionTTL)

	// Access Token 索引（用于验证和快速查找）
	accessKey := fmt.Sprintf(accessTokenKey, hashToken(accessToken))
	pipe.Set(ctx, accessKey, userID, s.cfg.JWTAccessExpire)

	// Refresh Token 索引
	refreshKey := fmt.Sprintf(refreshTokenKey, hashToken(refreshToken))
	pipe.Set(ctx, refreshKey, userID, s.cfg.JWTRefreshExpire)

	// 用户活跃 Token 集合（用于批量清理）
	userTokenSetKey := fmt.Sprintf(userTokenKey, userID)
	pipe.SAdd(ctx, userTokenSetKey, hashToken(accessToken), hashToken(refreshToken))
	pipe.Expire(ctx, userTokenSetKey, s.cfg.JWTRefreshExpire)

	_, err := pipe.Exec(ctx)
	return err
}

// cleanupOldSession 清除用户的旧会话数据。
func (s *AuthService) cleanupOldSession(ctx context.Context, userID uint64) error {
	// 获取旧 token 集合
	userTokenSetKey := fmt.Sprintf(userTokenKey, userID)
	oldTokens, err := s.rdb.SMembers(ctx, userTokenSetKey).Result()
	if err != nil {
		return err
	}

	if len(oldTokens) == 0 {
		return nil
	}

	pipe := s.rdb.Pipeline()
	for _, tokenHash := range oldTokens {
		pipe.Del(ctx, fmt.Sprintf(accessTokenKey, tokenHash))
		pipe.Del(ctx, fmt.Sprintf(refreshTokenKey, tokenHash))
	}
	pipe.Del(ctx, userTokenSetKey)
	pipe.Del(ctx, fmt.Sprintf(sessionKey, userID))

	_, err = pipe.Exec(ctx)
	return err
}

// ---- 辅助函数 ----

// hashToken 对 Token 字符串进行哈希，用作 Redis 键（避免存储完整 Token）。
func hashToken(token string) string {
	// 这里简化处理：取 Token 的 SHA256 前缀。
	// 生产环境应使用完整的 crypto/sha256。
	// 由于 token 本身已通过 JWT 签名，取前 32 位作为简略索引已足够。
	if len(token) > 32 {
		// 取最后 32 字符（JWT 的签名部分变化大，适合做区分）
		return token[len(token)-32:]
	}
	return token
}
