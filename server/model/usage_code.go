package model

import "time"

type UsageCode struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"uniqueIndex;size:32" json:"code"`
	Remark      string     `gorm:"size:255" json:"remark"`
	MaxFileSize int64      `json:"max_file_size"` // bytes, 0 = no limit
	MaxUses     int        `json:"max_uses"`
	UsedCount   int        `gorm:"default:0" json:"used_count"`
	ExpireAt    *time.Time `json:"expire_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
