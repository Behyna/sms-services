package service

import (
	"context"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MessageService interface {
	CreateMessageTransaction(ctx context.Context, cmd CreateMessageCommand) error
}

type Message struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository

	db     *gorm.DB
	logger *zap.Logger
}

func NewMessageService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	db *gorm.DB, logger *zap.Logger) MessageService {
	return &Message{messageRepo: messageRepo, txLogRepo: txLogRepo, db: db, logger: logger}
}

func (m *Message) CreateMessageTransaction(ctx context.Context, cmd CreateMessageCommand) error {
	message := model.Message{
		ClientMessageID: cmd.ClientMessageID,
		FromMSISDN:      cmd.FromMSISDN,
		ToMSISDN:        cmd.ToMSISDN,
		Text:            cmd.Text,
		Status:          model.MessageStatusQueued,
		AttemptCount:    0,
		LastAttemptAt:   nil,
		Provider:        nil,
		ProviderMsgID:   nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	txLog := &model.TxLog{
		FromMSISDN:  cmd.FromMSISDN,
		Amount:      1,
		State:       model.TxLogStatePending,
		Published:   false,
		PublishedAt: nil,
		LastError:   nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := m.db.Transaction(func(tx *gorm.DB) error {
		if err := m.messageRepo.CreateWithTx(tx, &message); err != nil {
			m.logger.Error("Failed to create message", zap.Error(err))
			return err
		}

		txLog.MessageID = message.ID

		if err := m.txLogRepo.CreateWithTx(tx, txLog); err != nil {
			m.logger.Error("Failed to create transaction log", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		m.logger.Error("Transaction failed", zap.Error(err))
		return err
	}

	return nil
}
