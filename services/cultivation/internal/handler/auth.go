// Package handler JWT 认证中间件
//
// 从 Authorization: Bearer <token> 中解析并验证 JWT Token，
// 将 player_id 注入请求上下文供后续 handler 使用。
// 使用 crypto/hmac 验证签名，无需外部 JWT 库。
//
// 注意：
//   - JWT Secret 通过环境变量 JWT_ACCESS_SECRET 配置，默认值与网关一致
//   - 生产环境建议替换为 gRPC 调用 Auth 服务的 ValidateToken
//   - 签名验证使用 HMAC-SHA256，与网关签发时一致
package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type contextKey string

const authPlayerIDKey contextKey = "auth_player_id"

// jwtSecret HMAC 签名密钥，从环境变量读取。
var jwtSecret = func() []byte {
	s := os.Getenv("JWT_ACCESS_SECRET")
	if s == "" {
		s = "default-access-secret-change-in-production"
	}
	return []byte(s)
}()

// AuthMiddleware HTTP 认证中间件。
// 从 Authorization: Bearer <token> 中提取 player_id 并注入请求上下文。
// 签名验证通过 HMAC-SHA256 完成。
// 跳过以下路径：/health, /api/v1/health
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 健康检查及内部端点不需要认证
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" ||
			r.URL.Path == "/api/v1/sync-exp" ||
			r.URL.Path == "/api/v1/player/status/set" {
			next.ServeHTTP(w, r)
			return
		}

		playerID, err := extractPlayerIDFromToken(r)
		if err != nil {
			http.Error(w, `{"code":401,"msg":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// 将 player_id 注入请求上下文
		ctx := context.WithValue(r.Context(), authPlayerIDKey, playerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAuthPlayerID 从请求上下文中获取认证的玩家 ID。
func GetAuthPlayerID(r *http.Request) (uint64, bool) {
	id, ok := r.Context().Value(authPlayerIDKey).(uint64)
	return id, ok
}

// extractPlayerIDFromToken 从 Authorization 头中解析 JWT 并提取 player_id。
// 验证 HMAC-SHA256 签名以确保 token 由网关签发。
func extractPlayerIDFromToken(r *http.Request) (uint64, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, errUnauthorized
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, errUnauthorized
	}

	tokenStr := parts[1]

	// JWT 格式: header.payload.signature
	segments := strings.Split(tokenStr, ".")
	if len(segments) != 3 {
		return 0, errUnauthorized
	}

	// 验证 HMAC-SHA256 签名
	if !verifyJWTSignature(segments[0]+"."+segments[1], segments[2]) {
		return 0, errUnauthorized
	}

	// Base64 解码 payload 部分
	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		payload, err = base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(segments[1])
		if err != nil {
			return 0, errUnauthorized
		}
	}

	var claims struct {
		PlayerID uint64 `json:"player_id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, errUnauthorized
	}

	if claims.PlayerID == 0 {
		return 0, errUnauthorized
	}

	return claims.PlayerID, nil
}

// verifyJWTSignature 验证 JWT HMAC-SHA256 签名。
func verifyJWTSignature(signingInput, signatureSegment string) bool {
	// Base64 URL 解码签名
	sig, err := base64.RawURLEncoding.DecodeString(signatureSegment)
	if err != nil {
		return false
	}

	// 计算期望的 HMAC-SHA256
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)

	return hmac.Equal(sig, expected)
}

var errUnauthorized = &unauthorizedError{}

type unauthorizedError struct{}

func (e *unauthorizedError) Error() string { return "unauthorized" }
