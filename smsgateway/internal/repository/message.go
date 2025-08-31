package repository

import (
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"gorm.io/gorm"
)

type MessageRepository interface {
	CreateWithTx(tx *gorm.DB, msg *model.Message) error
}

type Message struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &Message{db: db}
}

func (r *Message) CreateWithTx(tx *gorm.DB, message *model.Message) error {
	if err := tx.Create(message).Error; err != nil {
		return err
	}

	return nil
}
