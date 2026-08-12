package service

import (
	"errors"
	"net/url"
	"strings"

	"send/server/model"

	"gorm.io/gorm"
)

type SettingsService struct {
	db *gorm.DB
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

func (s *SettingsService) Get() (*model.SystemSetting, error) {
	var setting model.SystemSetting
	if err := s.db.First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *SettingsService) Initialize(maxFileSize int64, defaultExpireHours int) error {
	var count int64
	if err := s.db.Model(&model.SystemSetting{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.db.Create(&model.SystemSetting{
		MaxFileSize:        maxFileSize,
		DefaultExpireHours: defaultExpireHours,
	}).Error
}

func normalizeBaseURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", errors.New("当前URL必须是完整的 http 或 https 地址")
	}
	return value, nil
}

func (s *SettingsService) Update(maxFileSize int64, baseURL string) error {
	if maxFileSize <= 0 {
		return errors.New("文件大小限制必须大于0")
	}
	normalizedURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return err
	}
	var setting model.SystemSetting
	if err := s.db.First(&setting).Error; err != nil {
		return err
	}
	return s.db.Model(&setting).Updates(map[string]interface{}{
		"max_file_size": maxFileSize,
		"base_url":      normalizedURL,
	}).Error
}
