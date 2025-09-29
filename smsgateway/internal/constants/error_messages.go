package constants

const (
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodeDuplicateMessage    = "DUPLICATE_MESSAGE"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeInvalidRequestBody  = "INVALID_REQUEST_BODY"
)

const (
	ErrMsgUserNotFound        = "user not found"
	ErrMsgInsufficientBalance = "insufficient balance"
	ErrMsgDuplicateMessage    = "duplicate message"
	ErrMsgInternalError       = "Internal server error"
	ErrMsgInvalidRequestBody  = "failed to parse request body"
)

var errorMessages = map[string]string{
	ErrCodeUserNotFound:        ErrMsgUserNotFound,
	ErrCodeInsufficientBalance: ErrMsgInsufficientBalance,
	ErrCodeDuplicateMessage:    ErrMsgDuplicateMessage,
	ErrCodeInternalError:       ErrMsgInternalError,
	ErrCodeInvalidRequestBody:  ErrMsgInvalidRequestBody,
}

func GetErrorMessage(code string) string {
	if msg, exists := errorMessages[code]; exists {
		return msg
	}
	return ErrMsgInternalError
}

func GetHTTPStatus(code string) int {
	switch code {
	case ErrCodeInvalidRequestBody:
		return 400
	case ErrCodeUserNotFound:
		return 404
	case ErrCodeInsufficientBalance, ErrCodeDuplicateMessage:
		return 409
	case ErrCodeInternalError:
		return 500
	default:
		return 500
	}
}
