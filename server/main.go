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
	"send/server/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	db.AutoMigrate(&model.File{}, &model.Admin{}, &model.UsageCode{})

	seedAdmin(db)

	fileSvc := service.NewFileService(db, cfg)
	adminSvc := service.NewAdminService(db)
	usageCodeSvc := service.NewUsageCodeService(db)

	fileCtr := controller.NewFileController(fileSvc, cfg, adminSvc, usageCodeSvc)
	adminCtr := controller.NewAdminController(adminSvc)
	usageCodeCtr := controller.NewUsageCodeController(usageCodeSvc)

	adminMW := middleware.AdminAuth(adminSvc)

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			fileSvc.CleanupExpired()
			service.CleanupExpiredTokens()
		}
	}()

	r := router.Setup(fileCtr, adminCtr, usageCodeCtr, adminMW)

	fmt.Printf("服务启动于 :%s\n", cfg.Server.Port)
	r.Run(":" + cfg.Server.Port)
}

func seedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&model.Admin{}).Count(&count)
	if count > 0 {
		return
	}
	hash, err := utils.HashPassword("123456")
	if err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}
	admin := model.Admin{
		Username: "codeseed",
		Password: hash,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("创建管理员失败: %v", err)
	}
	log.Println("已创建默认管理员: codeseed / 123456")
}
