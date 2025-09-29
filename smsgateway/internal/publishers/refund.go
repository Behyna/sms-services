package publishers

import (
	"context"
	"encoding/json"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"go.uber.org/zap"
)

type RefundPublisher interface {
	Publish(ctx context.Context) error
}

type refundPublisher struct {
	service   service.MessageQueueService
	publisher mq.Publisher
	logger    *zap.Logger
}

func NewRefundPublisher(service service.MessageQueueService, publisher mq.Publisher, logger *zap.Logger) RefundPublisher {
	return &refundPublisher{service: service, publisher: publisher, logger: logger}
}

func (r *refundPublisher) Publish(ctx context.Context) error {
	refundRequests, err := r.service.FindRefundsToQueue(ctx, 100)
	if err != nil {
		return err
	}

	if len(refundRequests) == 0 {
		return nil
	}

	r.logger.Info("Publishing refunds", zap.Int("count", len(refundRequests)))

	successCount := 0
	for _, refundRequest := range refundRequests {
		body, _ := json.Marshal(refundRequest)
		if err := r.publisher.Publish(ctx, "", "sms.refund", body); err != nil {
			r.logger.Error("Failed to publish refund",
				zap.Error(err),
				zap.Int64("txLogID", refundRequest.TxLogID))
			continue
		}

		if err := r.service.MarkRefundAsQueued(ctx, refundRequest.TxLogID); err != nil {
			continue
		}

		successCount++
	}

	if successCount > 0 {
		r.logger.Info("Successfully published refunds",
			zap.Int("published", successCount),
			zap.Int("total", len(refundRequests)))
	}

	return nil
}
