// Package auth 提供 JWT 双 Token 鉴权（Access Token + Refresh Token）。
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 载荷。
type Claims struct {
	PlayerID uint64 `json:"player_id"` // 玩家 ID
	Account  string `json:"account"`   // 账号
	jwt.RegisteredClaims
}

// JWTManager 管理 JWT 令牌的签发与验证。
type JWTManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
	issuer        string
}

// NewJWTManager 创建 JWT 管理器。
func NewJWTManager(accessSecret, refreshSecret string, accessExpire, refreshExpire time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
		issuer:        issuer,
	}
}

// GenerateAccessToken 生成 Access Token（短时效）。
func (m *JWTManager) GenerateAccessToken(playerID uint64, account string) (string, error) {
	now := time.Now()
	claims := &Claims{
		PlayerID: playerID,
		Account:  account,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", playerID),
			ID:        fmt.Sprintf("access_%d_%d", playerID, now.UnixNano()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.accessSecret)
}

// GenerateRefreshToken 生成 Refresh Token（长时效）。
func (m *JWTManager) GenerateRefreshToken(playerID uint64, account string) (string, error) {
	now := time.Now()
	claims := &Claims{
		PlayerID: playerID,
		Account:  account,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", playerID),
			ID:        fmt.Sprintf("refresh_%d_%d", playerID, now.UnixNano()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.refreshSecret)
}

// ValidateAccessToken 验证 Access Token 并返回 Claims。
func (m *JWTManager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

// ValidateRefreshToken 验证 Refresh Token 并返回 Claims。
func (m *JWTManager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.refreshSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}
	return claims, nil
}

// RefreshAccessToken 使用 Refresh Token 换取新的 Access Token。
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}
	return m.GenerateAccessToken(claims.PlayerID, claims.Account)
}

// GenerateTokenPair 同时生成 Access Token 和 Refresh Token。
func (m *JWTManager) GenerateTokenPair(playerID uint64, account string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(playerID, account)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = m.GenerateRefreshToken(playerID, account)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
