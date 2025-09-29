package v1

type SendMessageRequest struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Text      string `json:"text"`
	MessageID string `json:"message_id"`
}

type GetMessagesRequest struct {
	UserID string `query:"user_id"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}
