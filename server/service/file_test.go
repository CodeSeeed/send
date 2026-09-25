package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	if err := svc.VerifyPassword(limited.Code, ""); err != ErrDownloadLimit {
		t.Fatalf("VerifyPassword() error = %v, want ErrDownloadLimit", err)
	}
	if _, _, err := svc.GetFilePathForPreview(limited.Code); err != ErrDownloadLimit {
		t.Fatalf("GetFilePathForPreview() error = %v, want ErrDownloadLimit", err)
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

func TestSameClientIP(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		// IPv4 must match exactly — a different client is a different client.
		{"ipv4 identical", "203.0.113.5", "203.0.113.5", true},
		{"ipv4 differ last octet", "203.0.113.5", "203.0.113.9", false},
		{"ipv4 differ subnet", "203.0.113.5", "203.0.114.5", false},

		// IPv6 privacy-extension rotation stays within the same /64, so the
		// same client rotating its interface id should still validate.
		{"ipv6 same /64 rotated suffix", "2001:db8::1:2345:6789", "2001:db8::1:c9d1:e2f3", true},
		{"ipv6 different /64", "2001:db8:1::1", "2001:db8:2::1", false},
		{"ipv6 identical", "2001:db8::1", "2001:db8::1", true},

		// An IPv4 address and its IPv4-mapped-IPv6 form (::ffff:a.b.c.d) denote
		// the same 32-bit host, so they must match — a client does not become a
		// different client merely because the kernel reported its socket as v6.
		{"ipv4 vs mapped-v6 same host", "203.0.113.5", "::ffff:203.0.113.5", true},

		// Malformed or empty inputs never match a real address.
		{"empty vs v4", "", "203.0.113.5", false},
		{"malformed vs v4", "not-an-ip", "203.0.113.5", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameClientIP(tc.a, tc.b); got != tc.want {
				t.Fatalf("sameClientIP(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestDownloadTokenAcceptsIPv6PrivacyRotation(t *testing.T) {
	// A token minted for an IPv6 client must still validate after the client
	// rotates its privacy-extension suffix within the same /64, and must be
	// rejected once consumed or for a different /64.
	downloadTokens = sync.Map{}
	t.Cleanup(func() { downloadTokens = sync.Map{} })

	const code = "token-share"
	mint := "2001:db8::abcd:ef01"
	token := GenerateDownloadToken(code, mint)

	rotated := "2001:db8::1234:5678"
	if !ValidateDownloadTokenPeek(token, code, rotated) {
		t.Fatalf("peek with rotated IPv6 suffix rejected")
	}
	// Peeking must not consume the token.
	if !ValidateDownloadToken(token, code, rotated) {
		t.Fatalf("download with rotated IPv6 suffix rejected")
	}
	// Token is one-time — a second use must fail.
	if ValidateDownloadTokenPeek(token, code, rotated) {
		t.Fatalf("download token was not consumed by the prior download")
	}

	// A token minted for a different /64 must not validate.
	other := GenerateDownloadToken(code, "2001:db8:1::1")
	if ValidateDownloadTokenPeek(other, code, "2001:db8:2::1") {
		t.Fatalf("token validated across a different /64")
	}
}

func TestPublicFileInfoOmitsReceiveCodeAndDownloadProgress(t *testing.T) {
	// The public file-info/receive-code response must not expose the receive
	// code (capability separation) or download_count/max_downloads. This guards
	// against a future field being added to the struct and silently leaking.
	rc := "ABCDEFGH"
	got := publicFileInfoFromModel(&model.File{
		ID:            42,
		Code:          "pubcode",
		ReceiveCode:   &rc,
		FileName:      "secret.zip",
		FileSize:      1024,
		DownloadCount: 3,
		MaxDownloads:  10,
		PasswordHash:  "hashed",
	})

	if got.Code != "pubcode" {
		t.Fatalf("Code = %q, want %q", got.Code, "pubcode")
	}
	if !got.HasPassword {
		t.Fatalf("HasPassword = false, want true")
	}
	// PublicFileInfo has no ReceiveCode/DownloadCount/MaxDownloads fields at
	// all — the struct shape itself is the guarantee. Confirm by JSON round-trip
	// that none of the sensitive fields survive serialization.
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal public info: %v", err)
	}
	for _, needle := range []string{rc, "receive_code", "download_count", "max_downloads"} {
		if strings.Contains(string(raw), needle) {
			t.Fatalf("public JSON leaks %q: %s", needle, raw)
		}
	}
}
