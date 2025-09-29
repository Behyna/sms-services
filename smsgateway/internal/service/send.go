package service

import (
	"context"
	"errors"
	"time"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"go.uber.org/zap"
)

const maxRetries = 3

type SendService interface {
	SendMessage(ctx context.Context, cmd SendMessageCommand) error
}

type send struct {
	messageRepo repository.MessageRepository
	txLogRepo   repository.TxLogRepository
	txManager   repository.TxManager
	provider    ProviderService
	logger      *zap.Logger
}

func NewSendService(messageRepo repository.MessageRepository, txLogRepo repository.TxLogRepository,
	txManager repository.TxManager, provider ProviderService, logger *zap.Logger) SendService {
	return &send{messageRepo: messageRepo, txLogRepo: txLogRepo, txManager: txManager, provider: provider, logger: logger}
}

func (s *send) SendMessage(ctx context.Context, cmd SendMessageCommand) error {
	msg, err := s.getMessageForProcessing(cmd.MessageID)
	if err != nil {
		s.logger.Debug("Message not processable",
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
		s.logger.Warn("Message exceeded max retries",
			zap.Int64("messageID", cmd.MessageID),
			zap.Int("attempts", attemptCount))

		updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: "exceeded max retries"}
		if err := s.updateMessageToPermanentFailure(ctx, updateFailedCmd); err != nil {
			return mq.Temporary(err)
		}

		return nil
	}

	updateSendingCmd := UpdateMessageToSendingCommand{MessageID: cmd.MessageID, AttemptCount: attemptCount}
	if err := s.updateMessageToSending(ctx, updateSendingCmd); err != nil {
		if errors.Is(err, ErrMessageBeingProcessed) {
			return nil
		}

		s.logger.Debug("Failed to update message to SENDING status",
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))
		return mq.Temporary(err)
	}

	s.logger.Debug("Attempting to send SMS",
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount),
		zap.Int("maxRetries", maxRetries),
		zap.String("to", cmd.ToMSISDN),
		zap.String("from", cmd.FromMSISDN))

	response, lastErr := s.provider.SendWithRetry(ctx, cmd.FromMSISDN, cmd.ToMSISDN, cmd.Text)
	if lastErr == nil {
		s.logger.Info("SMS sent successfully",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("providerMessageID", response.MessageID),
			zap.String("provider", response.Provider),
			zap.Int("attempt", attemptCount))

		updateCmd := UpdateMessageSuccessCommand{MessageID: cmd.MessageID, ProviderMsgID: response.MessageID, Provider: response.Provider}
		return s.updateMessageSucceed(ctx, updateCmd)
	}

	s.logger.Debug("SMS provider call failed",
		zap.Error(lastErr),
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount))

	if lastErr.Error() == smsprovider.ErrorCodeInvalidNumber {
		s.logger.Warn("Permanent failure due to invalid number, marking for refund",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("reason", "invalid_number"))

		updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: lastErr.Error()}
		if err := s.updateMessageToPermanentFailure(ctx, updateFailedCmd); err != nil {
			return mq.Temporary(err)
		}

		return nil
	}

	s.logger.Debug("Temporary failure, will retry",
		zap.Int64("messageID", cmd.MessageID),
		zap.Int("attempt", attemptCount),
		zap.Int("remainingRetries", maxRetries-attemptCount),
		zap.Error(lastErr))

	updateFailedCmd := UpdateMessageFailureCommand{MessageID: cmd.MessageID, LastError: lastErr.Error()}
	if err := s.updateMessageToTemporaryFailure(ctx, updateFailedCmd); err != nil {
		return mq.Temporary(err)
	}

	return mq.Temporary(lastErr)
}

func (s *send) getMessageForProcessing(messageID int64) (*model.Message, error) {
	msg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return nil, ErrMessageNotFound
		}

		return nil, ErrDatabase
	}

	switch msg.Status {
	case model.MessageStatusCreated:
		return msg, nil

	case model.MessageStatusSending:
		if msg.LastAttemptAt != nil && time.Since(*msg.LastAttemptAt) < 5*time.Minute {
			s.logger.Warn("Message being processed by another consumer",
				zap.Int64("messageID", messageID),
				zap.Time("lastAttempt", *msg.LastAttemptAt))
			return nil, ErrMessageBeingProcessed
		}

		return msg, nil

	case model.MessageStatusSubmitted, model.MessageStatusFailedPerm, model.MessageStatusRefunded:
		s.logger.Info("Message already processed successfully",
			zap.Int64("messageID", messageID), zap.String("status", string(msg.Status)))
		return nil, ErrMessageAlreadyProcessed

	case model.MessageStatusFailedTemp:
		s.logger.Info("Message was temporarily failed, retrying", zap.Int64("messageID", messageID))
		return msg, nil

	default:
		s.logger.Error("Unknown message status",
			zap.String("status", string(msg.Status)),
			zap.Int64("messageID", messageID))
		return nil, ErrUnknownMessageStatus
	}
}

