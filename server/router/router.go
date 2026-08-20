package router

import (
	"log"
	"send/server/config"
	"send/server/controller"
	"send/server/middleware"

	"github.com/gin-gonic/gin"
)

var (
	uploadLimiter   *middleware.RateLimiter
	adminLimiter    *middleware.RateLimiter
	loginLimiter    *middleware.RateLimiter
	downloadLimiter *middleware.RateLimiter
	verifyLimiter   *middleware.RateLimiter
)

func Setup(
	cfg *config.Config,
	fileCtr *controller.FileController,
	adminCtr *controller.AdminController,
	adminMW gin.HandlerFunc,
) *gin.Engine {
	// Lazy init rate limiters with config.
	rateLimitLocalhost := cfg.Server.RateLimitLocalhost
	uploadLimiter = middleware.NewRateLimiter(middleware.TierUpload, rateLimitLocalhost)
	adminLimiter = middleware.NewRateLimiter(middleware.TierAdmin, rateLimitLocalhost)
	loginLimiter = middleware.NewRateLimiter(middleware.TierLogin, rateLimitLocalhost)
	downloadLimiter = middleware.NewRateLimiter(middleware.TierDownload, rateLimitLocalhost)
	verifyLimiter = middleware.NewRateLimiter(middleware.TierDownload, rateLimitLocalhost)

	r := gin.Default()
	// Gin trusts every proxy by default. Always override that default: an empty
	// list means trust nobody, while configured entries are the only proxies
	// allowed to supply the client IP used by rate limits and download tokens.
	if err := configureTrustedProxies(r, cfg.Server.TrustedProxies); err != nil {
		panic(err)
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		log.Println("[warn] server.allowed_origins is empty: cross-origin browser requests are refused. " +
			"Set allowed_origins in config.yaml when the frontend is served from another origin.")
	}
	r.Use(middleware.SetupCORS(cfg))
	r.Use(middleware.SecurityHeaders())

	api := r.Group("/api")
	{
		// Login has its own strict limiter
		api.POST("/admin/login", loginLimiter.Middleware(), adminCtr.Login)
		api.GET("/admin/status", adminCtr.AdminStatus)
		api.POST("/admin/register", loginLimiter.Middleware(), adminCtr.Register)

		// File upload (admin-only)
		files := api.Group("/files")
		{
			files.POST("", adminMW, uploadLimiter.Middleware(), fileCtr.Upload)
			files.POST("/receive", verifyLimiter.Middleware(), fileCtr.Receive)
			files.GET("/:code", fileCtr.Info)
			files.POST("/:code/verify", verifyLimiter.Middleware(), fileCtr.VerifyPassword)
			files.GET("/:code/preview", downloadLimiter.Middleware(), fileCtr.Preview)
			files.GET("/:code/download", downloadLimiter.Middleware(), fileCtr.Download)
		}

		// Admin routes with generous rate limit
		admin := api.Group("/admin")
		admin.Use(adminMW, adminLimiter.Middleware())
		{
			admin.GET("/check", adminCtr.Check)
			admin.POST("/logout", adminCtr.Logout)
			admin.POST("/password", adminCtr.ChangePassword)
			admin.GET("/files", adminCtr.ListFiles)
			admin.DELETE("/files/:id", adminCtr.DeleteFile)
			admin.GET("/settings", adminCtr.GetSettings)
			admin.PUT("/settings", adminCtr.UpdateSettings)
		}
	}

	// Static file serving for the built frontend (NoRoute = only fires when no
	// /api route matches, which is how SPA + hash routing works).
	if cfg.Server.StaticDir != "" {
		r.NoRoute(middleware.StaticFiles(cfg.Server.StaticDir))
	}

	return r
}

func configureTrustedProxies(r *gin.Engine, proxies []string) error {
	if len(proxies) == 0 {
		return r.SetTrustedProxies(nil)
	}
	return r.SetTrustedProxies(proxies)
}
