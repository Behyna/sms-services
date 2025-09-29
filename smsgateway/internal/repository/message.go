package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/go-sql-driver/mysql"
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
	GetByUserID(userID string, limit, offset int) ([]model.Message, error)
	CountByUserID(userID string) (int, error)
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

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
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

func (m *Message) GetByUserID(userID string, limit, offset int) ([]model.Message, error) {
	var messages []model.Message

	err := m.db.Where("from_msisdn = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (m *Message) CountByUserID(userID string) (int, error) {
	var count int64

	err := m.db.Model(&model.Message{}).
		Where("from_msisdn = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return int(count), nil
}
