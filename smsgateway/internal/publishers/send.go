package publishers

import (
	"context"
	"encoding/json"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"go.uber.org/zap"
)

type SendPublisher interface {
	Publish(ctx context.Context) error
}

type sendPublisher struct {
	service   service.MessageQueueService
	publisher mq.Publisher
	logger    *zap.Logger
}

func NewSendPublisher(service service.MessageQueueService, publisher mq.Publisher, logger *zap.Logger) SendPublisher {
	return &sendPublisher{service: service, publisher: publisher, logger: logger}
}

func (s *sendPublisher) Publish(ctx context.Context) error {
	messages, err := s.service.FindMessagesToQueue(ctx, 100)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil
	}

	s.logger.Info("Publishing messages", zap.Int("count", len(messages)))

	successCount := 0
	for _, message := range messages {
		body, _ := json.Marshal(message)
		if err := s.publisher.Publish(ctx, "", "sms.send", body); err != nil {
			s.logger.Error("Failed to publish message",
				zap.Error(err),
				zap.Int64("messageID", message.MessageID))
			continue
		}

		if err := s.service.MarkMessageAsQueued(ctx, message.MessageID); err != nil {
			continue
		}

		successCount++
	}

	if successCount > 0 {
		s.logger.Info("Successfully published messages to send",
			zap.Int("published", successCount),
			zap.Int("total", len(messages)))
	}

	return nil
}
