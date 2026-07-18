package router

import (
	"send/server/controller"
	"send/server/middleware"

	"github.com/gin-gonic/gin"
)

var (
	uploadLimiter   = middleware.NewRateLimiter(middleware.TierUpload)
	adminLimiter    = middleware.NewRateLimiter(middleware.TierAdmin)
	loginLimiter    = middleware.NewRateLimiter(middleware.TierLogin)
	downloadLimiter = middleware.NewRateLimiter(middleware.TierDownload)
	verifyLimiter   = middleware.NewRateLimiter(middleware.TierDownload)
)

func Setup(
	fileCtr *controller.FileController,
	adminCtr *controller.AdminController,
	usageCodeCtr *controller.UsageCodeController,
	adminMW gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.SetupCORS())

	api := r.Group("/api")
	{
		// Login has its own strict limiter
		api.POST("/admin/login", loginLimiter.Middleware(), adminCtr.Login)

		// File upload
		files := api.Group("/files")
		{
			files.POST("", uploadLimiter.Middleware(), fileCtr.Upload)
			files.GET("/:code", fileCtr.Info)
			files.POST("/:code/verify", verifyLimiter.Middleware(), fileCtr.VerifyPassword)
			files.GET("/:code/download", downloadLimiter.Middleware(), fileCtr.Download)
		}

		// Admin routes with generous rate limit
		admin := api.Group("/admin")
		admin.Use(adminMW, adminLimiter.Middleware())
		{
			admin.POST("/password", adminCtr.ChangePassword)
			admin.GET("/files", adminCtr.ListFiles)
			admin.DELETE("/files/:id", adminCtr.DeleteFile)
			admin.POST("/usage-codes", usageCodeCtr.Create)
			admin.GET("/usage-codes", usageCodeCtr.List)
			admin.DELETE("/usage-codes/:id", usageCodeCtr.Delete)
		}
	}

	return r
}
