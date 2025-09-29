package consumers

import (
	"context"
	"encoding/json"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"go.uber.org/zap"
)

type SendConsumer interface {
	Consume(ctx context.Context) error
}

type sendConsumer struct {
	service  service.SendService
	consumer mq.Consumer
	logger   *zap.Logger
}

func NewSendConsumer(service service.SendService, consumer mq.Consumer, logger *zap.Logger) SendConsumer {
	return &sendConsumer{
		service:  service,
		consumer: consumer,
		logger:   logger,
	}
}

func (s *sendConsumer) Consume(ctx context.Context) error {
	return s.consumer.Consume(ctx, 1, "sms.send", s.handleMessage)
}

func (s *sendConsumer) handleMessage(ctx context.Context, body []byte) error {
	s.logger.Info("received send command", zap.ByteString("body", body))

	var cmd service.SendMessageCommand
	if err := json.Unmarshal(body, &cmd); err != nil {
		s.logger.Warn("invalid send command", zap.Error(err))
		return err
	}

	return s.service.SendMessage(ctx, cmd)
}
