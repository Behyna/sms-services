package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/common/pkg/mq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MessageService interface {
	CreateMessageTransaction(ctx context.Context, cmd CreateMessageCommand) error
	SendMessage(ctx context.Context, cmd SendMessageCommand) error
}

type Message struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository

	db     *gorm.DB
	logger *zap.Logger
	pub    mq.Publisher
}

func NewMessageService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	db *gorm.DB, logger *zap.Logger, publisher mq.Publisher) MessageService {
	return &Message{messageRepo: messageRepo, txLogRepo: txLogRepo, db: db, logger: logger, pub: publisher}
}

func (m *Message) CreateMessageTransaction(ctx context.Context, cmd CreateMessageCommand) error {
	// TODO call to credit.
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

	out := SendMessageCommand{
		MessageID:  message.ID,
		ToMSISDN:   cmd.ToMSISDN,
		FromMSISDN: cmd.FromMSISDN,
		Text:       cmd.Text,
	}

	body, _ := json.Marshal(out)
	if err := m.pub.Publish(ctx, "", "sms.send", body); err != nil {
		m.logger.Error("Failed to publish message to queue", zap.Error(err))
		return err
	}

	now := time.Now()
	txLog.Published = true
	txLog.PublishedAt = &now

	if err := m.txLogRepo.Update(txLog); err != nil {
		m.logger.Error("Failed to update transaction log", zap.Error(err))
		return err
	}

	return nil
}

func (m *Message) SendMessage(ctx context.Context, cmd SendMessageCommand) error {
	return nil
}
