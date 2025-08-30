package v1

type SendMessageRequest struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Text      string `json:"text"`
	MessageID string `json:"messageID"`
}
