package smsprovider

const (
	ErrorCodeServerError   = "SERVER_ERROR"   // For 5xx HTTP status
	ErrorCodeTimeout       = "TIMEOUT"        // For context timeout
	ErrorCodeInvalidNumber = "INVALID_NUMBER" // For 400/validation errors
	ErrorCodeNetworkError  = "NETWORK_ERROR"  // For connection failures
)
