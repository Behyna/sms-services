package constants

import "errors"

const MessageErrorFormat = "The '%s' format is invalid"

var (
	ErrUserBalanceAlreadyExists error = errors.New("user balance already exists")
	ErrInsufficientBalance      error = errors.New("insufficient balance")
)
