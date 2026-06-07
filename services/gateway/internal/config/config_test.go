// Package config 配置加载单元测试。
package config

import (
	"testing"
	"time"
)

// clearAllEnv clears all env vars that Load() reads so the defaults test
// is deterministic. Each variable is restored by t.Setenv on test completion.
func clearAllEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_PORT",
		"SHUTDOWN_TIMEOUT",
		"WS_READ_BUFFER_SIZE",
		"WS_WRITE_BUFFER_SIZE",
		"WS_MAX_MESSAGE_SIZE",
		"PING_INTERVAL",
		"PONG_TIMEOUT",
		"RECONNECT_WINDOW",
		"JWT_ACCESS_SECRET",
		"JWT_REFRESH_SECRET",
		"JWT_ACCESS_EXPIRE",
		"JWT_REFRESH_EXPIRE",
		"JWT_ISSUER",
		"RATE_LIMIT_RATE",
		"RATE_LIMIT_CAPACITY",
		"NATS_URL",
		"NATS_CONNECT_TIMEOUT",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"BACKEND_SERVICE_ADDR",
		"GRPC_DIAL_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}

// ---------------------------------------------------------------------------
// Default values
// ---------------------------------------------------------------------------

func TestConfigDefaults(t *testing.T) {
	clearAllEnv(t)

	cfg := Load()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ServerPort", cfg.ServerPort, 8080},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 15 * time.Second},
		{"WSReadBufferSize", cfg.WSReadBufferSize, 4096},
		{"WSWriteBufferSize", cfg.WSWriteBufferSize, 4096},
		{"WSMaxMessageSize", cfg.WSMaxMessageSize, int64(65536)},
		{"PingInterval", cfg.PingInterval, 30 * time.Second},
		{"PongTimeout", cfg.PongTimeout, 60 * time.Second},
		{"ReconnectWindow", cfg.ReconnectWindow, 60 * time.Second},
		{"JWTAccessSecret", cfg.JWTAccessSecret, "default-access-secret-change-in-production"},
		{"JWTRefreshSecret", cfg.JWTRefreshSecret, "default-refresh-secret-change-in-production"},
		{"JWTAccessExpire", cfg.JWTAccessExpire, 15 * time.Minute},
		{"JWTRefreshExpire", cfg.JWTRefreshExpire, 7 * 24 * time.Hour},
		{"JWTIssuer", cfg.JWTIssuer, "cultivation-game-gateway"},
		{"RateLimitRate", cfg.RateLimitRate, 10.0},
		{"RateLimitCapacity", cfg.RateLimitCapacity, 20},
		{"NATSURL", cfg.NATSURL, "nats://localhost:4222"},
		{"NATSConnectTimeout", cfg.NATSConnectTimeout, 5 * time.Second},
		{"RedisAddr", cfg.RedisAddr, "localhost:6379"},
		{"RedisPassword", cfg.RedisPassword, ""},
		{"RedisDB", cfg.RedisDB, 0},
		{"BackendServiceAddr", cfg.BackendServiceAddr, "localhost:50060"},
		{"GRPCDialTimeout", cfg.GRPCDialTimeout, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v (%T), want %v (%T)", tt.name, tt.got, tt.got, tt.want, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getEnv helper functions
// ---------------------------------------------------------------------------

func TestGetEnv(t *testing.T) {
	t.Run("default when not set", func(t *testing.T) {
		if got := getEnv("CONFIG_TEST_NONEXISTENT", "fallback"); got != "fallback" {
			t.Errorf("getEnv = %q, want %q", got, "fallback")
		}
	})

	t.Run("value from env", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_ENV", "custom_value")
		if got := getEnv("CONFIG_TEST_ENV", "fallback"); got != "custom_value" {
			t.Errorf("getEnv = %q, want %q", got, "custom_value")
		}
	})

	t.Run("empty env uses default", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_EMPTY", "")
		if got := getEnv("CONFIG_TEST_EMPTY", "fallback"); got != "fallback" {
			t.Errorf("getEnv = %q, want %q", got, "fallback")
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("default when not set", func(t *testing.T) {
		if got := getEnvInt("CONFIG_INT_NONEXISTENT", 42); got != 42 {
			t.Errorf("getEnvInt = %d, want %d", got, 42)
		}
	})

	t.Run("valid integer", func(t *testing.T) {
		t.Setenv("CONFIG_INT_VALID", "99")
		if got := getEnvInt("CONFIG_INT_VALID", 42); got != 99 {
			t.Errorf("getEnvInt = %d, want %d", got, 99)
		}
	})

	t.Run("invalid integer falls back", func(t *testing.T) {
		t.Setenv("CONFIG_INT_INVALID", "not-a-number")
		if got := getEnvInt("CONFIG_INT_INVALID", 42); got != 42 {
			t.Errorf("getEnvInt = %d, want default %d", got, 42)
		}
	})

	t.Run("empty string falls back", func(t *testing.T) {
		t.Setenv("CONFIG_INT_EMPTY", "")
		if got := getEnvInt("CONFIG_INT_EMPTY", 42); got != 42 {
			t.Errorf("getEnvInt = %d, want default %d", got, 42)
		}
	})

	t.Run("negative integer", func(t *testing.T) {
		t.Setenv("CONFIG_INT_NEG", "-5")
		if got := getEnvInt("CONFIG_INT_NEG", 42); got != -5 {
			t.Errorf("getEnvInt = %d, want %d", got, -5)
		}
	})
}

func TestGetEnvFloat(t *testing.T) {
	t.Run("default when not set", func(t *testing.T) {
		if got := getEnvFloat("CONFIG_FLOAT_NONEXISTENT", 3.14); got != 3.14 {
			t.Errorf("getEnvFloat = %f, want %f", got, 3.14)
		}
	})

	t.Run("valid float", func(t *testing.T) {
		t.Setenv("CONFIG_FLOAT_VALID", "2.718")
		if got := getEnvFloat("CONFIG_FLOAT_VALID", 3.14); got != 2.718 {
			t.Errorf("getEnvFloat = %f, want %f", got, 2.718)
		}
	})

	t.Run("invalid float falls back", func(t *testing.T) {
		t.Setenv("CONFIG_FLOAT_INVALID", "not-a-float")
		if got := getEnvFloat("CONFIG_FLOAT_INVALID", 3.14); got != 3.14 {
			t.Errorf("getEnvFloat = %f, want default %f", got, 3.14)
		}
	})

	t.Run("empty string falls back", func(t *testing.T) {
		t.Setenv("CONFIG_FLOAT_EMPTY", "")
		if got := getEnvFloat("CONFIG_FLOAT_EMPTY", 3.14); got != 3.14 {
			t.Errorf("getEnvFloat = %f, want default %f", got, 3.14)
		}
	})
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("default when not set", func(t *testing.T) {
		if got := getEnvDuration("CONFIG_DUR_NONEXISTENT", 30*time.Second); got != 30*time.Second {
			t.Errorf("getEnvDuration = %v, want %v", got, 30*time.Second)
		}
	})

	t.Run("valid duration", func(t *testing.T) {
		t.Setenv("CONFIG_DUR_VALID", "2m30s")
		if got := getEnvDuration("CONFIG_DUR_VALID", 30*time.Second); got != 2*time.Minute+30*time.Second {
			t.Errorf("getEnvDuration = %v, want %v", got, 2*time.Minute+30*time.Second)
		}
	})

	t.Run("invalid duration falls back", func(t *testing.T) {
		t.Setenv("CONFIG_DUR_INVALID", "not-a-duration")
		if got := getEnvDuration("CONFIG_DUR_INVALID", 30*time.Second); got != 30*time.Second {
			t.Errorf("getEnvDuration = %v, want default %v", got, 30*time.Second)
		}
	})

	t.Run("empty string falls back", func(t *testing.T) {
		t.Setenv("CONFIG_DUR_EMPTY", "")
		if got := getEnvDuration("CONFIG_DUR_EMPTY", 30*time.Second); got != 30*time.Second {
			t.Errorf("getEnvDuration = %v, want default %v", got, 30*time.Second)
		}
	})

	t.Run("seconds format", func(t *testing.T) {
		t.Setenv("CONFIG_DUR_SEC", "5s")
		if got := getEnvDuration("CONFIG_DUR_SEC", 30*time.Second); got != 5*time.Second {
			t.Errorf("getEnvDuration = %v, want %v", got, 5*time.Second)
		}
	})

	t.Run("hours format", func(t *testing.T) {
		t.Setenv("CONFIG_DUR_HOURS", "3h")
		if got := getEnvDuration("CONFIG_DUR_HOURS", 30*time.Second); got != 3*time.Hour {
			t.Errorf("getEnvDuration = %v, want %v", got, 3*time.Hour)
		}
	})
}

// ---------------------------------------------------------------------------
// Environment variable overrides
// ---------------------------------------------------------------------------

func TestConfigEnvOverrides(t *testing.T) {
	// Set specific env vars to known values.
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("WS_READ_BUFFER_SIZE", "8192")
	t.Setenv("WS_WRITE_BUFFER_SIZE", "16384")
	t.Setenv("WS_MAX_MESSAGE_SIZE", "131072")
	t.Setenv("PING_INTERVAL", "10s")
	t.Setenv("PONG_TIMEOUT", "20s")
	t.Setenv("RECONNECT_WINDOW", "120s")
	t.Setenv("JWT_ACCESS_SECRET", "custom-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "custom-refresh-secret")
	t.Setenv("JWT_ACCESS_EXPIRE", "5m")
	t.Setenv("JWT_REFRESH_EXPIRE", "24h")
	t.Setenv("JWT_ISSUER", "custom-issuer")
	t.Setenv("RATE_LIMIT_RATE", "50.5")
	t.Setenv("RATE_LIMIT_CAPACITY", "100")
	t.Setenv("NATS_URL", "nats://custom:4223")
	t.Setenv("NATS_CONNECT_TIMEOUT", "10s")
	t.Setenv("REDIS_ADDR", "redis.custom:6380")
	t.Setenv("REDIS_PASSWORD", "secret123")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("BACKEND_SERVICE_ADDR", "backend.custom:50061")
	t.Setenv("GRPC_DIAL_TIMEOUT", "8s")

	cfg := Load()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ServerPort", cfg.ServerPort, 9090},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 30 * time.Second},
		{"WSReadBufferSize", cfg.WSReadBufferSize, 8192},
		{"WSWriteBufferSize", cfg.WSWriteBufferSize, 16384},
		{"WSMaxMessageSize", cfg.WSMaxMessageSize, int64(131072)},
		{"PingInterval", cfg.PingInterval, 10 * time.Second},
		{"PongTimeout", cfg.PongTimeout, 20 * time.Second},
		{"ReconnectWindow", cfg.ReconnectWindow, 120 * time.Second},
		{"JWTAccessSecret", cfg.JWTAccessSecret, "custom-access-secret"},
		{"JWTRefreshSecret", cfg.JWTRefreshSecret, "custom-refresh-secret"},
		{"JWTAccessExpire", cfg.JWTAccessExpire, 5 * time.Minute},
		{"JWTRefreshExpire", cfg.JWTRefreshExpire, 24 * time.Hour},
		{"JWTIssuer", cfg.JWTIssuer, "custom-issuer"},
		{"RateLimitRate", cfg.RateLimitRate, 50.5},
		{"RateLimitCapacity", cfg.RateLimitCapacity, 100},
		{"NATSURL", cfg.NATSURL, "nats://custom:4223"},
		{"NATSConnectTimeout", cfg.NATSConnectTimeout, 10 * time.Second},
		{"RedisAddr", cfg.RedisAddr, "redis.custom:6380"},
		{"RedisPassword", cfg.RedisPassword, "secret123"},
		{"RedisDB", cfg.RedisDB, 3},
		{"BackendServiceAddr", cfg.BackendServiceAddr, "backend.custom:50061"},
		{"GRPCDialTimeout", cfg.GRPCDialTimeout, 8 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Invalid env var values — should fall back to defaults
// ---------------------------------------------------------------------------

func TestConfigInvalidEnvValues(t *testing.T) {
	clearAllEnv(t)

	// Set invalid values.
	t.Setenv("SERVER_PORT", "not-a-number")
	t.Setenv("RATE_LIMIT_RATE", "not-a-float")
	t.Setenv("PING_INTERVAL", "not-a-duration")

	cfg := Load()

	if cfg.ServerPort != 8080 {
		t.Errorf("ServerPort = %d, want default %d", cfg.ServerPort, 8080)
	}
	if cfg.RateLimitRate != 10.0 {
		t.Errorf("RateLimitRate = %f, want default %f", cfg.RateLimitRate, 10.0)
	}
	if cfg.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %v, want default %v", cfg.PingInterval, 30*time.Second)
	}
}

// ---------------------------------------------------------------------------
// JWT secret defaults
// ---------------------------------------------------------------------------

func TestConfigJWTDefaults(t *testing.T) {
	clearAllEnv(t)
	cfg := Load()

	if cfg.JWTAccessSecret == "" {
		t.Error("JWTAccessSecret should not be empty by default")
	}
	if cfg.JWTRefreshSecret == "" {
		t.Error("JWTRefreshSecret should not be empty by default")
	}
	if cfg.JWTAccessSecret == cfg.JWTRefreshSecret {
		t.Error("access and refresh secrets should be different by default")
	}
	if cfg.JWTIssuer == "" {
		t.Error("JWTIssuer should not be empty by default")
	}
}

// ---------------------------------------------------------------------------
// Port allocation default
// ---------------------------------------------------------------------------

func TestConfigPortDefault(t *testing.T) {
	clearAllEnv(t)
	cfg := Load()

	if cfg.ServerPort != 8080 {
		t.Errorf("ServerPort default = %d, want %d", cfg.ServerPort, 8080)
	}
}

func TestConfigPortOverride(t *testing.T) {
	clearAllEnv(t)

	t.Setenv("SERVER_PORT", "3000")
	cfg := Load()

	if cfg.ServerPort != 3000 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 3000)
	}
}
