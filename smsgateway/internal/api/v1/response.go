package v1

type SendMessageResponse struct {
	Status    string `json:"status"`
	MessageID int64  `json:"message_id"`
	Duplicate bool   `json:"duplicate"`
}
