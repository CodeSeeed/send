package model

import "time"

type SystemSetting struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	MaxFileSize        int64     `json:"max_file_size"`
	DefaultExpireHours int       `json:"default_expire_hours"`
	BaseURL            string    `gorm:"size:512" json:"base_url"`
	UpdatedAt          time.Time `json:"updated_at"`
}