func (s *send) updateMessageToSending(ctx context.Context, cmd UpdateMessageToSendingCommand) error {
	staleThreshold := time.Now().Add(-5 * time.Minute)

	attempt := time.Now()
	msg := model.Message{
		ID:            cmd.MessageID,
		Status:        model.MessageStatusSending,
		AttemptCount:  cmd.AttemptCount,
		LastAttemptAt: &attempt,
		UpdatedAt:     time.Now(),
	}

	err := s.messageRepo.UpdateForSending(ctx, &msg, staleThreshold)
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrNoRowsAffected) {
		s.logger.Info("Message not updated to SENDING, possibly processed by another consumer",
			zap.Int64("messageID", cmd.MessageID))

		return ErrMessageBeingProcessed
	}

	s.logger.Error("Failed to update message for send attempt",
		zap.Error(err),
		zap.Int64("messageID", cmd.MessageID))

	return ErrDatabase
}

func (s *send) updateMessageSucceed(ctx context.Context, cmd UpdateMessageSuccessCommand) error {
	msg := model.Message{
		ID:            cmd.MessageID,
		Status:        model.MessageStatusSubmitted,
		ProviderMsgID: &cmd.ProviderMsgID,
		Provider:      &cmd.Provider,
		UpdatedAt:     time.Now(),
	}

	if err := s.messageRepo.Update(ctx, &msg); err != nil {
		s.logger.Error("Failed to update message after send attempt",
			zap.Int64("messageID", cmd.MessageID),
			zap.String("providerMessageID", cmd.ProviderMsgID),
			zap.String("provider", cmd.Provider),
			zap.Error(err))
	}

	txLog := model.TxLog{
		MessageID: cmd.MessageID,
		State:     model.TxLogStateSuccess,
		LastError: nil,
		UpdatedAt: time.Now(),
	}

	if err := s.txLogRepo.UpdateByMessageID(ctx, &txLog); err != nil {
		s.logger.Error("Failed to update tx_log to published",
			zap.Error(err),
			zap.Int64("messageID", cmd.MessageID),
			zap.Error(err))
	}

	return nil
}

func (s *send) updateMessageToPermanentFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error {
	msg := &model.Message{
		ID:        cmd.MessageID,
		Status:    model.MessageStatusFailedPerm,
		UpdatedAt: time.Now(),
	}

	txLog := &model.TxLog{
		MessageID:   cmd.MessageID,
		State:       model.TxLogStateFailed,
		Published:   false,
		PublishedAt: nil,
		LastError:   &cmd.LastError,
		UpdatedAt:   time.Now(),
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.messageRepo.Update(ctx, msg); err != nil {
			s.logger.Error("Failed to update message status after perm failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		if err := s.txLogRepo.UpdateForPermFailed(ctx, txLog); err != nil {
			s.logger.Error("Failed to update transaction log after perm failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		return nil
	})
}

func (s *send) updateMessageToTemporaryFailure(ctx context.Context, cmd UpdateMessageFailureCommand) error {
	msg := model.Message{
		ID:        cmd.MessageID,
		Status:    model.MessageStatusFailedTemp,
		UpdatedAt: time.Now(),
	}

	txLog := model.TxLog{
		MessageID: cmd.MessageID,
		LastError: &cmd.LastError,
		UpdatedAt: time.Now(),
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.messageRepo.Update(ctx, &msg); err != nil {
			s.logger.Error("Failed to update message status after temp failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		if err := s.txLogRepo.UpdateByMessageID(ctx, &txLog); err != nil {
			s.logger.Error("Failed to update transaction log after temp failure",
				zap.Int64("messageID", cmd.MessageID),
				zap.Error(err))
			return err
		}

		return nil
	})
}
