package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Upload   UploadConfig   `yaml:"upload"`
}

type ServerConfig struct {
	Port               string   `yaml:"port"`
	StaticDir          string   `yaml:"static_dir"`
	AllowedOrigins     []string `yaml:"allowed_origins"`
	RateLimitLocalhost bool     `yaml:"rate_limit_localhost"`
	TrustedProxies     []string `yaml:"trusted_proxies"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type UploadConfig struct {
	Dir                string   `yaml:"dir"`
	MaxSize            int64    `yaml:"max_size"`
	DefaultExpireHours int      `yaml:"default_expire_hours"`
	CodeLength         int      `yaml:"code_length"`
	ReceiveCodeLength  int      `yaml:"receive_code_length"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	AllowedMimeTypes   []string `yaml:"allowed_mime_types"`
	MaxExpireHours     int      `yaml:"max_expire_hours"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:      "8080",
			StaticDir: "../web/dist",
		},
		Database: DatabaseConfig{
			Path: "data/send.db",
		},
		Upload: UploadConfig{
			Dir:                "uploads",
			MaxSize:            30 << 20,
			DefaultExpireHours: 168,
			CodeLength:         8,
			ReceiveCodeLength:  8,
			AllowedExtensions:  []string{".zip", ".pdf", ".png", ".jpg", ".jpeg", ".gif", ".mp4", ".txt", ".doc", ".docx", ".xls", ".xlsx", ".7z", ".rar", ".csv", ".json", ".md"},
			AllowedMimeTypes: []string{
				"application/zip", "application/pdf", "application/x-rar-compressed", "application/x-7z-compressed",
				"image/png", "image/jpeg", "image/gif",
				"video/mp4",
				"text/plain", "text/csv", "text/markdown",
				"application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/json",
			},
			MaxExpireHours: 8760,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		applyEnvOverrides(cfg) // env still wins when no YAML file exists
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(cfg)
	return cfg, nil
}

// applyEnvOverrides lets container deployments override the YAML file via
// environment variables. Priority is: env > YAML file > defaults — so a YAML
// file ships defaults into the image and env (or nothing) adjusts them at run
// time. Values are sanitized against the YAML schema; unrecognized variables
// are ignored silently.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SEND_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("SEND_STATIC_DIR"); v != "" {
		cfg.Server.StaticDir = v
	}
	if v := os.Getenv("SEND_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("SEND_UPLOAD_DIR"); v != "" {
		cfg.Upload.Dir = v
	}
	if v := os.Getenv("SEND_MAX_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			cfg.Upload.MaxSize = n
		}
	}
	if v := os.Getenv("SEND_RATE_LIMIT_LOCALHOST"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Server.RateLimitLocalhost = b
		}
	}
	// List-valued settings: override only when the env value contains at least
	// one real entry. An empty/whitespace-only value means "leave whatever the
	// YAML said" — overriding with an empty slice would silently reset origins
	// or proxies, changing CORS/proxy-trust behavior behind someone's back.
	if v := os.Getenv("SEND_ALLOWED_ORIGINS"); v != "" {
		if l := splitList(v); len(l) > 0 {
			cfg.Server.AllowedOrigins = l
		}
	}
	if v := os.Getenv("SEND_TRUSTED_PROXIES"); v != "" {
		if l := splitList(v); len(l) > 0 {
			cfg.Server.TrustedProxies = l
		}
	}
}

// splitList splits a comma-separated environment value into a slice,
// trimming whitespace and dropping empty entries so ", " mistakes don't
// introduce a bogus 403 (empty origin) or "" proxy entry.
func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
