// Package server 管理网关与后端服务之间的 gRPC 连接池。
package server

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"cultivation-game/services/auth/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCClient 多连接 gRPC 客户端连接池。
// 预热 N 个连接，轮询分发请求，断线自动重连。
type GRPCClient struct {
	addr      string
	poolSize  int
	timeout   time.Duration
	conns     []*grpc.ClientConn
	next      atomic.Uint64 // 轮询计数器
	mu        sync.RWMutex
	closed    bool
}

// NewGRPCClient 创建 gRPC 连接池。
// poolSize: 连接池大小（建议 4，单机场景下够用）。
func NewGRPCClient(addr string, timeout time.Duration) (*GRPCClient, error) {
	poolSize := 4
	c := &GRPCClient{
		addr:     addr,
		poolSize: poolSize,
		timeout:  timeout,
		conns:    make([]*grpc.ClientConn, 0, poolSize),
	}

	// 预热连接池
	var lastErr error
	for i := 0; i < poolSize; i++ {
		conn, err := c.dial()
		if err != nil {
			slog.Warn("gRPC 连接池预热失败", "index", i, "error", err)
			lastErr = err
			continue
		}
		c.conns = append(c.conns, conn)
		slog.Info("gRPC 连接池预热成功", "index", i, "addr", addr)
	}

	if len(c.conns) == 0 {
		return nil, lastErr
	}

	// 后台健康检查 + 自动补充连接
	go c.healthCheckLoop()

	slog.Info("gRPC 连接池初始化完成", "addr", addr, "pool_size", len(c.conns))
	return c, nil
}

// dial 建立单个 gRPC 连接。
func (c *GRPCClient) dial() (*grpc.ClientConn, error) {
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
		return nil, err
	}
	return conn, nil
}

// GetConn 轮询获取下一个可用连接。
func (c *GRPCClient) GetConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || len(c.conns) == 0 {
		return nil
	}

	// 轮询策略：从健康连接池中轮询分发
	n := len(c.conns)
	if n == 1 {
		return c.conns[0]
	}

	idx := c.next.Add(1) % uint64(n)
	conn := c.conns[idx]

	// 如果选中的连接不健康，尝试下一个（最多尝试一轮）
	if conn.GetState() == connectivity.Shutdown {
		for i := uint64(0); i < uint64(n-1); i++ {
			idx = (idx + 1) % uint64(n)
			conn = c.conns[idx]
			if conn.GetState() != connectivity.Shutdown {
				return conn
			}
		}
		// 所有连接都挂了，返回第一个（调用方会处理错误）
		return c.conns[0]
	}

	return conn
}

// Invoke 调用远程方法（简化包装）。
func (c *GRPCClient) Invoke(ctx context.Context, method string, args interface{}, reply interface{}) error {
	conn := c.GetConn()
	if conn == nil {
		return grpc.ErrClientConnClosing
	}
	return conn.Invoke(ctx, method, args, reply)
}

// healthCheckLoop 定期检查连接状态，断开时自动补充连接至 poolSize。
func (c *GRPCClient) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.RLock()
		if c.closed {
			c.mu.RUnlock()
			return
		}
		poolSize := len(c.conns)
		c.mu.RUnlock()

		// 统计健康连接数
		healthy := 0
		c.mu.RLock()
		for _, conn := range c.conns {
			if conn.GetState() != connectivity.Shutdown {
				healthy++
			}
		}
		c.mu.RUnlock()

		// 补充连接至 poolSize
		needed := c.poolSize - healthy
		if needed > 0 {
			slog.Info("gRPC 连接池需要补充",
				"healthy", healthy, "target", c.poolSize, "needed", needed)

			c.mu.Lock()
			for i := 0; i < needed && len(c.conns) < c.poolSize*2; i++ {
				conn, err := c.dial()
				if err != nil {
					slog.Warn("gRPC 补充连接失败", "error", err)
					break
				}
				c.conns = append(c.conns, conn)
			}
			c.mu.Unlock()
		}

		// 如果连接数超过 poolSize*2，清理关闭的连接
		c.mu.Lock()
		alive := make([]*grpc.ClientConn, 0, len(c.conns))
		for _, conn := range c.conns {
			if conn.GetState() != connectivity.Shutdown {
				alive = append(alive, conn)
			} else {
				conn.Close()
			}
		}
		// 收缩到 poolSize 左右（留一点余量）
		if len(alive) > c.poolSize*2 {
			for _, conn := range alive[c.poolSize:] {
				conn.Close()
			}
			alive = alive[:c.poolSize]
		}
		c.conns = alive
		c.mu.Unlock()

		if poolSize != len(c.conns) {
			slog.Debug("gRPC 连接池状态", "before", poolSize, "after", len(c.conns))
		}
	}
}

// Close 关闭连接池中所有连接。
func (c *GRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	var lastErr error
	for _, conn := range c.conns {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	c.conns = nil
	return lastErr
}

// =========================================================
// Auth 服务 gRPC 调用封装
// =========================================================

// authServiceClient 创建 AuthService gRPC 客户端（使用连接池连接）。
func (c *GRPCClient) authServiceClient() api.AuthServiceClient {
	conn := c.GetConn()
	if conn == nil {
		return nil
	}
	return api.NewAuthServiceClient(conn)
}

// AuthLogin 调用 Auth 服务的 Login RPC。
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

// AuthRegister 调用 Auth 服务的 Register RPC。
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

// AuthValidateToken 调用 Auth 服务的 ValidateToken RPC。
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
