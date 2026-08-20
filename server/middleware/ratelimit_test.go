package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func performLocalRequest(engine *gin.Engine) int {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:4321"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response.Code
}

func newRateLimitTestEngine(t *testing.T, rateLimitLocalhost bool) *gin.Engine {
	t.Helper()

	limiter := NewRateLimiter(
		RateLimitTier{Rate: 1, Window: time.Minute},
		rateLimitLocalhost,
	)
	t.Cleanup(func() { close(limiter.stopCh) })

	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("disable trusted proxies: %v", err)
	}
	engine.GET("/", limiter.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return engine
}

func TestRateLimiterSkipsLocalhostWhenDisabled(t *testing.T) {
	engine := newRateLimitTestEngine(t, false)
	if got := performLocalRequest(engine); got != http.StatusOK {
		t.Fatalf("first response = %d, want 200", got)
	}
	if got := performLocalRequest(engine); got != http.StatusOK {
		t.Fatalf("second response = %d, want localhost bypass", got)
	}
}

func TestRateLimiterLimitsLocalhostWhenEnabled(t *testing.T) {
	engine := newRateLimitTestEngine(t, true)
	if got := performLocalRequest(engine); got != http.StatusOK {
		t.Fatalf("first response = %d, want 200", got)
	}
	if got := performLocalRequest(engine); got != http.StatusTooManyRequests {
		t.Fatalf("second response = %d, want 429", got)
	}
}
