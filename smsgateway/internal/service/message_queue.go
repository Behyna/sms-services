package service

import (
	"context"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"go.uber.org/zap"
)

type MessageQueueService interface {
	FindMessagesToQueue(ctx context.Context, limit int) ([]SendMessageCommand, error)
	MarkMessageAsQueued(ctx context.Context, messageID int64) error
	FindRefundsToQueue(ctx context.Context, limit int) ([]ProcessRefundCommand, error)
	MarkRefundAsQueued(ctx context.Context, txLogID int64) error
}

type messageQueue struct {
	txLog  repository.TxLogRepository
	logger *zap.Logger
}

func NewMessageQueueService(txLogRepo repository.TxLogRepository, logger *zap.Logger) MessageQueueService {
	return &messageQueue{txLog: txLogRepo, logger: logger}
}

func (m *messageQueue) FindMessagesToQueue(ctx context.Context, limit int) ([]SendMessageCommand, error) {
	m.logger.Debug("Finding messages to publish", zap.Int("batchSize", limit))

	txLogs, err := m.txLog.FindUnpublishedCreated(limit)
	if err != nil {
		m.logger.Error("Failed to find unpublished messages", zap.Error(err))
		return nil, err
	}

	if len(txLogs) == 0 {
		m.logger.Debug("No messages found to publish")
		return nil, nil
	}

	messages := make([]SendMessageCommand, 0, len(txLogs))
	for _, log := range txLogs {
		msg := SendMessageCommand{
			MessageID:  log.MessageID,
			FromMSISDN: log.FromMSISDN,
			ToMSISDN:   log.Message.ToMSISDN,
			Text:       log.Message.Text,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (m *messageQueue) MarkMessageAsQueued(ctx context.Context, messageID int64) error {
	publishedAt := time.Now()
	txLog := model.TxLog{
		MessageID:   messageID,
		State:       model.TxLogStatePending,
		Published:   true,
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
	}

	if err := m.txLog.UpdateByMessageID(ctx, &txLog); err != nil {
		m.logger.Error("Failed to update tx_log to published",
			zap.Error(err),
			zap.Int64("messageID", messageID))
		return err
	}

	m.logger.Debug("Successfully marked message as published",
		zap.Int64("messageID", messageID))

	return nil
}

func (m *messageQueue) FindRefundsToQueue(ctx context.Context, limit int) ([]ProcessRefundCommand, error) {
	m.logger.Debug("Finding refunds to publish", zap.Int("batchSize", limit))

	failedTxLogs, err := m.txLog.FindUnpublishedFailed(limit)
	if err != nil {
		m.logger.Error("Failed to find unpublished failed transactions", zap.Error(err))
		return nil, err
	}

	if len(failedTxLogs) == 0 {
		m.logger.Debug("No refunds found to publish")
		return nil, nil
	}

	refundRequests := make([]ProcessRefundCommand, 0, len(failedTxLogs))
	for _, txLog := range failedTxLogs {
		refundRequest := ProcessRefundCommand{
			TxLogID:    txLog.ID,
			MessageID:  txLog.MessageID,
			FromMSISDN: txLog.FromMSISDN,
			Amount:     txLog.Amount,
		}

		refundRequests = append(refundRequests, refundRequest)
	}

	return refundRequests, nil
}

func (m *messageQueue) MarkRefundAsQueued(ctx context.Context, txLogID int64) error {
	publishedAt := time.Now()
	txLog := model.TxLog{
		ID:          txLogID,
		Published:   true,
		PublishedAt: &publishedAt,
	}

	if err := m.txLog.Update(&txLog); err != nil {
		m.logger.Error("Failed to mark refund tx as published",
			zap.Error(err), zap.Int64("txLogID", txLogID))
		return err
	}

	m.logger.Debug("Successfully marked refund tx as published", zap.Int64("txLogID", txLogID))

	return nil
}
