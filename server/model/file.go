package model

import "time"

type File struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Code          string     `gorm:"uniqueIndex;size:32" json:"code"`
	FileName      string     `gorm:"size:255" json:"file_name"`
	FilePath      string     `gorm:"size:255" json:"-"`
	FileSize      int64      `json:"file_size"`
	PasswordHash  string     `gorm:"size:255" json:"-"`
	ManageToken   string     `gorm:"index;size:64" json:"-"`
	DownloadCount int64      `gorm:"default:0" json:"download_count"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
