package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/mocks"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestMessage_CreateMessage(t *testing.T) {
	logger := zap.NewNop()

	cmd := service.CreateMessageCommand{
		ClientMessageID: "test-msg-123",
		FromMSISDN:      "1234567890",
		ToMSISDN:        "0987654321",
		Text:            "Hello World",
	}

	t.Run("creates message successfully", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		expectedChargeRequest := service.ChargePaymentCommand{
			UserID:         cmd.FromMSISDN,
			Amount:         1,
			IdempotencyKey: "charge-" + cmd.FromMSISDN + "-" + cmd.ClientMessageID,
		}

		mockPayment.On("Charge", context.Background(),
			mock.MatchedBy(func(req service.ChargePaymentCommand) bool {
				return req.UserID == expectedChargeRequest.UserID &&
					req.Amount == expectedChargeRequest.Amount &&
					req.IdempotencyKey == expectedChargeRequest.IdempotencyKey
			})).Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(msg *model.Message) bool {
				return msg.ClientMessageID == cmd.ClientMessageID &&
					msg.FromMSISDN == cmd.FromMSISDN &&
					msg.ToMSISDN == cmd.ToMSISDN &&
					msg.Text == cmd.Text &&
					msg.Status == model.MessageStatusCreated &&
					msg.AttemptCount == 0
			})).Run(func(args mock.Arguments) {
			msg := args.Get(1).(*model.Message)
			msg.ID = 123
		}).Return(nil)

		mockTxLogRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(txLog *model.TxLog) bool {
				return txLog.MessageID == 123 &&
					txLog.FromMSISDN == cmd.FromMSISDN &&
					txLog.Amount == 1 &&
					txLog.State == model.TxLogStateCreated &&
					txLog.Published == false &&
					txLog.PublishedAt == nil
			})).Return(nil)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.NoError(t, err)
		assert.Equal(t, int64(123), resp.MessageID)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("returns error when payment charge fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		chargeError := service.NewServiceError(
			constants.ErrCodeInsufficientBalance,
			errors.New("insufficient balance"),
		)

		mockPayment.On("Charge", context.Background(),
			mock.AnythingOfType("service.ChargePaymentCommand")).Return(chargeError)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, service.CreateMessageResponse{}, resp)
		assert.Equal(t, chargeError, err)

		mockPayment.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "Create")
		mockTxLogRepo.AssertNotCalled(t, "Create")
	})

	t.Run("refunds payment when message creation fails due to duplicate", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		expectedRefundRequest := service.RefundPaymentCommand{
			UserID:         cmd.FromMSISDN,
			Amount:         1,
			IdempotencyKey: "refund-" + cmd.FromMSISDN + "-" + cmd.ClientMessageID,
		}

		mockPayment.On("Charge", context.Background(), mock.AnythingOfType("service.ChargePaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(repository.ErrMessageDuplicate)

		mockPayment.On("Refund", context.Background(),
			mock.MatchedBy(func(req service.RefundPaymentCommand) bool {
				return req.UserID == expectedRefundRequest.UserID &&
					req.Amount == expectedRefundRequest.Amount &&
					req.IdempotencyKey == expectedRefundRequest.IdempotencyKey
			})).Return(nil)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, service.CreateMessageResponse{}, resp)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, constants.ErrCodeDuplicateMessage, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
		mockPayment.AssertNumberOfCalls(t, "Refund", 1)
	})

	t.Run("refunds payment when message creation fails due to database error", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		dbError := errors.New("database connection failed")

		mockPayment.On("Charge", context.Background(), mock.AnythingOfType("service.ChargePaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(dbError)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(nil)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, service.CreateMessageResponse{}, resp)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodeDatabase, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
		mockPayment.AssertNumberOfCalls(t, "Refund", 1)
	})

	t.Run("refunds payment when txlog creation fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		dbError := errors.New("txlog insert failed")

		mockPayment.On("Charge", context.Background(), mock.AnythingOfType("service.ChargePaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Run(func(args mock.Arguments) {
			msg := args.Get(1).(*model.Message)
			msg.ID = 123
		}).Return(nil)

		mockTxLogRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.TxLog")).Return(dbError)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(nil)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, service.CreateMessageResponse{}, resp)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodeDatabase, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
		mockPayment.AssertNumberOfCalls(t, "Refund", 1)
	})

	t.Run("returns message creation error when both message creation and refund fail", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		dbError := errors.New("database error")
		refundError := service.NewServiceError(
			service.ErrCodeRefundTimeout,
			errors.New("refund timeout"),
		)

		mockPayment.On("Charge", context.Background(), mock.AnythingOfType("service.ChargePaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(dbError)

		mockPayment.On("Refund", context.Background(), mock.AnythingOfType("service.RefundPaymentCommand")).
			Return(refundError)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, service.CreateMessageResponse{}, resp)

		// Should return original message creation error, not refund error
		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodeDatabase, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("creates message with correct idempotency keys", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		var capturedChargeKey string

		mockPayment.On("Charge", context.Background(),
			mock.AnythingOfType("service.ChargePaymentCommand")).Run(func(args mock.Arguments) {
			chargeCmd := args.Get(1).(service.ChargePaymentCommand)
			capturedChargeKey = chargeCmd.IdempotencyKey
		}).Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Run(func(args mock.Arguments) {
			msg := args.Get(1).(*model.Message)
			msg.ID = 123
		}).Return(nil)

		mockTxLogRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.TxLog")).Return(nil)

		resp, err := svc.CreateMessage(context.Background(), cmd)

		assert.NoError(t, err)
		assert.Equal(t, int64(123), resp.MessageID)
		assert.Equal(t, "charge-1234567890-test-msg-123", capturedChargeKey)

		mockPayment.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
		mockTxLogRepo.AssertExpectations(t)
	})

	t.Run("uses correct refund idempotency key", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		var capturedRefundKey string

		mockPayment.On("Charge", context.Background(), mock.AnythingOfType("service.ChargePaymentCommand")).
			Return(nil)

		mockTxManager.On("WithTx", context.Background(),
			mock.AnythingOfType("func(context.Context) error")).Return(nil)

		mockMessageRepo.On("Create", mock.AnythingOfType("*context.valueCtx"),
			mock.AnythingOfType("*model.Message")).Return(repository.ErrMessageDuplicate)

		mockPayment.On("Refund", context.Background(),
			mock.AnythingOfType("service.RefundPaymentCommand")).Run(func(args mock.Arguments) {
			refundCmd := args.Get(1).(service.RefundPaymentCommand)
			capturedRefundKey = refundCmd.IdempotencyKey
		}).Return(nil)

		_, err := svc.CreateMessage(context.Background(), cmd)

		assert.Error(t, err)
		assert.Equal(t, "refund-1234567890-test-msg-123", capturedRefundKey)

		mockPayment.AssertExpectations(t)
		mockMessageRepo.AssertExpectations(t)
	})
}

