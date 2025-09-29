package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"go.uber.org/zap"
)

const maxRetries = 3

type MessageWorkflowService interface {
	CreateMessage(ctx context.Context, cmd CreateMessageCommand) (CreateMessageResponse, error)
	SendMessage(ctx context.Context, cmd SendMessageCommand) error
}

type messageWorkflow struct {
	message  MessageService
	provider ProviderService
	payment  PaymentService
	logger   *zap.Logger
}

func NewMessageWorkflowService(message MessageService, provider ProviderService, payment PaymentService,
	logger *zap.Logger) MessageWorkflowService {
	return &messageWorkflow{message: message, provider: provider, payment: payment, logger: logger}
}

func (m *messageWorkflow) CreateMessage(ctx context.Context, cmd CreateMessageCommand) (
	CreateMessageResponse, error) {

	idempotencyKey := fmt.Sprintf("charge-%s-%s", cmd.FromMSISDN, cmd.ClientMessageID)
	request := ChargePaymentCommand{UserID: cmd.FromMSISDN, Amount: 1, IdempotencyKey: idempotencyKey}

	err := m.payment.Charge(ctx, request)
	if err != nil {
		m.logger.Debug("Message creation aborted due to payment failure",
			zap.String("clientMessageID", cmd.ClientMessageID))
		return CreateMessageResponse{}, err
	}

	resp, err := m.message.CreateMessageTx(ctx, cmd)
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

func (m *messageWorkflow) SendMessage(ctx context.Context, cmd SendMessageCommand) error {
	msg, err := m.message.GetMessageForProcessing(ctx, cmd.MessageID)
	if err != nil {
		m.logger.Debug("Message not processable",
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))

		if errors.Is(err, ErrDatabase) {
			return mq.Temporary(err)
		}

		return nil
	}

	attemptCount := msg.AttemptCount
	if msg.Status != model.MessageStatusSending {
		attemptCount += 1
	}

	if attemptCount > maxRetries {
		m.logger.Warn("Message exceeded max retries",
			zap.Int64("messageID", cmd.MessageID),
			zap.Int("attempts", attemptCount))

		updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: "exceeded max retries"}
		if err := m.message.UpdateMessageToPermanentFailure(ctx, updateFailedCmd); err != nil {
			return mq.Temporary(err)
		}

		return nil
	}

	updateSendingCmd := UpdateMessageToSendingCommand{MessageID: cmd.MessageID, AttemptCount: attemptCount}
	if err := m.message.UpdateMessageToSending(ctx, updateSendingCmd); err != nil {
		m.logger.Debug("Failed to update message to SENDING status",
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))
		return mq.Temporary(err)
	}

	m.logger.Debug("Attempting to send SMS",
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount),
		zap.Int("maxRetries", maxRetries),
		zap.String("to", cmd.ToMSISDN),
		zap.String("from", cmd.FromMSISDN))

	response, lastErr := m.provider.SendWithRetry(ctx, cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text)
	if lastErr == nil {
		m.logger.Info("SMS sent successfully",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("providerMessageID", response.MessageID),
			zap.String("provider", response.Provider),
			zap.Int("attempt", attemptCount))

		updateCmd := UpdateMessageSuccessCommand{MessageID: cmd.MessageID, ProviderMsgID: response.MessageID, Provider: response.Provider}
		return m.message.UpdateMessageSucceed(ctx, updateCmd)
	}

	m.logger.Debug("SMS provider call failed",
		zap.Error(lastErr),
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount))

	if lastErr.Error() == smsprovider.ErrorCodeInvalidNumber {
		m.logger.Warn("Permanent failure due to invalid number, marking for refund",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("reason", "invalid_number"))

		updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: lastErr.Error()}
		if err := m.message.UpdateMessageToPermanentFailure(ctx, updateFailedCmd); err != nil {
			return mq.Temporary(err)
		}

		return nil
	}

	m.logger.Debug("Temporary failure, will retry",
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount),
		zap.Int("remainingRetries", maxRetries-attemptCount),
		zap.Error(lastErr))

	updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: lastErr.Error()}
	if err := m.message.UpdateMessageToTemporaryFailure(ctx, updateFailedCmd); err != nil {
		return mq.Temporary(err)
	}

	return mq.Temporary(lastErr)
}
