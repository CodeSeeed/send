package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func clientIPResponse(t *testing.T, proxies []string, remoteAddr, forwardedFor string) string {
	t.Helper()

	engine := gin.New()
	if err := configureTrustedProxies(engine, proxies); err != nil {
		t.Fatalf("configure trusted proxies: %v", err)
	}
	engine.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = remoteAddr
	request.Header.Set("X-Forwarded-For", forwardedFor)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response.Body.String()
}

func TestConfigureTrustedProxiesTrustsNobodyByDefault(t *testing.T) {
	got := clientIPResponse(t, nil, "203.0.113.10:4321", "198.51.100.20")
	if got != "203.0.113.10" {
		t.Fatalf("ClientIP() = %q, want direct peer IP", got)
	}
}

func TestConfigureTrustedProxiesAcceptsConfiguredProxy(t *testing.T) {
	got := clientIPResponse(t, []string{"127.0.0.1"}, "127.0.0.1:4321", "198.51.100.20")
	if got != "198.51.100.20" {
		t.Fatalf("ClientIP() = %q, want forwarded client IP", got)
	}
}
