package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/mocks"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestPayment_Charge(t *testing.T) {
	logger := zap.NewNop()

	cmd := service.ChargePaymentCommand{
		UserID:         "user123",
		Amount:         100,
		IdempotencyKey: "unique-key-123",
	}

	cfg := &config.Config{PaymentGateway: paymentgateway.Config{MaxRetries: 3}}

	t.Run("Successful charge on first attempt", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{
			Code:    "success",
			Message: "Balance updated successfully",
		}

		mockPayment.On("Charge", context.Background(),
			mock.MatchedBy(func(req paymentgateway.UpdateUserBalanceRequest) bool {
				return req.UserID == expectedRequest.UserID &&
					req.Amount == expectedRequest.Amount &&
					req.IdempotencyKey == expectedRequest.IdempotencyKey
			})).Return(expectedResponse, nil)

		err := svc.Charge(context.Background(), cmd)

		assert.NoError(t, err)
		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
	})

	t.Run("Failed charge user not found no retries", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Charge", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrUserNotFound)

		err := svc.Charge(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, constants.ErrCodeUserNotFound, serviceErr.Code)
		assert.Equal(t, paymentgateway.ErrUserNotFound, serviceErr.Cause)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
	})

	t.Run("Failed charge insufficient balance no retries", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Charge", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrInsufficientBalance).Once()

		err := svc.Charge(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, constants.ErrCodeInsufficientBalance, serviceErr.Code)
		assert.Equal(t, paymentgateway.ErrInsufficientBalance, serviceErr.Cause)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 1)
	})

	t.Run("Failed charge timeout error retries until max attempts", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Charge", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrTimeout).Times(3)

		err := svc.Charge(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodeChargeTimeout, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 3)
	})

	t.Run("Failed charge server error retries until max attempts", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Charge", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrServerError).Times(3)

		err := svc.Charge(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodePaymentServiceError, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 3)

	})

	t.Run("success after retries", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{
			Code:    "success",
			Message: "Balance updated successfully",
		}

		mockPayment.On("Charge", context.Background(), expectedRequest).Return(paymentgateway.Response{},
			paymentgateway.ErrTimeout).Twice()
		mockPayment.On("Charge", context.Background(), expectedRequest).Return(expectedResponse, nil).Once()

		err := svc.Charge(context.Background(), cmd)

		assert.NoError(t, err)
		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Charge", 3)
	})
}

func TestPayment_Refund(t *testing.T) {
	logger := zap.NewNop()

	cmd := service.RefundPaymentCommand{
		UserID:         "user123",
		Amount:         100,
		IdempotencyKey: "unique-key-123",
	}

	cfg := &config.Config{PaymentGateway: paymentgateway.Config{MaxRetries: 3}}

	t.Run("Successful refund on first attempt", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{
			Code:    "success",
			Message: "Balance refunded successfully",
		}

		mockPayment.On("Refund", context.Background(),
			mock.MatchedBy(func(req paymentgateway.UpdateUserBalanceRequest) bool {
				return req.UserID == expectedRequest.UserID &&
					req.Amount == expectedRequest.Amount &&
					req.IdempotencyKey == expectedRequest.IdempotencyKey
			})).Return(expectedResponse, nil)

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)
		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Refund", 1)
	})

	t.Run("Failed refund user not found no retries", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Refund", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrUserNotFound).Once()

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, constants.ErrCodeUserNotFound, serviceErr.Code)
		assert.Equal(t, paymentgateway.ErrUserNotFound, serviceErr.Cause)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Refund", 1)
	})

	t.Run("Failed refund timeout error retries until max attempts", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Refund", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrTimeout).Times(3)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodeRefundTimeout, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Refund", 3)
	})

	t.Run("Failed refund server error retries until max attempts", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{}

		mockPayment.On("Refund", context.Background(),
			expectedRequest).Return(expectedResponse, paymentgateway.ErrServerError).Times(3)

		err := svc.Refund(context.Background(), cmd)

		assert.Error(t, err)

		var serviceErr service.Error
		assert.True(t, errors.As(err, &serviceErr))
		assert.Equal(t, service.ErrCodePaymentServiceError, serviceErr.Code)

		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Refund", 3)
	})

	t.Run("Success after retries", func(t *testing.T) {
		mockPayment := &mocks.PaymentGateway{}
		svc := service.NewPaymentService(mockPayment, cfg, logger)

		expectedRequest := paymentgateway.UpdateUserBalanceRequest{
			UserID:         cmd.UserID,
			Amount:         cmd.Amount,
			IdempotencyKey: cmd.IdempotencyKey,
		}

		expectedResponse := paymentgateway.Response{
			Code:    "success",
			Message: "Balance refunded successfully",
		}

		mockPayment.On("Refund", context.Background(), expectedRequest).Return(paymentgateway.Response{},
			paymentgateway.ErrTimeout).Twice()
		mockPayment.On("Refund", context.Background(), expectedRequest).Return(expectedResponse, nil).Once()

		err := svc.Refund(context.Background(), cmd)

		assert.NoError(t, err)
		mockPayment.AssertExpectations(t)
		mockPayment.AssertNumberOfCalls(t, "Refund", 3)
	})
}
