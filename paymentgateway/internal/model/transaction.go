package model

import "time"

type TxType string

const (
	TxTypeIncrease TxType = "increase"
	TxTypeDecrease TxType = "decrease"
)

type Transaction struct {
	TransactionID int64     `gorm:"primaryKey;autoIncrement"`
	UserID        string    `gorm:"not null"`
	Amount        int64     `gorm:"not null"`
	TxType        TxType    `gorm:"type:varchar(20);not null"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
