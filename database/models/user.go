package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string         `gorm:"unique" json:"email"`
	Password  string         `json:"-"`
	Sessions  []Session      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"uat"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"cat"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Session struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	JTI          string    `gorm:"unique" json:"jti"`
	UserID       uint      `json:"uid"`
	User         User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
	RefreshToken string    `json:"-"`
	Revoked      bool      `gorm:"default:false" json:"revoked"`
	IssuedAt     time.Time `gorm:"autoCreateTime" json:"iat"`
	ExpiresAt    time.Time `json:"exp"`
}
