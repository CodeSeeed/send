package middleware

import (
	"send/server/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupCORS(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.Server.AllowedOrigins
	if len(origins) == 0 {
		// Dev mode: echo back any origin (allows credentials with specific origin)
		return cors.New(cors.Config{
			AllowAllOrigins:  false,
			AllowOriginFunc:  func(origin string) bool { return true },
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