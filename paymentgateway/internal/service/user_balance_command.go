package service

import (
	"time"

	"github.com/Behyna/sms-services/paymentgateway/internal/model"
)

type UserBalanceCommand struct {
	UserID         string
	Amount         int64
	IdempotencyKey string
}

type CreateUserResult struct {
	UserBalance     model.UserBalance `json:"user_balance"`
	TransactionID   int64             `json:"transaction_id"`
	TransactionTime time.Time         `json:"transaction_time"`
}

type UpdateBalanceResult struct {
	UserBalance     model.UserBalance `json:"user_balance"`
	TransactionID   int64             `json:"transaction_id"`
	TransactionTime time.Time         `json:"transaction_time"`
}
