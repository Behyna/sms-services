package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"go.uber.org/zap"
)

type RefundService interface {
	Refund(ctx context.Context, request ProcessRefundCommand) error
}

type Refund struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository
	txManager   repository.TxManager
	payment     PaymentService
	logger      *zap.Logger
}

func NewRefundService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	txManager repository.TxManager, payment PaymentService, logger *zap.Logger) RefundService {
	return &Refund{messageRepo: messageRepo, txLogRepo: txLogRepo, txManager: txManager,
		payment: payment, logger: logger}
}

func (r *Refund) Refund(ctx context.Context, cmd ProcessRefundCommand) error {
	r.logger.Info("Processing refund",
		zap.Int64("txLogID", cmd.TxLogID),
		zap.Int64("messageID", cmd.MessageID),
		zap.String("fromMSISDN", cmd.FromMSISDN),
		zap.Int("amount", cmd.Amount))

	_, err := r.getRefundableTransaction(ctx, cmd.TxLogID)
	if err != nil {
		r.logger.Debug("TX not processable",
			zap.Int64("txLogID", cmd.TxLogID),
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))

		if errors.Is(err, ErrDatabase) {
			return mq.Temporary(err)
		}

		return nil
	}

	pgRequest := RefundPaymentCommand{
		UserID:         cmd.FromMSISDN,
		Amount:         int64(cmd.Amount),
		IdempotencyKey: fmt.Sprintf("refund-%d", cmd.MessageID),
	}

	err = r.payment.Refund(ctx, pgRequest)
	if err == nil {
		if err := r.updateMessageToRefunded(ctx, cmd.MessageID); err != nil {
			r.logger.Error("Payment refunded but database update failed",
				zap.Int64("txLogID", cmd.TxLogID),
				zap.Error(err))
			return mq.Temporary(err)
		}

		r.logger.Info("Refund completed successfully",
			zap.Int64("txLogID", cmd.TxLogID))
		return nil
	}

	r.logger.Warn("Payment gateway refund failed",
		zap.Int64("txLogID", cmd.TxLogID),
		zap.Error(err))

	var serviceErr Error
	if errors.As(err, &serviceErr) && (serviceErr.Code == constants.ErrCodeUserNotFound) {
		r.logger.Info("Permanent refund failure",
			zap.Int64("txLogID", cmd.TxLogID),
			zap.String("reason", serviceErr.Code))
		return nil
	}

	r.logger.Debug("Temporary refund failure, will retry",
		zap.Int64("txLogID", cmd.TxLogID),
		zap.String("reason", serviceErr.Code))

	return mq.Temporary(err)
}

func (r *Refund) getRefundableTransaction(ctx context.Context, txLogID int64) (*model.TxLog, error) {
	txLog, err := r.txLogRepo.GetByID(txLogID)
	if err != nil {
		if errors.Is(err, repository.ErrTxLogNotFound) {
			return nil, ErrTxLogNotFound
		}

		return nil, ErrDatabase
	}

	switch txLog.State {
	case model.TxLogStateCreated, model.TxLogStatePending, model.TxLogStateSuccess:
		r.logger.Warn("Transaction not in refundable state",
			zap.Int64("txLogID", txLogID),
			zap.String("state", txLog.State))
		return nil, ErrTxInvalidState

	case model.TxLogStateFailed:
		return txLog, nil

	case model.TxLogStateRefunded:
		r.logger.Info("Transaction already refunded", zap.Int64("txLogID", txLogID))
		return nil, ErrRefundAlreadyProcessed

	default:
		r.logger.Error("Unknown transaction state",
			zap.String("state", txLog.State),
			zap.Int64("txLogID", txLogID))
		return nil, ErrUnknownTxState
	}
}

func (r *Refund) updateMessageToRefunded(ctx context.Context, messageID int64) error {
	msg := model.Message{
		ID:        messageID,
		Status:    model.MessageStatusRefunded,
		UpdatedAt: time.Now(),
	}

	txLog := model.TxLog{
		MessageID: messageID,
		State:     model.TxLogStateRefunded,
		UpdatedAt: time.Now(),
	}

	err := r.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := r.messageRepo.Update(ctx, &msg); err != nil {
			r.logger.Error("Failed to update message status to REFUNDED",
				zap.Int64("messageID", messageID),
				zap.Error(err))
			return err
		}

		if err := r.txLogRepo.UpdateByMessageID(ctx, &txLog); err != nil {
			r.logger.Error("Failed to update transaction log to REFUNDED",
				zap.Int64("messageID", messageID),
				zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	r.logger.Info("Successfully updated to refunded state",
		zap.Int64("messageID", messageID))

	return nil
}
