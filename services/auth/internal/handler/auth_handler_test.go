package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// mockAuthService 模拟认证服务
type mockAuthService struct {
	registerFunc func(username, password string) (uint64, string, string, error)
	loginFunc    func(username, password string) (uint64, string, string, error)
	refreshFunc  func(refreshToken string) (string, string, error)
}

func setupTestRouter(svc *mockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// 创建一个简单的 handler 用于测试
	r := gin.Default()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
			return
		}
		if len(req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "密码长度不能少于6位"})
			return
		}
		playerID, access, refresh, err := svc.registerFunc(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{
				"player_id":      playerID,
				"access_token":   access,
				"refresh_token":  refresh,
			},
		})
		_ = logger
	})

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
			return
		}
		playerID, access, refresh, err := svc.loginFunc(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{
				"player_id":     playerID,
				"access_token":  access,
				"refresh_token": refresh,
			},
		})
	})

	return r
}

func TestRegister_Success(t *testing.T) {
	svc := &mockAuthService{
		registerFunc: func(username, password string) (uint64, string, string, error) {
			return 1, "access-token-123", "refresh-token-456", nil
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "testuser",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", resp["code"])
	}
	data := resp["data"].(map[string]interface{})
	if data["player_id"].(float64) != 1 {
		t.Errorf("expected player_id 1, got %v", data["player_id"])
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	svc := &mockAuthService{
		registerFunc: func(username, password string) (uint64, string, string, error) {
			return 0, "", "", &mockError{"用户名已存在"}
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "existinguser",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc := &mockAuthService{
		registerFunc: func(username, password string) (uint64, string, string, error) {
			return 0, "", "", nil
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "testuser",
		"password": "123", // too short
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for weak password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	svc := &mockAuthService{
		loginFunc: func(username, password string) (uint64, string, string, error) {
			if username == "testuser" && password == "correctpass" {
				return 1, "access-abc", "refresh-def", nil
			}
			return 0, "", "", &mockError{"用户名或密码错误"}
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "testuser",
		"password": "correctpass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := &mockAuthService{
		loginFunc: func(username, password string) (uint64, string, string, error) {
			return 0, "", "", &mockError{"用户名或密码错误"}
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "testuser",
		"password": "wrongpass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	svc := &mockAuthService{
		loginFunc: func(username, password string) (uint64, string, string, error) {
			return 0, "", "", &mockError{"用户不存在"}
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "nobody",
		"password": "somepass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for nonexistent user, got %d", w.Code)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	svc := &mockAuthService{
		registerFunc: func(username, password string) (uint64, string, string, error) {
			return 0, "", "", nil
		},
	}
	r := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"username": "testuser",
		// missing password
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing fields, got %d", w.Code)
	}
}

// mockError 模拟错误类型
type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
