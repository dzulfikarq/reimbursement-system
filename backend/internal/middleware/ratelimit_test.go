package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimitAllowsThenBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(rate.Limit(2), 2)) // 2 req burst
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		r.ServeHTTP(w, req)
		codes = append(codes, w.Code)
	}

	if codes[0] != http.StatusOK || codes[1] != http.StatusOK {
		t.Fatalf("first two requests should pass, got %v", codes[:2])
	}
	for _, code := range codes[2:] {
		if code != http.StatusTooManyRequests {
			t.Fatalf("requests beyond burst should be 429, got %v", codes)
		}
	}

	// 429 body must match the error envelope.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	r.ServeHTTP(w, req)
	var body struct {
		Error struct{ Code string }
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Error.Code != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED envelope, got %s (err=%v)", w.Body.String(), err)
	}
}

func TestRateLimitIsPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(rate.Limit(1), 1))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "9.9.9.9:2000"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("different IP should have its own budget, got %d", w.Code)
	}
}
