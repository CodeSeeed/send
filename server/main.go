package main

import (
	"fmt"
	"log"
	"time"

	"send/server/config"
	"send/server/controller"
	"send/server/middleware"
	"send/server/model"
	"send/server/router"
	"send/server/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	db.AutoMigrate(&model.File{}, &model.Admin{}, &model.SystemSetting{})

	fileSvc := service.NewFileService(db, cfg)
	adminSvc := service.NewAdminService(db)
	settingsSvc := service.NewSettingsService(db)
	if err := settingsSvc.Initialize(cfg.Upload.MaxSize, cfg.Upload.DefaultExpireHours); err != nil {
		log.Fatalf("初始化系统设置失败: %v", err)
	}

	fileCtr := controller.NewFileController(fileSvc, cfg, settingsSvc)
	adminCtr := controller.NewAdminController(adminSvc, settingsSvc)

	adminMW := middleware.AdminAuth(adminSvc)

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			fileSvc.CleanupExpired()
			service.CleanupExpiredTokens()
		}
	}()

	r := router.Setup(cfg, fileCtr, adminCtr, adminMW)

	fmt.Printf("服务启动于 :%s\n", cfg.Server.Port)
	r.Run(":" + cfg.Server.Port)
}
