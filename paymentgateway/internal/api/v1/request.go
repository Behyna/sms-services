package v1

type CreateUserBalanceRequest struct {
	UserID         string `json:"user_id" validate:"required,len=11"`
	InitialBalance int64  `json:"initial_balance" validate:"required,min=1"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

type GetUserBalanceRequest struct {
	UserID string `json:"user_id" validate:"required,len=11"`
}

type UpdateUserBalanceRequest struct {
	UserID         string `json:"user_id" validate:"required,len=11"`
	Amount         int64  `json:"amount" validate:"required,min=1"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}
