package smsprovider

type Response struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Provider  string `json:"provider,omitempty"`
}
