package router

import (
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
	// Lazy init rate limiters with config
	skipLocal := cfg.Server.RateLimitLocalhost
	uploadLimiter = middleware.NewRateLimiter(middleware.TierUpload, skipLocal)
	adminLimiter = middleware.NewRateLimiter(middleware.TierAdmin, skipLocal)
	loginLimiter = middleware.NewRateLimiter(middleware.TierLogin, skipLocal)
	downloadLimiter = middleware.NewRateLimiter(middleware.TierDownload, skipLocal)
	verifyLimiter = middleware.NewRateLimiter(middleware.TierDownload, skipLocal)

	r := gin.Default()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		panic(err)
	}
	r.Use(middleware.SetupCORS(cfg))
	r.Use(middleware.SecurityHeaders())

	api := r.Group("/api")
	{
		// Login has its own strict limiter
		api.POST("/admin/login", loginLimiter.Middleware(), adminCtr.Login)
		api.GET("/admin/status", adminCtr.AdminStatus)
		api.POST("/admin/register", loginLimiter.Middleware(), adminCtr.Register)
		api.GET("/settings", adminCtr.GetSettings)

		// File upload (admin-only)
		files := api.Group("/files")
		{
			files.POST("", adminMW, uploadLimiter.Middleware(), fileCtr.Upload)
			files.GET("/:code", fileCtr.Info)
			files.POST("/:code/verify", verifyLimiter.Middleware(), fileCtr.VerifyPassword)
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

	return r
}