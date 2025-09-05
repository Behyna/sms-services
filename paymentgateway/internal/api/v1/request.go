package v1

type CreateUserBalanceRequest struct {
	UserID         int64   `json:"user_id" validate:"required,min=1"`
	InitialBalance float64 `json:"initial_balance" validate:"required,min=1"`
}

type GetUserBalanceRequest struct {
	UserID int64 `json:"user_id" validate:"required,min=1"`
}

type UpdateUserBalanceRequest struct {
	UserID int64   `json:"user_id" validate:"required,min=1"`
	Amount float64 `json:"amount" validate:"required,min=1"`
}
