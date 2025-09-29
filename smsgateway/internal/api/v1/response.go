package v1

type SendMessageResponse struct {
	Status    string `json:"status"`
	MessageID int64  `json:"message_id"`
	Duplicate bool   `json:"duplicate"`
}

type GetMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
}

type MessageResponse struct {
	MessageID string `json:"message_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Text      string `json:"text"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}
