package model

import "time"

type TxType string

const (
	TxTypeIncrease TxType = "INCREASE"
	TxTypeDecrease TxType = "DECREASE"
)

type Transaction struct {
	TransactionID  int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         string    `gorm:"column:user_id;not null"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:varchar(36);not null"`
	TxType         TxType    `gorm:"column:tx_type;type:enum('INCREASE','DECREASE');not null"`
	Amount         int64     `gorm:"column:amount;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
}

func (Transaction) TableName() string {
	return "transactions"
}
