package model

import "time"

type UserBalance struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Balance     float64   `gorm:"default:0"`
	LastUpdated time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
