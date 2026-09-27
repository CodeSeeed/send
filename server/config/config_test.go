package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("SEND_PORT", "9090")
	t.Setenv("SEND_STATIC_DIR", "/static")
	t.Setenv("SEND_DB_PATH", "/data/db.sqlite")
	t.Setenv("SEND_UPLOAD_DIR", "/data/uploads")
	t.Setenv("SEND_MAX_SIZE", "1048576")
	t.Setenv("SEND_RATE_LIMIT_LOCALHOST", "true")
	t.Setenv("SEND_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")
	t.Setenv("SEND_TRUSTED_PROXIES", "127.0.0.1, ::1")

	cfg, err := Load(filepath.Join("testdata", "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Server.Port)
	}
	if cfg.Server.StaticDir != "/static" {
		t.Errorf("StaticDir = %q, want /static", cfg.Server.StaticDir)
	}
	if cfg.Database.Path != "/data/db.sqlite" {
		t.Errorf("Database.Path = %q, want /data/db.sqlite", cfg.Database.Path)
	}
	if cfg.Upload.Dir != "/data/uploads" {
		t.Errorf("Upload.Dir = %q, want /data/uploads", cfg.Upload.Dir)
	}
	if cfg.Server.RateLimitLocalhost != true {
		t.Errorf("RateLimitLocalhost = %v, want true", cfg.Server.RateLimitLocalhost)
	}
	wantOrigins := []string{"https://a.example.com", "https://b.example.com"}
	if got := cfg.Server.AllowedOrigins; !equalStrings(got, wantOrigins) {
		t.Errorf("AllowedOrigins = %v, want %v", got, wantOrigins)
	}
	wantProxies := []string{"127.0.0.1", "::1"}
	if got := cfg.Server.TrustedProxies; !equalStrings(got, wantProxies) {
		t.Errorf("TrustedProxies = %v, want %v", got, wantProxies)
	}
	if cfg.Upload.MaxSize != 1048576 {
		t.Errorf("MaxSize = %d, want 1048576", cfg.Upload.MaxSize)
	}
}

func TestApplyEnvOverridesIgnoresGarbage(t *testing.T) {
	t.Setenv("SEND_MAX_SIZE", "not-a-number")
	t.Setenv("SEND_RATE_LIMIT_LOCALHOST", "maybe")

	cfg, err := Load(filepath.Join("testdata", "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Upload.MaxSize != 30<<20 {
		t.Errorf("MaxSize = %d, want default 30<<20", cfg.Upload.MaxSize)
	}
	if cfg.Server.RateLimitLocalhost {
		t.Error("RateLimitLocalhost = true, want default false")
	}
}

func TestApplyEnvOverridesEmptyListKeysAreIgnored(t *testing.T) {
	// A YAML file with origins present, and an env value that parses to an
	// empty list — the env must NOT clobber the file's origins.
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	const yamlContent = "server:\n  allowed_origins:\n    - https://from-yaml.example.com\n"
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SEND_ALLOWED_ORIGINS", " , , ")

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{"https://from-yaml.example.com"}
	if got := cfg.Server.AllowedOrigins; !equalStrings(got, want) {
		t.Errorf("AllowedOrigins = %v, want %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
