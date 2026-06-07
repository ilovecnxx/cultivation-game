// Package server 管理网关与后端服务之间的 gRPC 连接。
package server

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"cultivation-game/services/auth/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCClient gRPC 客户端连接管理器。
// 支持连接池复用、健康检查和自动重连。
type GRPCClient struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	mu      sync.RWMutex
	closed  bool
}

// NewGRPCClient 创建 gRPC 客户端，并建立连接。
func NewGRPCClient(addr string, timeout time.Duration) (*GRPCClient, error) {
	c := &GRPCClient{
		addr:    addr,
		timeout: timeout,
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	go c.healthCheckLoop()
	return c, nil
}

// connect 建立 gRPC 连接。
func (c *GRPCClient) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64*1024),
			grpc.MaxCallSendMsgSize(64*1024),
		),
	)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		conn.Close()
		return nil
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = conn
	slog.Info("gRPC connected", "addr", c.addr)
	return nil
}

// GetConn 获取当前 gRPC 连接。
func (c *GRPCClient) GetConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// Invoke 调用远程方法（简化包装）。
func (c *GRPCClient) Invoke(ctx context.Context, method string, args interface{}, reply interface{}) error {
	conn := c.GetConn()
	if conn == nil {
		return grpc.ErrClientConnClosing
	}
	return conn.Invoke(ctx, method, args, reply)
}

// healthCheckLoop 定期检查连接状态，断开时自动重连。
func (c *GRPCClient) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.RLock()
		if c.closed {
			c.mu.RUnlock()
			return
		}
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil || conn.GetState() == connectivity.Shutdown {
			slog.Warn("gRPC connection lost, reconnecting", "addr", c.addr)
			if err := c.connect(); err != nil {
				slog.Error("gRPC reconnect failed", "error", err, "addr", c.addr)
			}
		}
	}
}

// Close 关闭连接。
func (c *GRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// =========================================================
// Auth 服务 gRPC 调用封装
// =========================================================

// authServiceClient 创建 AuthService gRPC 客户端（每次调用新建，底层复用连接）。
func (c *GRPCClient) authServiceClient() api.AuthServiceClient {
	conn := c.GetConn()
	if conn == nil {
		return nil
	}
	return api.NewAuthServiceClient(conn)
}

// AuthLogin 调用 Auth 服务的 Login RPC 验证用户名密码。
// 返回 Auth 服务的登录响应，包含 user_id、player_id 等身份信息。
func (c *GRPCClient) AuthLogin(ctx context.Context, username, password string) (*api.LoginResponse, error) {
	client := c.authServiceClient()
	if client == nil {
		return nil, grpc.ErrClientConnClosing
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	slog.Debug("calling auth service Login", "username", username)
	return client.Login(ctx, &api.LoginRequest{
		Username: username,
		Password: password,
	})
}

// AuthRegister 调用 Auth 服务的 Register RPC 注册新用户。
// 注册成功后返回用户身份信息及已签发的 Token 对。
func (c *GRPCClient) AuthRegister(ctx context.Context, username, password, nickname string) (*api.RegisterResponse, error) {
	client := c.authServiceClient()
	if client == nil {
		return nil, grpc.ErrClientConnClosing
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	slog.Debug("calling auth service Register", "username", username)
	return client.Register(ctx, &api.RegisterRequest{
		Username: username,
		Password: password,
		Nickname: nickname,
	})
}

// AuthValidateToken 调用 Auth 服务的 ValidateToken RPC 验证 Access Token 有效性。
func (c *GRPCClient) AuthValidateToken(ctx context.Context, token string) (*api.ValidateTokenResponse, error) {
	client := c.authServiceClient()
	if client == nil {
		return nil, grpc.ErrClientConnClosing
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return client.ValidateToken(ctx, &api.ValidateTokenRequest{
		AccessToken: token,
	})
}
