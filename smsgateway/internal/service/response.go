package service

type CreateMessageResponse struct {
	MessageID int64 `json:"message_id"`
}

type GetMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int64     `json:"total"`
}

type Message struct {
	MessageID string `json:"message_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Text      string `json:"text"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}
