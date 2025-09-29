package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"gorm.io/gorm"
)

var ErrMessageNotFound = errors.New("MESSAGE_NOT_FOUND")
var ErrMessageDuplicate = errors.New("MESSAGE_DUPLICATE")
var ErrNoRowsAffected = errors.New("NO_ROWS_AFFECTED")

type MessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	Update(ctx context.Context, message *model.Message) error
	UpdateForSending(ctx context.Context, message *model.Message, staleThreshold time.Time) error
	GetByID(id int64) (*model.Message, error)
}

type Message struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &Message{db: db}
}

func (m *Message) Create(ctx context.Context, message *model.Message) error {
	db := GetTx(ctx, m.db)
	err := db.Create(message).Error
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrMessageDuplicate
	}

	return err
}

func (m *Message) Update(ctx context.Context, message *model.Message) error {
	db := GetTx(ctx, m.db)
	return db.Model(message).Where("ID = ?", message.ID).Updates(message).Error
}

func (m *Message) UpdateForSending(ctx context.Context, message *model.Message, staleThreshold time.Time) error {
	db := GetTx(ctx, m.db)
	result := db.Model(message).Where("ID = ? AND (status IN (?, ?) OR (status = ? AND last_attempt_at < ?))",
		message.ID,
		model.TxLogStateCreated,
		model.MessageStatusFailedTemp,
		model.MessageStatusSending,
		staleThreshold).Updates(message)

	if result.RowsAffected == 0 {
		return ErrNoRowsAffected
	}

	return result.Error
}

func (m *Message) GetByID(id int64) (*model.Message, error) {
	var message model.Message

	err := m.db.Where("id = ?", id).First(&message).Error
	if err == nil {
		return &message, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMessageNotFound
	}

	return nil, err
}
