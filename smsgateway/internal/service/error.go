package service

import "errors"

const (
	ErrCodeRefundTimeout       = "REFUND_TIMEOUT"
	ErrCodeChargeTimeout       = "CHARGE_TIMEOUT"
	ErrCodePaymentServiceError = "PAYMENT_SERVICE_ERROR"
	ErrCodeDatabase            = "DATABASE_ERROR"
)

var (
	ErrMessageNotFound         = errors.New("MESSAGE_NOT_FOUND")
	ErrMessageBeingProcessed   = errors.New("MESSAGE_BEING_PROCESSED")
	ErrMessageAlreadyProcessed = errors.New("MESSAGE_ALREADY_PROCESSED")
	ErrUnknownMessageStatus    = errors.New("UNKNOWN_MESSAGE_STATUS")
	ErrTxLogNotFound           = errors.New("TX_LOG_NOT_FOUND")
	ErrTxInvalidState          = errors.New("REFUND_INVALID_STATE")
	ErrRefundAlreadyProcessed  = errors.New("REFUND_ALREADY_PROCESSED")
	ErrUnknownTxState          = errors.New("UNKNOWN_TX_STATE")
	ErrDatabase                = errors.New("DATABASE_ERROR")
)

type Error struct {
	Code  string
	Cause error
}

func NewServiceError(code string, cause error) error {
	return Error{Code: code, Cause: cause}
}

func (e Error) Error() string {
	return e.Cause.Error()
}

func (e Error) Unwrap() error {
	return e.Cause
}
