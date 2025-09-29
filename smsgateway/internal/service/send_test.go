package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/mocks"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestSend_SendMessage(t *testing.T) {
	logger := zap.NewNop()

	cmd := service.SendMessageCommand{
		MessageID:  123,
		FromMSISDN: "1234567890",
		ToMSISDN:   "0987654321",
		Text:       "Hello World",
	}

	t.Run("send message successfully on first attempt", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerResponse := smsprovider.Response{
			MessageID: "provider-msg-123",
			Provider:  "test-provider",
			Status:    "sent",
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 &&
					msg.Status == model.MessageStatusSending &&
					msg.AttemptCount == 1 &&
					msg.LastAttemptAt != nil
			}), mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(providerResponse, nil)

		mockMessageRepo.On("Update", context.Background(),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 &&
					msg.Status == model.MessageStatusSubmitted &&
					*msg.ProviderMsgID == "provider-msg-123" &&
					*msg.Provider == "test-provider"
			})).Return(nil)

		mockTxLogRepo.On("UpdateByMessageID", context.Background(),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					txLog.State == model.TxLogStateSuccess &&
					txLog.LastError == nil
			})).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("send message successfully after temporary failure", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusFailedTemp,
			AttemptCount: 1,
		}

		providerResponse := smsprovider.Response{
			MessageID: "provider-msg-456",
			Provider:  "test-provider",
			Status:    "sent",
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 && msg.AttemptCount == 2
			}), mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(providerResponse, nil)

		mockMessageRepo.On("Update", context.Background(), mock.AnythingOfType("*model.Message")).Return(nil)
		mockTxLogRepo.On("UpdateByMessageID", context.Background(), mock.AnythingOfType("*model.TxLog")).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("dequeue when message not found", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		mockMessageRepo.On("GetByID", int64(123)).Return((*model.Message)(nil), repository.ErrMessageNotFound)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("requeue when database error occurs during get", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		dbError := errors.New("database connection failed")
		mockMessageRepo.On("GetByID", int64(123)).Return((*model.Message)(nil), dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when message already submitted", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:     123,
			Status: model.MessageStatusSubmitted,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when message permanently failed", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:     123,
			Status: model.MessageStatusFailedPerm,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when message already refunded", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:     123,
			Status: model.MessageStatusRefunded,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when message being processed by another consumer", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		recentTime := time.Now().Add(-2 * time.Minute)
		message := &model.Message{
			ID:            123,
			Status:        model.MessageStatusSending,
			LastAttemptAt: &recentTime,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("process stale message in sending status", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		staleTime := time.Now().Add(-10 * time.Minute)
		message := &model.Message{
			ID:            123,
			Status:        model.MessageStatusSending,
			AttemptCount:  1,
			LastAttemptAt: &staleTime,
		}

		providerResponse := smsprovider.Response{
			MessageID: "provider-msg-123",
			Provider:  "test-provider",
			Status:    "sent",
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 && msg.AttemptCount == 1
			}), mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(providerResponse, nil)

		mockMessageRepo.On("Update", context.Background(), mock.AnythingOfType("*model.Message")).Return(nil)
		mockTxLogRepo.On("UpdateByMessageID", context.Background(), mock.AnythingOfType("*model.TxLog")).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("dequeue when max retries exceeded", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusFailedTemp,
			AttemptCount: 3,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 && msg.Status == model.MessageStatusFailedPerm
			})).Return(nil)

		mockTxLogRepo.On("UpdateForPermFailed", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					txLog.State == model.TxLogStateFailed &&
					txLog.Published == false &&
					*txLog.LastError == "exceeded max retries"
			})).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("requeue when max retries exceeded but update to permanent failure status fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusFailedTemp,
			AttemptCount: 3,
		}

		dbError := errors.New("database update failed")

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("requeue when update to sending fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		dbError := errors.New("database error")

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when during update to sending message being processed by another consumer", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(repository.ErrNoRowsAffected)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockProvider.AssertNotCalled(t, "SendWithRetry")
	})

	t.Run("dequeue when provider returns invalid number (non retryable error)", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerError := errors.New(smsprovider.ErrorCodeInvalidNumber)

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(smsprovider.Response{}, providerError)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 && msg.Status == model.MessageStatusFailedPerm
			})).Return(nil)

		mockTxLogRepo.On("UpdateForPermFailed", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					txLog.State == model.TxLogStateFailed &&
					txLog.Published == false &&
					*txLog.LastError == smsprovider.ErrorCodeInvalidNumber
			})).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("requeue when permanent failure update fails after invalid number", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerError := errors.New(smsprovider.ErrorCodeInvalidNumber)
		dbError := errors.New("database error")

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(smsprovider.Response{}, providerError)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("requeue when provider returns temporary error", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerError := errors.New("provider timeout")

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(smsprovider.Response{}, providerError)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == 123 && msg.Status == model.MessageStatusFailedTemp
			})).Return(nil)

		mockTxLogRepo.On("UpdateByMessageID", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					*txLog.LastError == "provider timeout"
			})).Return(nil)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("requeue when temporary failure update fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerError := errors.New("provider timeout")
		dbError := errors.New("database error")

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(smsprovider.Response{}, providerError)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockMessageRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("continues when message update succeeds but txlog update fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockProvider := &mocks.ProviderService{}

		svc := service.NewSendService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockProvider, logger)

		message := &model.Message{
			ID:           123,
			Status:       model.MessageStatusCreated,
			AttemptCount: 0,
		}

		providerResponse := smsprovider.Response{
			MessageID: "provider-msg-123",
			Provider:  "test-provider",
			Status:    "sent",
		}

		mockMessageRepo.On("GetByID", int64(123)).Return(message, nil)

		mockMessageRepo.On("UpdateForSending", context.Background(),
			mock.AnythingOfType("*model.Message"),
			mock.AnythingOfType("time.Time")).Return(nil)

		mockProvider.On("SendWithRetry", context.Background(), cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text).
			Return(providerResponse, nil)

		mockMessageRepo.On("Update", context.Background(), mock.AnythingOfType("*model.Message")).Return(nil)

		dbError := errors.New("txlog update failed")
		mockTxLogRepo.On("UpdateByMessageID", context.Background(), mock.AnythingOfType("*model.TxLog")).Return(dbError)

		err := svc.SendMessage(context.Background(), cmd)

		assert.NoError(t, err)

		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})
}

func isTemporaryError(err error) bool {
	type temporary interface {
		Temporary() bool
	}
	te, ok := err.(temporary)
	return ok && te.Temporary()
}
