package service

import (
	"context"
	"errors"

	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"go.uber.org/zap"
)

type PaymentService interface {
	Charge(ctx context.Context, cmd ChargePaymentCommand) error
	Refund(ctx context.Context, cmd RefundPaymentCommand) error
}

type Payment struct {
	paymentGateway paymentgateway.PaymentGateway
	maxRetry       int
	logger         *zap.Logger
}

func NewPaymentService(paymentGateway paymentgateway.PaymentGateway, config *config.Config, logger *zap.Logger) PaymentService {
	return &Payment{paymentGateway: paymentGateway, maxRetry: config.PaymentGateway.MaxRetries, logger: logger}
}

func (p *Payment) Charge(ctx context.Context, cmd ChargePaymentCommand) error {
	request := paymentgateway.UpdateUserBalanceRequest{
		UserID:         cmd.UserID,
		Amount:         cmd.Amount,
		IdempotencyKey: cmd.IdempotencyKey,
	}

	var lastErr error
	for attempt := 1; attempt <= p.maxRetry; attempt++ {
		resp, err := p.paymentGateway.Charge(ctx, request)
		if err == nil {
			p.logger.Info("User charged successfully",
				zap.String("userID", cmd.UserID),
				zap.Int("attempt", attempt),
				zap.String("idempotencyKey", cmd.IdempotencyKey),
				zap.Int64("transactionID", resp.Result.TransactionID))

			return nil
		}

		if errors.Is(err, paymentgateway.ErrUserNotFound) {
			p.logger.Warn("Non-retryable error encountered",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.String("userID", cmd.UserID))
			return NewServiceError(constants.ErrCodeUserNotFound, err)
		}

		if errors.Is(err, paymentgateway.ErrInsufficientBalance) {
			p.logger.Warn("Non-retryable error encountered",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.String("userID", cmd.UserID))
			return NewServiceError(constants.ErrCodeInsufficientBalance, err)
		}

		lastErr = err
	}

	if errors.Is(lastErr, paymentgateway.ErrTimeout) {
		p.logger.Error("Charge attempts timed out",
			zap.Error(lastErr),
			zap.Int("maxRetries", p.maxRetry),
			zap.String("userID", cmd.UserID))
		return NewServiceError(ErrCodeChargeTimeout, lastErr)
	}

	p.logger.Error("Payment service unavailable after all retries",
		zap.Error(lastErr),
		zap.Int("maxRetries", p.maxRetry),
		zap.String("userID", cmd.UserID))

	return NewServiceError(ErrCodePaymentServiceError, lastErr)
}

func (p *Payment) Refund(ctx context.Context, cmd RefundPaymentCommand) error {
	request := paymentgateway.UpdateUserBalanceRequest{
		UserID:         cmd.UserID,
		Amount:         cmd.Amount,
		IdempotencyKey: cmd.IdempotencyKey,
	}

	var lastErr error
	for attempt := 1; attempt <= p.maxRetry; attempt++ {
		resp, err := p.paymentGateway.Refund(ctx, request)
		if err == nil {
			p.logger.Info("User refunded successfully",
				zap.String("userID", cmd.UserID),
				zap.Int("attempt", attempt),
				zap.Int64("transactionID", resp.Result.TransactionID),
				zap.String("idempotencyKey", cmd.IdempotencyKey))

			return nil
		}

		p.logger.Warn("Refund attempt failed",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.String("userID", cmd.UserID))

		if errors.Is(err, paymentgateway.ErrUserNotFound) {
			p.logger.Error("Non-retryable error encountered",
				zap.Error(err),
				zap.String("userID", cmd.UserID))

			return NewServiceError(constants.ErrCodeUserNotFound, err)
		}

		lastErr = err
	}

	if errors.Is(lastErr, paymentgateway.ErrTimeout) {
		p.logger.Error("Refund attempts timed out",
			zap.Error(lastErr),
			zap.Int("maxRetries", p.maxRetry),
			zap.String("userID", cmd.UserID))

		return NewServiceError(ErrCodeRefundTimeout, lastErr)
	}

	p.logger.Error("Payment service unavailable after all retries",
		zap.Error(lastErr),
		zap.Int("maxRetries", p.maxRetry),
		zap.String("userID", cmd.UserID))

	return NewServiceError(ErrCodePaymentServiceError, lastErr)
}
