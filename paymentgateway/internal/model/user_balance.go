package model

import "time"

type UserBalance struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    string    `gorm:"not null"`
	Balance   int64     `gorm:"default:0"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
