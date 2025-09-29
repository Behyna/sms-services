package paymentgateway

import "errors"

const (
	StatusOK                  = 200
	StatusNotFound            = 404
	StatusUnprocessableEntity = 422
	StatusConflict            = 409
)

const (
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeTimeout             = "TIMEOUT"
	ErrCodeServerError         = "SERVER_ERROR"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
)

var (
	ErrValidationFailed    = errors.New(ErrCodeValidationFailed)
	ErrUserNotFound        = errors.New(ErrCodeUserNotFound)
	ErrTimeout             = errors.New(ErrCodeTimeout)
	ErrServerError         = errors.New(ErrCodeServerError)
	ErrInsufficientBalance = errors.New(ErrCodeInsufficientBalance)
)

var statusErrorMap = map[int]error{
	StatusNotFound:            ErrUserNotFound,
	StatusUnprocessableEntity: ErrValidationFailed,
	StatusConflict:            ErrInsufficientBalance,
}

func MapStatusToError(statusCode int) error {
	if err, exists := statusErrorMap[statusCode]; exists {
		return err
	}

	return ErrServerError
}
