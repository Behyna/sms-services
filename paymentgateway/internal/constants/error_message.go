package constants

const MessageErrorFormat = "The '%s' format is invalid"

const (
	ErrCodeUserExisted         = "user_already_exists"
	ErrCodeUserNotFound        = "user_not_found"
	ErrCodeOperationFailed     = "operation_failed"
	ErrCodeInsufficientBalance = "insufficient_balance"
	ErrCodeValidationFailed    = "validation_failed"
)

const (
	ErrMsgUserExisted         = "user balance already exists"
	ErrMsgOperationFailed     = "operation failed"
	ErrMsgUserNotFound        = "user not found"
	ErrMsgInsufficientBalance = "insufficient balance"
)

var errorMessages = map[string]string{
	ErrCodeUserExisted:         ErrMsgUserExisted,
	ErrCodeUserNotFound:        ErrMsgUserNotFound,
	ErrCodeOperationFailed:     ErrMsgOperationFailed,
	ErrCodeInsufficientBalance: ErrMsgInsufficientBalance,
}

func GetErrorMessage(code string) string {
	msg, exists := errorMessages[code]
	if !exists {
		return ""
	}
	return msg
}
