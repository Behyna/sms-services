package paymentgateway

type UpdateUserBalanceRequest struct {
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}
