package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- CORS tests ----------

func TestCORS_SetsCorrectHeaders(t *testing.T) {
	router := gin.New()
	router.Use(CORS(DefaultCORS()))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	tests := []struct {
		header string
		want   string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Credentials", "true"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS"},
		{"Access-Control-Max-Age", "86400"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := w.Header().Get(tt.header); got != tt.want {
				t.Errorf("header %q = %q, want %q", tt.header, got, tt.want)
			}
		})
	}

	// Access-Control-Allow-Headers should not be empty
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Access-Control-Allow-Headers should not be empty")
	}
}

func TestCORS_HandlesPreflight(t *testing.T) {
	router := gin.New()
	router.Use(CORS(DefaultCORS()))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status for OPTIONS = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Headers should still be set for preflight
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Access-Control-Allow-Origin header should be set on OPTIONS")
	}
}

func TestCORS_WithSpecificOrigins(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"http://trusted.com", "http://valid.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Authorization"},
		MaxAge:         3600,
	}

	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}

	t.Run("allowed origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(opts))
		router.GET("/test", handler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://trusted.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://trusted.com" {
			t.Errorf("Allow-Origin = %q, want %q", got, "http://trusted.com")
		}
	})

	t.Run("blocked origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(opts))
		router.GET("/test", handler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
		}
	})

	t.Run("wildcard in allowed origins allows all", func(t *testing.T) {
		optsWithWild := CORSOptions{
			AllowedOrigins: []string{"http://trusted.com", "*"},
			AllowedMethods: []string{"GET"},
			AllowedHeaders: []string{"*"},
			MaxAge:         3600,
		}
		router := gin.New()
		router.Use(CORS(optsWithWild))
		router.GET("/test", handler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://any-origin.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://any-origin.com" {
			t.Errorf("Allow-Origin = %q, want echo of origin %q", got, "http://any-origin.com")
		}
	})
}

func TestCORS_EmptyAllowedOriginsDefaultsStar(t *testing.T) {
	// When AllowedOrigins is empty, it should behave as if "*" is allowed
	opts := CORSOptions{
		AllowedOrigins: []string{},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{},
		MaxAge:         0,
	}

	router := gin.New()
	router.Use(CORS(opts))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// With empty AllowedOrigins, len > 0 check fails, defaults to "*"
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want %q", got, "*")
	}
}

// ---------- RequestLogger tests ----------

func TestRequestLogger_DoesNotCrash(t *testing.T) {
	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequestLogger_WithErrorStatus(t *testing.T) {
	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/notfound", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/notfound", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRequestLogger_WithServerError(t *testing.T) {
	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ---------- Recovery tests ----------

func TestRecovery_CatchesPanics(t *testing.T) {
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic from handler")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "internal server error" {
		t.Errorf("error body = %v, want %q", body["error"], "internal server error")
	}
}

func TestRecovery_PassesThroughNormalRequests(t *testing.T) {
	router := gin.New()
	router.Use(Recovery())
	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "all good"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ok", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------- JWTAuth tests ----------

func TestJWTAuth_BlocksRequestsWithoutToken(t *testing.T) {
	router := gin.New()
	router.Use(JWTAuth())
	router.GET("/secure", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"secret": "data"})
	})

	t.Run("no authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/secure", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}

		var body map[string]interface{}
		json.NewDecoder(w.Body).Decode(&body)
		if body["error"] != "missing authorization token" {
			t.Errorf("error = %v, want %q", body["error"], "missing authorization token")
		}
	})

	t.Run("empty authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/secure", nil)
		req.Header.Set("Authorization", "")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestJWTAuth_PassesWithValidToken(t *testing.T) {
	router := gin.New()
	router.Use(JWTAuth())
	router.GET("/secure", func(c *gin.Context) {
		token, exists := c.Get("token")
		if !exists {
			t.Error("token should be set in context")
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	t.Run("bearer token in header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/secure", nil)
		req.Header.Set("Authorization", "Bearer my-jwt-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var body map[string]interface{}
		json.NewDecoder(w.Body).Decode(&body)
		// The token should have "Bearer " prefix stripped
		if body["token"] != "my-jwt-token" {
			t.Errorf("token = %v, want %q", body["token"], "my-jwt-token")
		}
	})

	t.Run("token in query parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/secure?token=query-token-value", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var body map[string]interface{}
		json.NewDecoder(w.Body).Decode(&body)
		if body["token"] != "query-token-value" {
			t.Errorf("token = %v, want %q", body["token"], "query-token-value")
		}
	})

	t.Run("token without bearer prefix", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/secure", nil)
		req.Header.Set("Authorization", "raw-token-without-bearer")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var body map[string]interface{}
		json.NewDecoder(w.Body).Decode(&body)
		if body["token"] != "raw-token-without-bearer" {
			t.Errorf("token = %v, want %q", body["token"], "raw-token-without-bearer")
		}
	})
}

