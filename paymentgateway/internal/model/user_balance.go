package model

import "time"

type UserBalance struct {
	UserID    string    `gorm:"column:user_id;primaryKey;type:char(11)"`
	Balance   int64     `gorm:"column:balance;not null;default:0"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
}

func (UserBalance) TableName() string {
	return "user_balances"
}
