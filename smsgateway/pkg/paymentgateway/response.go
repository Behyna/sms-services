package paymentgateway

import "time"

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	TrackID string `json:"x_track_id,omitempty"`
	Result  Result `json:"result,omitempty"`
}

type Result struct {
	UserBalance     any       `json:"user_balance"`
	TransactionID   int64     `json:"transaction_id"`
	TransactionTime time.Time `json:"transaction_time"`
}
