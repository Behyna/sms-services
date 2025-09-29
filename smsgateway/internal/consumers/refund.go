package consumers

import (
	"context"
	"encoding/json"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"go.uber.org/zap"
)

type RefundConsumer interface {
	Consume(ctx context.Context) error
}

type refundConsumer struct {
	service  service.RefundService
	consumer mq.Consumer
	logger   *zap.Logger
}

func NewRefundConsumer(service service.RefundService, consumer mq.Consumer, logger *zap.Logger) RefundConsumer {
	return &refundConsumer{service: service, consumer: consumer, logger: logger}
}

func (r *refundConsumer) Consume(ctx context.Context) error {
	return r.consumer.Consume(ctx, 1, "sms.refund", r.handleMessage)
}

func (r *refundConsumer) handleMessage(ctx context.Context, body []byte) error {
	r.logger.Info("received refund command", zap.ByteString("body", body))

	var cmd service.ProcessRefundCommand
	if err := json.Unmarshal(body, &cmd); err != nil {
		r.logger.Warn("invalid refund command", zap.Error(err))
		return err
	}

	return r.service.Refund(ctx, cmd)
}
