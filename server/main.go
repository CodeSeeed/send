package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	// A fresh checkout does not contain the ignored runtime directories. Create
	// them before SQLite or the upload handler tries to write into them.
	databaseDir := filepath.Dir(cfg.Database.Path)
	if databaseDir != "." {
		if err := os.MkdirAll(databaseDir, 0o750); err != nil {
			log.Fatalf("创建数据库目录失败: %v", err)
		}
	}
	if err := os.MkdirAll(cfg.Upload.Dir, 0o750); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
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
	adminCtr := controller.NewAdminController(adminSvc, settingsSvc, len(cfg.Server.TrustedProxies) > 0)

	adminMW := middleware.AdminAuth(adminSvc)

	// Clean up orphaned files on startup (files without a DB record, e.g.
	// from a previous crash before the DB-first ordering was introduced).
	fileSvc.CleanupOrphans()

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
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
