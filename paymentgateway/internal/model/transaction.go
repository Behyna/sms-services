package model

import "time"

type Transaction struct {
	TransactionID   int64     `gorm:"primaryKey;autoIncrement"`
	UserBalanceID   int64     `gorm:"not null"`
	Amount          float64     `gorm:"not null"`
	TransactionType string    `gorm:"type:varchar(20);not null"`
	Status          string    `gorm:"type:varchar(20);default:'completed'"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
