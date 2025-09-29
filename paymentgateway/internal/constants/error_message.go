package constants

const MessageErrorFormat = "The '%s' format is invalid"

const (
	ErrCodeUserExisted         = "USER_ALREADY_EXISTS"
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeOperationFailed     = "OPERATION_FAILED"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
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
	ErrCodeInsufficientBalance: ErrMsgInsufficientBalance,
}

func GetErrorMessage(code string) string {
	msg, exists := errorMessages[code]
	if !exists {
		return ""
	}
	return msg
}
