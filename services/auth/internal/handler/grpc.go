// Package handler 实现 gRPC 传输层，将 gRPC 请求转换为 Service 层调用。
package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"cultivation-game/services/auth/api"
	"cultivation-game/services/auth/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler 实现 api.AuthServiceServer 接口，处理 gRPC 请求。
// 职责：参数校验、错误码映射、日志记录，不包含业务逻辑。
type AuthHandler struct {
	api.UnimplementedAuthServiceServer
	svc *service.AuthService
	log *slog.Logger
}

// NewAuthHandler 创建 AuthHandler。
func NewAuthHandler(svc *service.AuthService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{
		svc: svc,
		log: log,
	}
}

// Login 用户登录 gRPC 处理器。
func (h *AuthHandler) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名不能为空")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "密码不能为空")
	}

	user, accessToken, refreshToken, err := h.svc.Login(ctx, req.Username, req.Password, req.DeviceId)
	if err != nil {
		h.log.WarnContext(ctx, "登录失败", "username", req.Username, "error", err)
		return nil, mapServiceError(err)
	}

	now := time.Now()
	return &api.LoginResponse{
		UserId:            user.ID,
		PlayerId:          user.PlayerID,
		AccessToken:       accessToken,
		RefreshToken:      refreshToken,
		ExpiresAt:         now.Add(1 * time.Hour).Unix(),
		RefreshExpiresAt:  now.Add(7 * 24 * time.Hour).Unix(),
		IsNewPlayer:       user.PlayerID == 0,
	}, nil
}

// Register 用户注册 gRPC 处理器。
func (h *AuthHandler) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名不能为空")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "密码不能为空")
	}
	if req.Nickname == "" {
		return nil, status.Error(codes.InvalidArgument, "角色昵称不能为空")
	}

	user, accessToken, refreshToken, err := h.svc.Register(ctx, req.Username, req.Password, req.Nickname, req.SpiritRoot)
	if err != nil {
		h.log.WarnContext(ctx, "注册失败", "username", req.Username, "error", err)
		return nil, mapServiceError(err)
	}

	return &api.RegisterResponse{
		UserId:       user.ID,
		PlayerId:     user.PlayerID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

// TokenRefresh 令牌刷新 gRPC 处理器。
func (h *AuthHandler) TokenRefresh(ctx context.Context, req *api.TokenRefreshRequest) (*api.TokenRefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "刷新令牌不能为空")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 不能为空")
	}

	newAccessToken, newRefreshToken, expiresAt, err := h.svc.TokenRefresh(ctx, req.RefreshToken, req.UserId)
	if err != nil {
		h.log.WarnContext(ctx, "令牌刷新失败", "user_id", req.UserId, "error", err)
		return nil, mapServiceError(err)
	}

	return &api.TokenRefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout 用户登出 gRPC 处理器。
func (h *AuthHandler) Logout(ctx context.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 不能为空")
	}

	if err := h.svc.Logout(ctx, req.UserId, req.AccessToken); err != nil {
		h.log.ErrorContext(ctx, "登出失败", "user_id", req.UserId, "error", err)
		return nil, status.Error(codes.Internal, "登出失败")
	}

	return &api.LogoutResponse{
		Success: true,
	}, nil
}

// ValidateToken 令牌验证 gRPC 处理器（内部服务调用）。
func (h *AuthHandler) ValidateToken(ctx context.Context, req *api.ValidateTokenRequest) (*api.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "令牌不能为空")
	}

	claims, err := h.svc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &api.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &api.ValidateTokenResponse{
		UserId:   claims.UserID,
		PlayerId: claims.PlayerID,
		Username: claims.Username,
		Valid:    true,
	}, nil
}

// mapServiceError 将 Service 层的业务错误映射为 gRPC 状态码。
func mapServiceError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, service.ErrUserExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, service.ErrUserBanned):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, service.ErrTokenExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, service.ErrSessionExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, "服务器内部错误")
	}
}
