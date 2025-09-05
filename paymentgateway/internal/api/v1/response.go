package v1

type UserResponse struct {
	Status  string  `json:"status"`
	UserID  int64   `json:"user_id"`
	Balance float64 `json:"balance"`
}
