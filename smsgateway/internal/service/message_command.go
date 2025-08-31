package service

type CreateMessageCommand struct {
	ClientMessageID string
	FromMSISDN      string
	ToMSISDN        string
	Text            string
}
