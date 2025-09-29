package model

import "time"

const (
	TxLogStateCreated  = "CREATED"
	TxLogStatePending  = "PENDING"
	TxLogStateSuccess  = "SUCCESS"
	TxLogStateRefunded = "REFUNDED"
	TxLogStateFailed   = "FAILED"
)

type TxLog struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;<-:create"`
	MessageID   int64      `gorm:"not null;<-:create"`
	FromMSISDN  string     `gorm:"type:varchar(255);not null"`
	Amount      int        `gorm:"default:1;not null"`
	State       string     `gorm:"type:enum('CREATED','PENDING','SUCCESS','REFUNDED','FAILED');not null"`
	Published   bool       `gorm:"default:false;not null"`
	PublishedAt *time.Time `gorm:"type:timestamp;null"`
	LastError   *string    `gorm:"type:text;null"`
	CreatedAt   time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `gorm:"type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`

	Message Message `gorm:"foreignKey:MessageID"`
}
