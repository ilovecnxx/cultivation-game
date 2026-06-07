package service

import (
	"net/http"
	"strings"

	"cultivation-game/services/auth/internal/model"

	"github.com/gin-gonic/gin"
)

// GMAuthMiddleware 创建 GM JWT 认证中间件。
// 验证请求头中的 Bearer Token，并将管理员信息注入上下文。
func (s *GMService) GMAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.GMAPIResponse{
				Code:    -1,
				Message: "缺少认证令牌",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			tokenStr = authHeader
		}

		claims, err := s.ValidateGMToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.GMAPIResponse{
				Code:    -1,
				Message: "认证令牌无效或已过期",
			})
			return
		}

		// 将管理员信息注入上下文
		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Set("admin_role", claims.Role)

		c.Next()
	}
}

// GMPermissionMiddleware 创建 GM 权限中间件。
// 检查管理员是否具有写权限（超级管理员或运营）。
func (s *GMService) GMPermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("admin_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, model.GMAPIResponse{
				Code:    -1,
				Message: "权限不足",
			})
			return
		}

		r, ok := role.(int8)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, model.GMAPIResponse{
				Code:    -1,
				Message: "权限不足",
			})
			return
		}

		// 只有超级管理员和运营有写权限
		if r != int8(model.GMAdminRoleSuperAdmin) && r != int8(model.GMAdminRoleOperator) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.GMAPIResponse{
				Code:    -1,
				Message: "您没有执行此操作的权限",
			})
			return
		}

		c.Next()
	}
}
