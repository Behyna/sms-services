package service

import (
	"context"
	"errors"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"go.uber.org/zap"
)

type MessageService interface {
	CreateMessageTx(ctx context.Context, cmd CreateMessageCommand) (CreateMessageResponse, error)
	GetMessageForProcessing(ctx context.Context, messageID int64) (*model.Message, error)
	UpdateMessageToSending(ctx context.Context, cmd UpdateMessageToSendingCommand) error
	UpdateMessageSucceed(ctx context.Context, cmd UpdateMessageSuccessCommand) error
	UpdateMessageToPermanentFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error
	UpdateMessageToTemporaryFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error
}

type message struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository
	txManager   repository.TxManager
	logger      *zap.Logger
}

func NewMessageService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	txManager repository.TxManager, logger *zap.Logger) MessageService {
	return &message{messageRepo: messageRepo, txLogRepo: txLogRepo, txManager: txManager, logger: logger}
}

func (m *message) CreateMessageTx(ctx context.Context, cmd CreateMessageCommand) (CreateMessageResponse, error) {
	message := model.Message{
		ClientMessageID: cmd.ClientMessageID,
		FromMSISDN:      cmd.FromMSISDN,
		ToMSISDN:        cmd.ToMSISDN,
		Text:            cmd.Text,
		Status:          model.MessageStatusCreated,
		AttemptCount:    0,
		LastAttemptAt:   nil,
		Provider:        nil,
		ProviderMsgID:   nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	txLog := model.TxLog{
		FromMSISDN:  cmd.FromMSISDN,
		Amount:      1,
		State:       model.TxLogStateCreated,
		Published:   false,
		PublishedAt: nil,
		LastError:   nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := m.txManager.WithTx(ctx, func(ctx context.Context) error {
		err := m.messageRepo.Create(ctx, &message)
		if err != nil && errors.Is(err, repository.ErrMessageDuplicate) {
			m.logger.Warn("Duplicate message detected",
				zap.String("fromMSISDN", cmd.FromMSISDN),
				zap.String("clientMessageID", cmd.ClientMessageID))
			return NewServiceError(constants.ErrCodeDuplicateMessage, err)
		}

		if err != nil {
			m.logger.Warn("Failed to create message", zap.Error(err))
			return NewServiceError(ErrCodeDatabase, err)
		}

		txLog.MessageID = message.ID

		if err := m.txLogRepo.Create(ctx, &txLog); err != nil {
			m.logger.Warn("Failed to create transaction log", zap.Error(err))
			return NewServiceError(ErrCodeDatabase, err)
		}

		return nil
	})

	if err != nil {
		m.logger.Error("Message transaction failed",
			zap.String("clientMessageID", cmd.ClientMessageID),
			zap.Error(err))
		return CreateMessageResponse{}, err
	}

	return CreateMessageResponse{MessageID: message.ID}, nil
}

func (m *message) GetMessageForProcessing(ctx context.Context, messageID int64) (*model.Message, error) {
	msg, err := m.messageRepo.GetByID(messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return nil, ErrMessageNotFound
		}

		return nil, ErrDatabase
	}

	switch msg.Status {
	case model.MessageStatusCreated:
		return msg, nil

	case model.MessageStatusSending:
		if msg.LastAttemptAt != nil && time.Since(*msg.LastAttemptAt) < 5*time.Minute {
			m.logger.Warn("Message being processed by another consumer",
				zap.Int64("messageID", messageID),
				zap.Time("lastAttempt", *msg.LastAttemptAt))
			return nil, ErrMessageBeingProcessed
		}

		return msg, nil

	case model.MessageStatusSubmitted, model.MessageStatusFailedPerm, model.MessageStatusRefunded:
		m.logger.Info("Message already processed successfully",
			zap.Int64("messageID", messageID), zap.String("status", string(msg.Status)))
		return nil, ErrMessageAlreadyProcessed

	case model.MessageStatusFailedTemp:
		m.logger.Info("Message was temporarily failed, retrying", zap.Int64("messageID", messageID))
		return msg, nil

	default:
		m.logger.Error("Unknown message status",
			zap.String("status", string(msg.Status)),
			zap.Int64("messageID", messageID))
		return nil, ErrUnknownMessageStatus
	}
}

func (m *message) UpdateMessageToSending(ctx context.Context, cmd UpdateMessageToSendingCommand) error {
	staleThreshold := time.Now().Add(-5 * time.Minute)

	attempt := time.Now()
	msg := model.Message{
		ID:            cmd.MessageID,
		Status:        model.MessageStatusSending,
		AttemptCount:  cmd.AttemptCount,
		LastAttemptAt: &attempt,
		UpdatedAt:     time.Now(),
	}

	err := m.messageRepo.UpdateForSending(ctx, &msg, staleThreshold)
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrNoRowsAffected) {
		m.logger.Info("Message not updated to SENDING, possibly processed by another consumer",
			zap.Int64("messageID", cmd.MessageID))

		return nil
	}

	m.logger.Error("Failed to update message for send attempt",
		zap.Error(err),
		zap.Int64("messageID", cmd.MessageID))

	return ErrDatabase
}

func (m *message) UpdateMessageSucceed(ctx context.Context, cmd UpdateMessageSuccessCommand) error {
	msg := model.Message{
		ID:            cmd.MessageID,
		Status:        model.MessageStatusSubmitted,
		ProviderMsgID: &cmd.ProviderMsgID,
		Provider:      &cmd.Provider,
		UpdatedAt:     time.Now(),
	}

	if err := m.messageRepo.Update(ctx, &msg); err != nil {
		m.logger.Error("Failed to update message after send attempt",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("providerMessageID", cmd.ProviderMsgID),
			zap.String("provider", cmd.Provider),
			zap.Error(err))
	}

	txLog := model.TxLog{
		MessageID: cmd.MessageID,
		State:     model.TxLogStateSuccess,
		LastError: nil,
		UpdatedAt: time.Now(),
	}

	if err := m.txLogRepo.UpdateByMessageID(ctx, &txLog); err != nil {
		m.logger.Error("Failed to update tx_log to published",
			zap.Error(err),
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))
	}

	return nil
}

func (m *message) UpdateMessageToPermanentFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error {
	msg := &model.Message{
		ID:        cmd.MessageID,
		Status:    model.MessageStatusFailedPerm,
		UpdatedAt: time.Now(),
	}

	txLog := &model.TxLog{
		MessageID:   cmd.MessageID,
		State:       model.TxLogStateFailed,
		Published:   false,
		PublishedAt: nil,
		LastError:   &cmd.LastError,
		UpdatedAt:   time.Now(),
	}

	return m.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := m.messageRepo.Update(ctx, msg); err != nil {
			m.logger.Error("Failed to update message status after perm failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		if err := m.txLogRepo.UpdateForPermFailed(ctx, txLog); err != nil {
			m.logger.Error("Failed to update transaction log after perm failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		return nil
	})
}

func (m *message) UpdateMessageToTemporaryFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error {
	msg := model.Message{
		ID:        cmd.MessageID,
		Status:    model.MessageStatusFailedTemp,
		UpdatedAt: time.Now(),
	}

	txLog := model.TxLog{
		MessageID: cmd.MessageID,
		LastError: &cmd.LastError,
		UpdatedAt: time.Now(),
	}

	return m.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := m.messageRepo.Update(ctx, &msg); err != nil {
			m.logger.Error("Failed to update message status after temp failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		if err := m.txLogRepo.UpdateByMessageID(ctx, &txLog); err != nil {
			m.logger.Error("Failed to update transaction log after temp failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		return nil
	})
}
