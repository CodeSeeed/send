package middleware

import (
	"send/server/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupCORS(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.Server.AllowedOrigins
	if len(origins) == 0 {
		// No allowlist configured — accept same-origin requests only.
		// Previously this echoed back any origin with AllowCredentials: true,
		// which let any website drive credentialed requests against the API
		// whenever an operator forgot to configure allowed_origins.
		// Same-origin requests (frontend + API behind the same host/proxy) are
		// unaffected; cross-origin deployments must set allowed_origins explicitly.
		return cors.New(cors.Config{
			AllowAllOrigins:  false,
			AllowOriginFunc:  func(origin string) bool { return false },
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Admin-Token"},
			AllowCredentials: true,
		})
	}
	return cors.New(cors.Config{
		AllowAllOrigins:  false,
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Admin-Token"},
		AllowCredentials: true,
	})
}
