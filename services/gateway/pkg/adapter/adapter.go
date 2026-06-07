// Package gatewayadapter wraps gateway internal packages for monolith consumption.
package gatewayadapter

import (
	"time"

	"cultivation-game/services/gateway/internal/auth"
)

// JWTManager is a type alias for the internal auth.JWTManager.
type JWTManager = auth.JWTManager

// NewJWTManager creates a JWT manager using the gateway's auth package.
func NewJWTManager(accessSecret, refreshSecret string, accessExpire, refreshExpire time.Duration, issuer string) *JWTManager {
	return auth.NewJWTManager(accessSecret, refreshSecret, accessExpire, refreshExpire, issuer)
}
