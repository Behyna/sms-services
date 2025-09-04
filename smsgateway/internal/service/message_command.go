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
