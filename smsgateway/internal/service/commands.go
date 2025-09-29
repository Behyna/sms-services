package service

type CreateMessageCommand struct {
	ClientMessageID string
	FromMSISDN      string
	ToMSISDN        string
	Text            string
}

type SendMessageCommand struct {
	MessageID  int64  `json:"message_id"`
	FromMSISDN string `json:"from_msisdn"`
	ToMSISDN   string `json:"to_msisdn"`
	Text       string `json:"text"`
}

type GetMessagesQuery struct {
	UserID string
	Limit  int
	Offset int
}

type UpdateMessageToSendingCommand struct {
	MessageID    int64
	AttemptCount int
}

type UpdateMessageSuccessCommand struct {
	MessageID     int64
	ProviderMsgID string
	Provider      string
}

type UpdateMessageFailureCommand struct {
	MessageID int64
	LastError string
}

type ChargePaymentCommand struct {
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

type RefundPaymentCommand struct {
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ProcessRefundCommand struct {
	TxLogID         int64  `json:"tx_log_id"`
	MessageID       int64  `json:"message_id"`
	ClientMessageID string `json:"client_message_id"`
	FromMSISDN      string `json:"from_msisdn"`
	Amount          int    `json:"amount"`
}
