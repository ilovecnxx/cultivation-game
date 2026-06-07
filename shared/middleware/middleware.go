// Package middleware 提供 Gin 服务共享的 HTTP 中间件。
//
// 包含 CORS、JWT 鉴权、请求日志、限流、错误恢复等中间件。
package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSOptions CORS 配置
type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

// DefaultCORS 返回宽松的 CORS 配置（开发环境）
func DefaultCORS() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			"X-Requested-With", "X-CSRF-Token",
		},
		MaxAge: 86400,
	}
}

// CORS 中间件：处理跨域请求
func CORS(opts CORSOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := "*"

		// 如果配置了具体的允许域名，检查来源是否匹配
		if len(opts.AllowedOrigins) > 0 && opts.AllowedOrigins[0] != "*" {
			allowOrigin = ""
			for _, o := range opts.AllowedOrigins {
				if o == origin || o == "*" {
					allowOrigin = origin
					break
				}
			}
			if allowOrigin == "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", join(opts.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", join(opts.AllowedHeaders, ", "))
		c.Header("Access-Control-Max-Age", itoa(opts.MaxAge))
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		if status >= 500 {
			log.Printf("[HTTP] %s %s %d %v client=%s [ERROR]", method, path, status, duration, clientIP)
		} else if status >= 400 {
			log.Printf("[HTTP] %s %s %d %v client=%s [WARN]", method, path, status, duration, clientIP)
		} else {
			log.Printf("[HTTP] %s %s %d %v client=%s", method, path, status, duration, clientIP)
		}
	}
}

// Recovery 错误恢复中间件（防止 panic 导致服务崩溃）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// JWTAuth JWT 鉴权中间件（简单版本，仅验证 token 存在性）
// 生产环境应与 gateway 的 JWT 验证逻辑集成。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization token",
			})
			return
		}

		// 去掉 "Bearer " 前缀
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// 将 token 存入上下文，具体验证由各服务自行处理
		c.Set("token", token)

		// 尝试从 token 提取 player_id（由 gateway 在签发时设置）
		c.Next()
	}
}

// RateLimit 简易请求限流中间件（单机版，生产环境应使用 Redis 实现）
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	type entry struct {
		count    int
		windowStart time.Time
	}
	store := make(map[string]*entry)

	// 定期清理过期条目
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			for k, v := range store {
				if now.Sub(v.windowStart) > window {
					delete(store, k)
				}
			}
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()

		e, ok := store[key]
		if !ok || time.Since(e.windowStart) > window {
			store[key] = &entry{count: 1, windowStart: time.Now()}
			c.Next()
			return
		}

		e.count++
		if e.count > maxRequests {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

// HealthCheck 返回一个简单的健康检查端点处理函数
func HealthCheck(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"service":   serviceName,
			"timestamp": time.Now().Unix(),
		})
	}
}

// ServiceInfo 返回服务信息端点处理函数
func ServiceInfo(serviceName, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"version": version,
		})
	}
}

// 辅助函数
func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
