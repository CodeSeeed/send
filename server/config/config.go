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
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type UploadConfig struct {
	Dir                string `yaml:"dir"`
	MaxSize            int64  `yaml:"max_size"`
	DefaultExpireHours int    `yaml:"default_expire_hours"`
	CodeLength         int    `yaml:"code_length"`
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
