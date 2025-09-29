package model

import "time"

type MessageStatus string

const (
	MessageStatusCreated    MessageStatus = "CREATED"
	MessageStatusSending    MessageStatus = "SENDING"
	MessageStatusSubmitted  MessageStatus = "SUBMITTED"
	MessageStatusFailedTemp MessageStatus = "FAILED_TEMP"
	MessageStatusFailedPerm MessageStatus = "FAILED_PERM"
	MessageStatusRefunded   MessageStatus = "REFUNDED"
)

type Message struct {
	ID              int64         `gorm:"primaryKey;autoIncrement;column:id;<-:create"`
	ClientMessageID string        `gorm:"column:client_message_id;index:idx_client_msg_from,unique"`
	FromMSISDN      string        `gorm:"column:from_msisdn;index:idx_client_msg_from,unique"`
	ToMSISDN        string        `gorm:"column:to_msisdn"`
	Text            string        `gorm:"column:text"`
	Status          MessageStatus `gorm:"column:status"`
	AttemptCount    int           `gorm:"column:attempt_count"`
	LastAttemptAt   *time.Time    `gorm:"column:last_attempt_at"`
	Provider        *string       `gorm:"column:provider"`
	ProviderMsgID   *string       `gorm:"column:provider_msg_id"`
	CreatedAt       time.Time     `gorm:"column:created_at"`
	UpdatedAt       time.Time     `gorm:"column:updated_at"`
}
