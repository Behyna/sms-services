package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"go.uber.org/zap"
)

type MessageService interface {
	CreateMessage(ctx context.Context, cmd CreateMessageCommand) (CreateMessageResponse, error)
	GetMessagesByUserID(ctx context.Context, cmd GetMessagesQuery) (GetMessagesResponse, error)
}

type message struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository
	txManager   repository.TxManager
	payment     PaymentService
	logger      *zap.Logger
}

func NewMessageService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	txManager repository.TxManager, payment PaymentService, logger *zap.Logger) MessageService {
	return &message{messageRepo: messageRepo, txLogRepo: txLogRepo, txManager: txManager, payment: payment, logger: logger}
}

func (m *message) CreateMessage(ctx context.Context, cmd CreateMessageCommand) (
	CreateMessageResponse, error) {

	idempotencyKey := fmt.Sprintf("charge-%s-%s", cmd.FromMSISDN, cmd.ClientMessageID)
	request := ChargePaymentCommand{UserID: cmd.FromMSISDN, Amount: 1, IdempotencyKey: idempotencyKey}

	err := m.payment.Charge(ctx, request)
	if err != nil {
		m.logger.Debug("Message creation aborted due to payment failure",
			zap.String("clientMessageID", cmd.ClientMessageID))
		return CreateMessageResponse{}, err
	}

	resp, err := m.createMessageTx(ctx, cmd)
	if err == nil {
		m.logger.Info("Message created successfully",
			zap.Int64("messageID", resp.MessageID),
			zap.String("clientMessageID", cmd.ClientMessageID))
		return resp, nil
	}

	m.logger.Error("Critical: Payment succeeded but message creation failed, initiating refund",
		zap.String("clientMessageID", cmd.ClientMessageID))

	idempotencyKey = fmt.Sprintf("refund-%s-%s", cmd.FromMSISDN, cmd.ClientMessageID)
	refundReq := RefundPaymentCommand{UserID: cmd.FromMSISDN, Amount: request.Amount, IdempotencyKey: idempotencyKey}

	refundErr := m.payment.Refund(ctx, refundReq)
	if refundErr != nil {
		m.logger.Error("CRITICAL: User charged without service - manual intervention required",
			zap.String("clientMessageID", cmd.ClientMessageID))

		// TODO: alerting for manual investigation
	}

	m.logger.Warn("Payment refunded after DB failure", zap.String("clientMessageID", cmd.ClientMessageID))

	return CreateMessageResponse{}, err
}

func (m *message) GetMessagesByUserID(ctx context.Context, cmd GetMessagesQuery) (GetMessagesResponse, error) {
	messages, err := m.messageRepo.GetByUserID(cmd.UserID, cmd.Limit, cmd.Offset)
	if err != nil {
		m.logger.Error("Failed to get messages by user ID",
			zap.String("user_id", cmd.UserID),
			zap.Error(err))
		return GetMessagesResponse{}, ErrDatabase
	}

	if len(messages) == 0 {
		return GetMessagesResponse{}, NewServiceError(constants.ErrCodeUserNotFound, err)
	}

	total, err := m.messageRepo.CountByUserID(cmd.UserID)
	if err != nil {
		m.logger.Error("Failed to count messages by user ID",
			zap.String("user_id", cmd.UserID),
			zap.Error(err))
		return GetMessagesResponse{}, ErrDatabase
	}

	responseMessages := make([]Message, len(messages))
	for i, msg := range messages {
		responseMessages[i] = Message{
			MessageID: msg.ClientMessageID,
			From:      msg.FromMSISDN,
			To:        msg.ToMSISDN,
			Text:      msg.Text,
			Status:    string(msg.Status),
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return GetMessagesResponse{
		Messages: responseMessages,
		Total:    int64(total),
	}, nil
}

func (m *message) createMessageTx(ctx context.Context, cmd CreateMessageCommand) (CreateMessageResponse, error) {
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
