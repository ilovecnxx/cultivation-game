// Package authadapter wraps auth service internals for monolith consumption.
package authadapter

import (
	"context"
	"database/sql"
	"log/slog"

	"cultivation-game/services/auth/internal/config"
	"cultivation-game/services/auth/internal/handler"
	"cultivation-game/services/auth/internal/repository"
	"cultivation-game/services/auth/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Components holds all initialized auth service components.
type Components struct {
	AuthService *service.AuthService
	GMHandler   *handler.GMHandler
	GMService   *service.GMService
	Config      *config.Config
}

// LoginResult holds login result with exported fields only.
type LoginResult struct {
	UserID       uint64
	PlayerID     uint64
	AccessToken  string
	RefreshToken string
	Err          error
}

// RegisterResult holds register result with exported fields only.
type RegisterResult struct {
	UserID       uint64
	PlayerID     uint64
	AccessToken  string
	RefreshToken string
	Err          error
}

// Bootstrap initializes the auth service layer and returns its components.
func Bootstrap(db *sql.DB, rdb *redis.Client, log *slog.Logger) *Components {
	cfg := config.Load()
	userRepo := repository.NewUserRepo(db, log)
	authSvc := service.NewAuthService(userRepo, rdb, cfg, log)

	gmRepo := repository.NewGMRepo(db, log)
	gmSvc := service.NewGMService(gmRepo, userRepo, cfg, log)
	gmHandler := handler.NewGMHandler(gmSvc, log)

	return &Components{
		AuthService: authSvc,
		GMHandler:   gmHandler,
		GMService:   gmSvc,
		Config:      cfg,
	}
}

// Login delegates to AuthService.Login.
func (c *Components) Login(ctx context.Context, username, password string) *LoginResult {
	user, accessToken, refreshToken, err := c.AuthService.Login(ctx, username, password, "")
	if err != nil {
		return &LoginResult{Err: err}
	}
	return &LoginResult{
		UserID:       user.ID,
		PlayerID:     user.PlayerID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// Register delegates to AuthService.Register.
func (c *Components) Register(ctx context.Context, username, password, nickname, gender string) *RegisterResult {
	user, accessToken, refreshToken, err := c.AuthService.Register(ctx, username, password, nickname, gender)
	if err != nil {
		return &RegisterResult{Err: err}
	}
	return &RegisterResult{
		UserID:       user.ID,
		PlayerID:     user.PlayerID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// GMAuthMiddleware returns the GM auth middleware.
func (c *Components) GMAuthMiddleware() gin.HandlerFunc {
	return c.GMService.GMAuthMiddleware()
}

// GMPermissionMiddleware returns the GM permission middleware.
func (c *Components) GMPermissionMiddleware() gin.HandlerFunc {
	return c.GMService.GMPermissionMiddleware()
}

// SeedDefaultAdmin seeds the default GM admin account.
func (c *Components) SeedDefaultAdmin() error {
	c.GMService.SetJWTSecret(c.Config.GMJWTSecret)
	return c.GMService.SeedDefaultAdmin(context.Background())
}