// ---------- RateLimit tests ----------

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	maxReqs := 3
	router := gin.New()
	router.Use(RateLimit(maxReqs, 5*time.Minute))
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < maxReqs; i++ {
		t.Run("request", func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
			}
		})
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	maxReqs := 2
	router := gin.New()
	router.Use(RateLimit(maxReqs, 5*time.Minute))
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First 2 requests should pass
	for i := 0; i < maxReqs; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api", nil)
		req.RemoteAddr = "10.0.0.2:54321"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// 3rd request should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.0.0.2:54321"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "rate limit exceeded" {
		t.Errorf("error = %v, want %q", body["error"], "rate limit exceeded")
	}
}

func TestRateLimit_DifferentIPsAreIndependent(t *testing.T) {
	router := gin.New()
	router.Use(RateLimit(1, 5*time.Minute))
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Each IP should be able to make 1 request
	for _, ip := range []string{"10.0.0.1:12345", "10.0.0.2:54321", "10.0.0.3:9999"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api", nil)
		req.RemoteAddr = ip
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("IP %s: status = %d, want %d", ip, w.Code, http.StatusOK)
		}
	}
}

// ---------- HealthCheck tests ----------

func TestHealthCheck_ReturnsCorrectResponse(t *testing.T) {
	router := gin.New()
	router.GET("/health", HealthCheck("my-service"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tests := []struct {
		field string
		want  interface{}
	}{
		{"status", "ok"},
		{"service", "my-service"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if body[tt.field] != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, body[tt.field], tt.want)
			}
		})
	}

	// timestamp should be present and numeric
	ts, ok := body["timestamp"].(float64)
	if !ok {
		t.Fatalf("timestamp = %v (%T), want float64", body["timestamp"], body["timestamp"])
	}
	if ts <= 0 {
		t.Errorf("timestamp should be positive, got %f", ts)
	}
}

func TestHealthCheck_DifferentServiceName(t *testing.T) {
	router := gin.New()
	router.GET("/health", HealthCheck("cultivation-service"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["service"] != "cultivation-service" {
		t.Errorf("service = %v, want %q", body["service"], "cultivation-service")
	}
}

// ---------- ServiceInfo tests ----------

func TestServiceInfo_ReturnsVersion(t *testing.T) {
	router := gin.New()
	router.GET("/info", ServiceInfo("player", "1.2.3"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/info", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["service"] != "player" {
		t.Errorf("service = %v, want %q", body["service"], "player")
	}
	if body["version"] != "1.2.3" {
		t.Errorf("version = %v, want %q", body["version"], "1.2.3")
	}
}

func TestServiceInfo_EmptyVersion(t *testing.T) {
	router := gin.New()
	router.GET("/info", ServiceInfo("svc", ""))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/info", nil)
	router.ServeHTTP(w, req)

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	if body["version"] != "" {
		t.Errorf("version = %v, want empty string", body["version"])
	}
}

// ---------- Helper function tests ----------

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		sep   string
		want  string
	}{
		{"multiple items", []string{"a", "b", "c"}, ",", "a,b,c"},
		{"single item", []string{"only"}, ",", "only"},
		{"empty slice", []string{}, ",", ""},
		{"custom separator", []string{"x", "y", "z"}, " | ", "x | y | z"},
		{"nil slice", nil, ",", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := join(tt.parts, tt.sep); got != tt.want {
				t.Errorf("join(%v, %q) = %q, want %q", tt.parts, tt.sep, got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"small number", 42, "42"},
		{"hundred", 100, "100"},
		{"large number", 999999, "999999"},
		{"ten", 10, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := itoa(tt.n); got != tt.want {
				t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestJoinAndItoaWorkTogether(t *testing.T) {
	// This tests the actual CORS usage pattern: join uses itoa for MaxAge
	opts := DefaultCORS()
	maxAgeStr := itoa(opts.MaxAge)

	// Verify it matches the expected format
	if maxAgeStr != "86400" {
		t.Errorf("itoa(86400) = %q, want %q", maxAgeStr, "86400")
	}

	headers := join(opts.AllowedHeaders, ", ")
	if headers == "" {
		t.Error("joined headers should not be empty")
	}

	methods := join(opts.AllowedMethods, ", ")
	if methods == "" {
		t.Error("joined methods should not be empty")
	}
}