func TestMessage_GetMessagesByUserID(t *testing.T) {
	logger := zap.NewNop()

	query := service.GetMessagesQuery{
		UserID: "1234567890",
		Limit:  10,
		Offset: 0,
	}

	t.Run("returns messages successfully", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		now := time.Now()
		messages := []model.Message{
			{
				ID:              123,
				ClientMessageID: "msg-1",
				FromMSISDN:      "1234567890",
				ToMSISDN:        "0987654321",
				Text:            "Hello",
				Status:          model.MessageStatusSubmitted,
				CreatedAt:       now,
			},
			{
				ID:              124,
				ClientMessageID: "msg-2",
				FromMSISDN:      "1234567890",
				ToMSISDN:        "1111111111",
				Text:            "World",
				Status:          model.MessageStatusCreated,
				CreatedAt:       now.Add(-1 * time.Hour),
			},
		}

		mockMessageRepo.On("GetByUserID", query.UserID, query.Limit, query.Offset).
			Return(messages, nil)

		mockMessageRepo.On("CountByUserID", query.UserID).Return(25, nil)

		resp, err := svc.GetMessagesByUserID(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, resp.Messages, 2)
		assert.Equal(t, int64(25), resp.Total)

		assert.Equal(t, "msg-1", resp.Messages[0].MessageID)
		assert.Equal(t, "1234567890", resp.Messages[0].From)
		assert.Equal(t, "0987654321", resp.Messages[0].To)
		assert.Equal(t, "Hello", resp.Messages[0].Text)
		assert.Equal(t, string(model.MessageStatusSubmitted), resp.Messages[0].Status)

		assert.Equal(t, "msg-2", resp.Messages[1].MessageID)
		assert.Equal(t, "1234567890", resp.Messages[1].From)
		assert.Equal(t, "1111111111", resp.Messages[1].To)
		assert.Equal(t, "World", resp.Messages[1].Text)
		assert.Equal(t, string(model.MessageStatusCreated), resp.Messages[1].Status)

		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no messages found", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		mockMessageRepo.On("GetByUserID", query.UserID, query.Limit, query.Offset).
			Return([]model.Message{}, nil)

		resp, err := svc.GetMessagesByUserID(context.Background(), query)

		assert.Error(t, err)
		assert.Equal(t, service.GetMessagesResponse{}, resp)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, constants.ErrCodeUserNotFound, serviceErr.Code)

		mockMessageRepo.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "CountByUserID")
	})

	t.Run("returns error when GetByUserID fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		dbError := errors.New("database connection failed")

		mockMessageRepo.On("GetByUserID", query.UserID, query.Limit, query.Offset).
			Return([]model.Message{}, dbError)

		resp, err := svc.GetMessagesByUserID(context.Background(), query)

		assert.Error(t, err)
		assert.Equal(t, service.GetMessagesResponse{}, resp)
		assert.Equal(t, service.ErrDatabase, err)

		mockMessageRepo.AssertExpectations(t)
		mockMessageRepo.AssertNotCalled(t, "CountByUserID")
	})

	t.Run("returns error when CountByUserID fails", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		messages := []model.Message{
			{
				ID:              123,
				ClientMessageID: "msg-1",
				FromMSISDN:      "1234567890",
				ToMSISDN:        "0987654321",
				Text:            "Hello",
				Status:          model.MessageStatusSubmitted,
				CreatedAt:       time.Now(),
			},
		}

		dbError := errors.New("count query failed")

		mockMessageRepo.On("GetByUserID", query.UserID, query.Limit, query.Offset).
			Return(messages, nil)

		mockMessageRepo.On("CountByUserID", query.UserID).Return(0, dbError)

		resp, err := svc.GetMessagesByUserID(context.Background(), query)

		assert.Error(t, err)
		assert.Equal(t, service.GetMessagesResponse{}, resp)
		assert.Equal(t, service.ErrDatabase, err)

		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("formats timestamps correctly", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		createdAt, _ := time.Parse(time.RFC3339, "2023-06-15T10:30:00Z")
		messages := []model.Message{
			{
				ID:              123,
				ClientMessageID: "msg-1",
				FromMSISDN:      "1234567890",
				ToMSISDN:        "0987654321",
				Text:            "Hello",
				Status:          model.MessageStatusSubmitted,
				CreatedAt:       createdAt,
			},
		}

		mockMessageRepo.On("GetByUserID", query.UserID, query.Limit, query.Offset).
			Return(messages, nil)

		mockMessageRepo.On("CountByUserID", query.UserID).Return(1, nil)

		resp, err := svc.GetMessagesByUserID(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, resp.Messages, 1)
		assert.Equal(t, "2023-06-15T10:30:00Z", resp.Messages[0].CreatedAt)

		mockMessageRepo.AssertExpectations(t)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		mockMessageRepo := &mocks.MessageRepository{}
		mockTxLogRepo := &mocks.TxLogRepository{}
		mockTxManager := &mocks.TxManager{}
		mockPayment := &mocks.PaymentService{}

		svc := service.NewMessageService(mockMessageRepo, mockTxLogRepo, mockTxManager, mockPayment, logger)

		customQuery := service.GetMessagesQuery{
			UserID: "1234567890",
			Limit:  5,
			Offset: 10,
		}

		messages := []model.Message{
			{
				ID:              123,
				ClientMessageID: "msg-1",
				FromMSISDN:      "1234567890",
				ToMSISDN:        "0987654321",
				Text:            "Hello",
				Status:          model.MessageStatusSubmitted,
				CreatedAt:       time.Now(),
			},
		}

		mockMessageRepo.On("GetByUserID", customQuery.UserID, customQuery.Limit, customQuery.Offset).
			Return(messages, nil)

		mockMessageRepo.On("CountByUserID", customQuery.UserID).Return(50, nil)

		resp, err := svc.GetMessagesByUserID(context.Background(), customQuery)

		assert.NoError(t, err)
		assert.Equal(t, int64(50), resp.Total)

		mockMessageRepo.AssertExpectations(t)
		mockMessageRepo.AssertCalled(t, "GetByUserID", customQuery.UserID, 5, 10)
	})
}
