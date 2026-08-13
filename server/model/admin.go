package model

import "time"

type Admin struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Username       string     `gorm:"uniqueIndex;size:64" json:"username"`
	Password       string     `gorm:"size:255" json:"-"`
	Token          string     `gorm:"size:64" json:"-"`
	TokenExpiresAt *time.Time `json:"-"`
}
