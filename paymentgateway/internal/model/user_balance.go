package model

import "time"

type UserBalance struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null"`
	Balance   float64   `gorm:"default:0"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
