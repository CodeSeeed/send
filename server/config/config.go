package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Upload   UploadConfig   `yaml:"upload"`
}

type ServerConfig struct {
	Port              string `yaml:"port"`
	AllowedOrigins    []string `yaml:"allowed_origins"`
	RateLimitLocalhost bool   `yaml:"rate_limit_localhost"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type UploadConfig struct {
	Dir                string   `yaml:"dir"`
	MaxSize            int64    `yaml:"max_size"`
	DefaultExpireHours int      `yaml:"default_expire_hours"`
	CodeLength         int      `yaml:"code_length"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	AllowedMimeTypes   []string `yaml:"allowed_mime_types"`
	MaxExpireHours     int      `yaml:"max_expire_hours"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Database: DatabaseConfig{
			Path: "data/send.db",
		},
		Upload: UploadConfig{
			Dir:                "uploads",
			MaxSize:            30 << 20,
			DefaultExpireHours: 168,
			CodeLength:         8,
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
		return cfg, nil // return defaults if no config file
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
