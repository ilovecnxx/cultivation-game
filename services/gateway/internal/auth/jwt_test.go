// Package auth JWT 鉴权单元测试。
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAccessSecret  = "test-access-secret-key-for-hmac-signing-32chars"
	testRefreshSecret = "test-refresh-secret-key-for-hmac-signing-32c"
	testIssuer        = "test-cultivation-game-gateway"
)

func newManager(accessExp, refreshExp time.Duration) *JWTManager {
	return NewJWTManager(testAccessSecret, testRefreshSecret, accessExp, refreshExp, testIssuer)
}

func newDefaultManager() *JWTManager {
	return newManager(15*time.Minute, 7*24*time.Hour)
}

// ---------------------------------------------------------------------------
// GenerateAccessToken
// ---------------------------------------------------------------------------

func TestGenerateAccessToken(t *testing.T) {
	mgr := newDefaultManager()

	tests := []struct {
		name     string
		playerID uint64
		account  string
	}{
		{"normal player", 10001, "test_player"},
		{"zero player ID", 0, "anonymous"},
		{"large player ID", 18446744073709551615, "max_uint64"},
		{"empty account", 20002, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := mgr.GenerateAccessToken(tt.playerID, tt.account)
			if err != nil {
				t.Fatalf("GenerateAccessToken failed: %v", err)
			}
			if token == "" {
				t.Fatal("expected non-empty token")
			}

			claims, err := mgr.ValidateAccessToken(token)
			if err != nil {
				t.Fatalf("ValidateAccessToken failed: %v", err)
			}
			if claims.PlayerID != tt.playerID {
				t.Errorf("claims.PlayerID = %d, want %d", claims.PlayerID, tt.playerID)
			}
			if claims.Account != tt.account {
				t.Errorf("claims.Account = %q, want %q", claims.Account, tt.account)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateRefreshToken
// ---------------------------------------------------------------------------

func TestGenerateRefreshToken(t *testing.T) {
	mgr := newDefaultManager()

	tests := []struct {
		name     string
		playerID uint64
		account  string
	}{
		{"normal player", 10001, "test_player"},
		{"empty account", 20002, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := mgr.GenerateRefreshToken(tt.playerID, tt.account)
			if err != nil {
				t.Fatalf("GenerateRefreshToken failed: %v", err)
			}
			if token == "" {
				t.Fatal("expected non-empty token")
			}

			claims, err := mgr.ValidateRefreshToken(token)
			if err != nil {
				t.Fatalf("ValidateRefreshToken failed: %v", err)
			}
			if claims.PlayerID != tt.playerID {
				t.Errorf("claims.PlayerID = %d, want %d", claims.PlayerID, tt.playerID)
			}
			if claims.Account != tt.account {
				t.Errorf("claims.Account = %q, want %q", claims.Account, tt.account)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — valid
// ---------------------------------------------------------------------------

func TestValidateAccessToken_Valid(t *testing.T) {
	mgr := newDefaultManager()
	token, err := mgr.GenerateAccessToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims == nil {
		t.Fatal("expected non-nil claims")
	}
	if claims.PlayerID != 10001 {
		t.Errorf("claims.PlayerID = %d, want %d", claims.PlayerID, 10001)
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — expired
// ---------------------------------------------------------------------------

func TestValidateAccessToken_Expired(t *testing.T) {
	mgr := newDefaultManager()

	// Manually craft an already-expired token with the same secret.
	claims := &Claims{
		PlayerID: 10001,
		Account:  "player1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    testIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testAccessSecret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = mgr.ValidateAccessToken(tokenStr)
	if err == nil {
		t.Error("expected error for expired access token, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — malformed / empty / invalid inputs
// ---------------------------------------------------------------------------

func TestValidateAccessToken_Malformed(t *testing.T) {
	mgr := newDefaultManager()

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"garbage", "this-is-not-a-jwt-token"},
		{"two parts", "header.payload"},
		{"four parts", "a.b.c.d"},
		{"invalid base64 in header", "!!!.payload.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.ValidateAccessToken(tt.token)
			if err == nil {
				t.Error("expected error for malformed token, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — wrong secret
// ---------------------------------------------------------------------------

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	mgrA := NewJWTManager(
		"correct-access-secret-here-1234567890",
		testRefreshSecret,
		15*time.Minute, 7*24*time.Hour, testIssuer,
	)
	mgrB := NewJWTManager(
		"different-access-secret-here-abcdefgh",
		testRefreshSecret,
		15*time.Minute, 7*24*time.Hour, testIssuer,
	)

	token, err := mgrA.GenerateAccessToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = mgrB.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for token signed with different secret, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — different (non-HMAC) algorithm
// ---------------------------------------------------------------------------

func TestValidateAccessToken_DifferentAlgorithm(t *testing.T) {
	mgr := newDefaultManager()

	// Create an ES256-signed token (ECDSA, not HMAC).
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}

	claims := &Claims{
		PlayerID: 10001,
		Account:  "player1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    testIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign ES256 token: %v", err)
	}

	_, err = mgr.ValidateAccessToken(tokenStr)
	if err == nil {
		t.Error("expected error for ES256-signed token (non-HMAC), got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateAccessToken — refresh token cannot be used as access token
// ---------------------------------------------------------------------------

func TestValidateAccessToken_RefreshTokenFails(t *testing.T) {
	mgr := newDefaultManager()

	refreshToken, err := mgr.GenerateRefreshToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	// A refresh token should not validate as an access token
	// (different signing key).
	_, err = mgr.ValidateAccessToken(refreshToken)
	if err == nil {
		t.Error("expected error when validating refresh token as access token")
	}
}

// ---------------------------------------------------------------------------
// ValidateRefreshToken — valid
// ---------------------------------------------------------------------------

func TestValidateRefreshToken_Valid(t *testing.T) {
	mgr := newDefaultManager()
	token, err := mgr.GenerateRefreshToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	claims, err := mgr.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if claims == nil {
		t.Fatal("expected non-nil claims")
	}
	if claims.Account != "player1" {
		t.Errorf("claims.Account = %s, want %s", claims.Account, "player1")
	}
}

// ---------------------------------------------------------------------------
// ValidateRefreshToken — expired
// ---------------------------------------------------------------------------

func TestValidateRefreshToken_Expired(t *testing.T) {
	mgr := newDefaultManager()

	claims := &Claims{
		PlayerID: 10001,
		Account:  "player1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    testIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testRefreshSecret))
	if err != nil {
		t.Fatalf("failed to sign expired refresh token: %v", err)
	}

	_, err = mgr.ValidateRefreshToken(tokenStr)
	if err == nil {
		t.Error("expected error for expired refresh token, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateRefreshToken — access token cannot be used as refresh token
// ---------------------------------------------------------------------------

func TestValidateRefreshToken_AccessTokenFails(t *testing.T) {
	mgr := newDefaultManager()

	accessToken, err := mgr.GenerateAccessToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = mgr.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Error("expected error when validating access token as refresh token")
	}
}

// ---------------------------------------------------------------------------
// ValidateRefreshToken — malformed
// ---------------------------------------------------------------------------

func TestValidateRefreshToken_Malformed(t *testing.T) {
	mgr := newDefaultManager()
	_, err := mgr.ValidateRefreshToken("")
	if err == nil {
		t.Error("expected error for empty refresh token")
	}
}

// ---------------------------------------------------------------------------
// GenerateTokenPair
// ---------------------------------------------------------------------------

func TestGenerateTokenPair(t *testing.T) {
	mgr := newDefaultManager()
	const playerID uint64 = 10001
	const account = "test_player"

	accessToken, refreshToken, err := mgr.GenerateTokenPair(playerID, account)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}
	if accessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if refreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	accessClaims, err := mgr.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	refreshClaims, err := mgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if accessClaims.PlayerID != playerID {
		t.Errorf("access token playerID = %d, want %d", accessClaims.PlayerID, playerID)
	}
	if refreshClaims.PlayerID != playerID {
		t.Errorf("refresh token playerID = %d, want %d", refreshClaims.PlayerID, playerID)
	}
	if accessClaims.Account != refreshClaims.Account {
		t.Errorf("account mismatch: access=%q refresh=%q", accessClaims.Account, refreshClaims.Account)
	}
}

// ---------------------------------------------------------------------------
// RefreshAccessToken
// ---------------------------------------------------------------------------

func TestRefreshAccessToken(t *testing.T) {
	mgr := newDefaultManager()
	const playerID uint64 = 10001
	const account = "player1"

	refreshToken, err := mgr.GenerateRefreshToken(playerID, account)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	newAccessToken, err := mgr.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if newAccessToken == "" {
		t.Fatal("expected non-empty new access token")
	}

	claims, err := mgr.ValidateAccessToken(newAccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.PlayerID != playerID {
		t.Errorf("claims.PlayerID = %d, want %d", claims.PlayerID, playerID)
	}
	if claims.Account != account {
		t.Errorf("claims.Account = %s, want %s", claims.Account, account)
	}
}

func TestRefreshAccessToken_ExpiredRefresh(t *testing.T) {
	mgr := newManager(15*time.Minute, -1*time.Hour) // refresh token already expired

	// GenerateRefreshToken with a negative expiry: the token is expired immediately.
	refreshToken, err := mgr.GenerateRefreshToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	_, err = mgr.RefreshAccessToken(refreshToken)
	if err == nil {
		t.Error("expected error for expired refresh token, got nil")
	}
}

// ---------------------------------------------------------------------------
// Claims verification
// ---------------------------------------------------------------------------

func TestTokenClaims(t *testing.T) {
	mgr := newDefaultManager()
	const playerID uint64 = 10001
	const account = "player_with_claims"

	accessToken, err := mgr.GenerateAccessToken(playerID, account)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.PlayerID != playerID {
		t.Errorf("PlayerID = %d, want %d", claims.PlayerID, playerID)
	}
	if claims.Account != account {
		t.Errorf("Account = %s, want %s", claims.Account, account)
	}
	if claims.Issuer != testIssuer {
		t.Errorf("Issuer = %s, want %s", claims.Issuer, testIssuer)
	}
	if claims.Subject != "10001" {
		t.Errorf("Subject = %s, want %s", claims.Subject, "10001")
	}

	// The token ID should contain "access_{playerID}_".
	if len(claims.ID) == 0 {
		t.Error("expected non-empty token ID")
	}
}

// ---------------------------------------------------------------------------
// Expiry time verification
// ---------------------------------------------------------------------------

func TestAccessTokenExpiry(t *testing.T) {
	accessExp := 15 * time.Minute
	mgr := newManager(accessExp, 7*24*time.Hour)

	token, err := mgr.GenerateAccessToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	parsedClaims := &Claims{}
	parser := jwt.NewParser()
	_, _, err = parser.ParseUnverified(token, parsedClaims)
	if err != nil {
		t.Fatalf("ParseUnverified failed: %v", err)
	}

	got := parsedClaims.ExpiresAt.Time.Sub(parsedClaims.IssuedAt.Time)
	// Allow small tolerance for test execution time.
	if got < accessExp-5*time.Second || got > accessExp+5*time.Second {
		t.Errorf("access token lifetime = %v, want ~%v", got, accessExp)
	}
}

func TestRefreshTokenExpiry(t *testing.T) {
	refreshExp := 7 * 24 * time.Hour
	mgr := newManager(15*time.Minute, refreshExp)

	token, err := mgr.GenerateRefreshToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	parsedClaims := &Claims{}
	parser := jwt.NewParser()
	_, _, err = parser.ParseUnverified(token, parsedClaims)
	if err != nil {
		t.Fatalf("ParseUnverified failed: %v", err)
	}

	got := parsedClaims.ExpiresAt.Time.Sub(parsedClaims.IssuedAt.Time)
	if got < refreshExp-5*time.Second || got > refreshExp+5*time.Second {
		t.Errorf("refresh token lifetime = %v, want ~%v", got, refreshExp)
	}
}

// ---------------------------------------------------------------------------
// Edge: empty / missing issuer
// ---------------------------------------------------------------------------

func TestJWTManager_EmptyIssuer(t *testing.T) {
	mgr := NewJWTManager(testAccessSecret, testRefreshSecret, 15*time.Minute, 7*24*time.Hour, "")

	token, err := mgr.GenerateAccessToken(10001, "player1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.Issuer != "" {
		t.Errorf("Issuer = %q, want empty", claims.Issuer)
	}
}
