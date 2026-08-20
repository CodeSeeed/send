package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"send/server/config"
	"send/server/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCleanupOrphansReconcilesDiskAndDatabase(t *testing.T) {
	root := t.TempDir()
	uploadDir := filepath.Join(root, "uploads")
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		t.Fatalf("create upload dir: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.AutoMigrate(&model.File{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	createRecord := func(code, path string, size int64) {
		t.Helper()
		file := &model.File{
			Code:     code,
			FileName: code + ".bin",
			FilePath: path,
			FileSize: size,
		}
		if err := db.Create(file).Error; err != nil {
			t.Fatalf("create record %s: %v", code, err)
		}
	}

	validPath := filepath.Join(uploadDir, "valid")
	if err := os.WriteFile(validPath, []byte("valid"), 0o600); err != nil {
		t.Fatalf("write valid file: %v", err)
	}
	createRecord("valid", validPath, 5)

	missingPath := filepath.Join(uploadDir, "missing")
	createRecord("missing", missingPath, 7)

	partialPath := filepath.Join(uploadDir, "partial")
	if err := os.WriteFile(partialPath, []byte("bad"), 0o600); err != nil {
		t.Fatalf("write partial file: %v", err)
	}
	createRecord("partial", partialPath, 7)

	orphanPath := filepath.Join(uploadDir, "orphan")
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0o600); err != nil {
		t.Fatalf("write orphan file: %v", err)
	}

	svc := NewFileService(db, &config.Config{
		Upload: config.UploadConfig{Dir: uploadDir},
	})
	svc.CleanupOrphans()

	assertRecordCount := func(code string, want int64) {
		t.Helper()
		var count int64
		if err := db.Model(&model.File{}).Where("code = ?", code).Count(&count).Error; err != nil {
			t.Fatalf("count record %s: %v", code, err)
		}
		if count != want {
			t.Fatalf("record %s count = %d, want %d", code, count, want)
		}
	}

	assertRecordCount("valid", 1)
	assertRecordCount("missing", 0)
	assertRecordCount("partial", 0)

	if _, err := os.Stat(validPath); err != nil {
		t.Fatalf("valid file was removed: %v", err)
	}
	for _, path := range []string{partialPath, orphanPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("orphan path %s still exists or stat failed: %v", path, err)
		}
	}
}

func TestCreateFileWithOptionalReceiveCode(t *testing.T) {
	root := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "receive-code.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.AutoMigrate(&model.File{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	svc := NewFileService(db, &config.Config{
		Upload: config.UploadConfig{
			Dir:               filepath.Join(root, "uploads"),
			CodeLength:        8,
			ReceiveCodeLength: 8,
		},
	})
	withCode, err := svc.Create(&CreateFileParams{
		FileName:            "with-code.txt",
		FileSize:            1,
		GenerateReceiveCode: true,
	})
	if err != nil {
		t.Fatalf("create file with receive code: %v", err)
	}
	if withCode.ReceiveCode == nil || len(*withCode.ReceiveCode) != 8 {
		t.Fatalf("receive code = %v, want 8 characters", withCode.ReceiveCode)
	}
	if *withCode.ReceiveCode != strings.ToUpper(*withCode.ReceiveCode) {
		t.Fatalf("receive code = %q, want uppercase", *withCode.ReceiveCode)
	}

	info, err := svc.GetByReceiveCode(strings.ToLower(*withCode.ReceiveCode))
	if err != nil {
		t.Fatalf("look up receive code case-insensitively: %v", err)
	}
	if info.Code != withCode.Code {
		t.Fatalf("lookup code = %q, want %q", info.Code, withCode.Code)
	}

	withoutCode, err := svc.Create(&CreateFileParams{
		FileName: "without-code.txt",
		FileSize: 1,
	})
	if err != nil {
		t.Fatalf("create file without receive code: %v", err)
	}
	if withoutCode.ReceiveCode != nil {
		t.Fatalf("receive code = %q, want nil", *withoutCode.ReceiveCode)
	}
}

func TestDownloadLimitBlocksTokenVerificationAndPreview(t *testing.T) {
	root := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "download-limit.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.AutoMigrate(&model.File{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	filePath := filepath.Join(root, "limited.txt")
	limited := &model.File{
		Code:          "limited",
		FileName:      "limited.txt",
		FilePath:      filePath,
		FileSize:      7,
		DownloadCount: 1,
		MaxDownloads:  1,
	}
	if err := db.Create(limited).Error; err != nil {
		t.Fatalf("create limited file: %v", err)
	}

	svc := NewFileService(db, &config.Config{})
	if err := svc.VerifyPassword(limited.Code, ""); err == nil || err.Error() != "下载次数已达上限" {
		t.Fatalf("VerifyPassword() error = %v, want download limit error", err)
	}
	if _, _, err := svc.GetFilePathForPreview(limited.Code); err == nil || err.Error() != "下载次数已达上限" {
		t.Fatalf("GetFilePathForPreview() error = %v, want download limit error", err)
	}

	if err := db.Model(limited).UpdateColumn("download_count", 0).Error; err != nil {
		t.Fatalf("reset download count: %v", err)
	}
	if err := svc.VerifyPassword(limited.Code, ""); err != nil {
		t.Fatalf("VerifyPassword() below limit: %v", err)
	}
	path, name, err := svc.GetFilePathForPreview(limited.Code)
	if err != nil {
		t.Fatalf("GetFilePathForPreview() below limit: %v", err)
	}
	if path != filePath || name != limited.FileName {
		t.Fatalf("preview result = (%q, %q), want (%q, %q)", path, name, filePath, limited.FileName)
	}
}
