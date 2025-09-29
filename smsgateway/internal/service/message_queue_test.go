package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/mocks"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestMessageQueue_FindMessagesToQueue(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns messages successfully", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		txLogs := []model.TxLog{
			{
				ID:         1,
				MessageID:  101,
				FromMSISDN: "1234567890",
				Amount:     1,
				State:      model.TxLogStateCreated,
				Published:  false,
				Message: model.Message{
					ID:         101,
					ToMSISDN:   "0987654321",
					Text:       "Hello World",
					FromMSISDN: "1234567890",
				},
			},
			{
				ID:         2,
				MessageID:  102,
				FromMSISDN: "1111111111",
				Amount:     1,
				State:      model.TxLogStateCreated,
				Published:  false,
				Message: model.Message{
					ID:         102,
					ToMSISDN:   "2222222222",
					Text:       "Test Message",
					FromMSISDN: "1111111111",
				},
			},
		}

		mockTxLogRepo.On("FindUnpublishedCreated", 100).Return(txLogs, nil)

		messages, err := svc.FindMessagesToQueue(context.Background(), 100)

		assert.NoError(t, err)
		assert.Len(t, messages, 2)

		assert.Equal(t, int64(101), messages[0].MessageID)
		assert.Equal(t, "1234567890", messages[0].FromMSISDN)
		assert.Equal(t, "0987654321", messages[0].ToMSISDN)
		assert.Equal(t, "Hello World", messages[0].Text)

		assert.Equal(t, int64(102), messages[1].MessageID)
		assert.Equal(t, "1111111111", messages[1].FromMSISDN)
		assert.Equal(t, "2222222222", messages[1].ToMSISDN)
		assert.Equal(t, "Test Message", messages[1].Text)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns empty slice when no messages found", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("FindUnpublishedCreated", 100).Return([]model.TxLog{}, nil)

		messages, err := svc.FindMessagesToQueue(context.Background(), 100)

		assert.NoError(t, err)
		assert.Nil(t, messages)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		dbError := errors.New("database connection failed")
		mockTxLogRepo.On("FindUnpublishedCreated", 100).Return([]model.TxLog{}, dbError)

		messages, err := svc.FindMessagesToQueue(context.Background(), 100)

		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Equal(t, dbError, err)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("respects batch size limit", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("FindUnpublishedCreated", 50).Return([]model.TxLog{}, nil)

		_, err := svc.FindMessagesToQueue(context.Background(), 50)

		assert.NoError(t, err)
		mockTxLogRepo.AssertExpectations(t)
		mockTxLogRepo.AssertCalled(t, "FindUnpublishedCreated", 50)
	})
}

func TestMessageQueue_MarkMessageAsQueued(t *testing.T) {
	logger := zap.NewNop()

	t.Run("marks message as queued successfully", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("UpdateByMessageID", context.Background(),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					txLog.State == model.TxLogStatePending &&
					txLog.Published == true &&
					txLog.PublishedAt != nil
			})).Return(nil)

		err := svc.MarkMessageAsQueued(context.Background(), 123)

		assert.NoError(t, err)
		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository update fails", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		dbError := errors.New("database update failed")
		mockTxLogRepo.On("UpdateByMessageID", context.Background(),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123
			})).Return(dbError)

		err := svc.MarkMessageAsQueued(context.Background(), 123)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)

		mockTxLogRepo.AssertExpectations(t)
	})
}

func TestMessageQueue_FindRefundsToQueue(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns refunds successfully", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		txLogs := []model.TxLog{
			{
				ID:         1,
				MessageID:  101,
				FromMSISDN: "1234567890",
				Amount:     1,
				State:      model.TxLogStateFailed,
				Published:  false,
			},
			{
				ID:         2,
				MessageID:  102,
				FromMSISDN: "1111111111",
				Amount:     1,
				State:      model.TxLogStateFailed,
				Published:  false,
			},
		}

		mockTxLogRepo.On("FindUnpublishedFailed", 100).Return(txLogs, nil)

		refunds, err := svc.FindRefundsToQueue(context.Background(), 100)

		assert.NoError(t, err)
		assert.Len(t, refunds, 2)

		assert.Equal(t, int64(1), refunds[0].TxLogID)
		assert.Equal(t, int64(101), refunds[0].MessageID)
		assert.Equal(t, "1234567890", refunds[0].FromMSISDN)
		assert.Equal(t, 1, refunds[0].Amount)

		assert.Equal(t, int64(2), refunds[1].TxLogID)
		assert.Equal(t, int64(102), refunds[1].MessageID)
		assert.Equal(t, "1111111111", refunds[1].FromMSISDN)
		assert.Equal(t, 1, refunds[1].Amount)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns empty slice when no refunds found", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("FindUnpublishedFailed", 100).Return([]model.TxLog{}, nil)

		refunds, err := svc.FindRefundsToQueue(context.Background(), 100)

		assert.NoError(t, err)
		assert.Nil(t, refunds)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		dbError := errors.New("database connection failed")
		mockTxLogRepo.On("FindUnpublishedFailed", 100).Return([]model.TxLog{}, dbError)

		refunds, err := svc.FindRefundsToQueue(context.Background(), 100)

		assert.Error(t, err)
		assert.Nil(t, refunds)
		assert.Equal(t, dbError, err)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("respects batch size limit", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("FindUnpublishedFailed", 50).Return([]model.TxLog{}, nil)

		_, err := svc.FindRefundsToQueue(context.Background(), 50)

		assert.NoError(t, err)
		mockTxLogRepo.AssertExpectations(t)
		mockTxLogRepo.AssertCalled(t, "FindUnpublishedFailed", 50)
	})
}

func TestMessageQueue_MarkRefundAsQueued(t *testing.T) {
	logger := zap.NewNop()

	t.Run("marks refund as queued successfully", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		mockTxLogRepo.On("Update",
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.ID == 123 &&
					txLog.Published == true &&
					txLog.PublishedAt != nil
			})).Return(nil)

		err := svc.MarkRefundAsQueued(context.Background(), 123)

		assert.NoError(t, err)
		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository update fails", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		dbError := errors.New("database update failed")
		mockTxLogRepo.On("Update",
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.ID == 123
			})).Return(dbError)

		err := svc.MarkRefundAsQueued(context.Background(), 123)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)

		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("sets published_at timestamp", func(t *testing.T) {
		mockTxLogRepo := &mocks.TxLogRepository{}

		svc := service.NewMessageQueueService(mockTxLogRepo, logger)

		before := time.Now()

		mockTxLogRepo.On("Update",
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				if txLog.PublishedAt == nil {
					return false
				}
				after := time.Now()
				return !txLog.PublishedAt.Before(before) && !txLog.PublishedAt.After(after)
			})).Return(nil)

		err := svc.MarkRefundAsQueued(context.Background(), 123)

		assert.NoError(t, err)
		mockTxLogRepo.AssertExpectations(t)
	})
}
