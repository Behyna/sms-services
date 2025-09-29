package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/mocks"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestRefund_Refund(t *testing.T) {
	logger := zap.NewNop()

	cmd := service.ProcessRefundCommand{
		TxLogID:         1,
		MessageID:       123,
		FromMSISDN:      "1234567890",
		Amount:          1,
		ClientMessageID: "abc",
	}

	t.Run("process refund successfully", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		expectedIdempotencyKey := "refund-1234567890-abc"

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		mockPayment.On("Refund", context.Background(),
			mock.MatchedBy(func(req service.RefundPaymentCommand) bool {
				return req.UserID == cmd.FromMSISDN &&
					req.Amount == int64(cmd.Amount) &&
					req.IdempotencyKey == expectedIdempotencyKey
			})).Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ID == cmd.MessageID &&
					msg.Status == model.MessageStatusRefunded
			})).Return(nil)

		mockTxLogRepo.On("UpdateByMessageID", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == cmd.MessageID &&
					txLog.State == model.TxLogStateRefunded
			})).Return(nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("dequeue when transaction not found", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		mockTxLogRepo.On("GetByID", int64(1)).Return((*model.TxLog)(nil), repository.ErrTxLogNotFound)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
		mockMessageRepo.AssertNotCalled(t, "Update")
	})

	t.Run("requeue when database error occurs during get", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		dbError := errors.New("database connection failed")
		mockTxLogRepo.On("GetByID", int64(1)).Return((*model.TxLog)(nil), dbError)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when transaction in created state", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateCreated,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when transaction in pending state", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStatePending,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when transaction in success state", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateSuccess,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when transaction already refunded", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateRefunded,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when transaction has unknown state", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     "UNKNOWN_STATE",
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNotCalled(t, "Refund")
	})

	t.Run("dequeue when payment gateway returns user not found (non retryable error)", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		paymentError := service.NewServiceError(
			constants.ErrCodeUserNotFound,
			errors.New("user not found"),
		)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(paymentError)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "Update")
	})

	t.Run("requeue when payment gateway returns timeout (retryable error)", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		paymentError := service.NewServiceError(
			service.ErrCodeRefundTimeout,
			errors.New("refund timeout"),
		)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(paymentError)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "Update")
	})

	t.Run("requeue when payment gateway returns service error", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		paymentError := service.NewServiceError(
			service.ErrCodePaymentServiceError,
			errors.New("payment service error"),
		)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(paymentError)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "Update")
	})

	t.Run("requeue when message update fails after successful refund", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(nil)

		dbError := errors.New("database update failed")
		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(dbError)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("requeue when txlog update fails after successful refund", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(nil)

		dbError := errors.New("txlog update failed")
		mockTxLogRepo.On("UpdateByMessageID", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.TxLog")).Return(dbError)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)
		assert.True(t, isTemporaryError(err))

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("updates both message and txlog to refund state", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewRefundService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		txLog := &model.TxLog{
			ID:        1,
			MessageID: 123,
			State:     model.TxLogStateFailed,
		}

		mockTxLogRepo.On("GetByID", int64(1)).Return(txLog, nil)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		messageUpdated := false
		mockMessageRepo.On("Update", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				if msg.ID == 123 && msg.Status == model.MessageStatusRefunded {
					messageUpdated = true
					return true
				}
				return false
			})).Return(nil)

		txLogUpdated := false
		mockTxLogRepo.On("UpdateByMessageID", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				if txLog.MessageID == 123 && txLog.State == model.TxLogStateRefunded {
					txLogUpdated = true
					return true
				}
				return false
			})).Return(nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)
		assert.True(t, messageUpdated, "Message should be updated to REFUNDED")
		assert.True(t, txLogUpdated, "TxLog should be updated to REFUNDED")

		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})
}
