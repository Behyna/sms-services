package v1

type SendMessageResponse struct {
	Status    string `json:"status"`
	MessageID string `json:"messageID"`
	Duplicate bool   `json:"duplicate"`
}
