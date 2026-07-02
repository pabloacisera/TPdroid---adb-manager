package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	return r
}

func TestMiddleware_AllowedOrigin(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "http://localhost:8080" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:8080', got %q", v)
	}
}

func TestMiddleware_DisallowedOrigin(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("expected empty Access-Control-Allow-Origin for disallowed origin, got %q", v)
	}
}

func TestMiddleware_NoOriginHeader(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("expected empty Access-Control-Allow-Origin when no Origin header, got %q", v)
	}
}

func TestMiddleware_NullOriginRejected(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "null")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("expected empty Access-Control-Allow-Origin for null origin, got %q", v)
	}
}

func TestMiddleware_OptionsPreflight(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "http://localhost:8080" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:8080', got %q", v)
	}
}

func TestMiddleware_SessionToken_MissingOnPOST(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionTokenMiddleware())
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for missing session token, got %d", w.Code)
	}
}

func TestMiddleware_SessionToken_ValidOnPOST(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionTokenMiddleware())
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-Session-Token", GetSessionToken())
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for valid session token, got %d", w.Code)
	}
}

func TestMiddleware_SessionToken_InvalidOnPOST(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionTokenMiddleware())
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-Session-Token", "invalidtoken123456789012345678901234")
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for invalid session token, got %d", w.Code)
	}
}

func TestMiddleware_SessionToken_GETNotChecked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionTokenMiddleware())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for GET without token (bypassed), got %d", w.Code)
	}
}

func TestMiddleware_SessionToken_GetSessionTokenNotEmpty(t *testing.T) {
	token := GetSessionToken()
	if len(token) != 64 {
		t.Errorf("expected token length 64 hex chars, got %d: %q", len(token), token)
	}
}

func TestMiddleware_FileProtocolRejected(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "file://")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("expected empty Access-Control-Allow-Origin for file:// origin, got %q", v)
	}
}
