package main

import (
	"context"

	"github.com/Behyna/common/pkg/httpclient"
	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/consumers"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			zap.NewProduction,
			NewConnectionDB,
			NewMQConnection,
			NewMQConsumer,

			repository.NewMessageRepository,
			repository.NewTxLogRepository,
			repository.NewTransactionManager,
			NewPaymentGateway,
			service.NewPaymentService,
			service.NewRefundService,

			consumers.NewRefundConsumer,
		),
		fx.Invoke(runRefundConsumer),
	).Run()
}

func runRefundConsumer(cfg *config.Config, refundConsumer consumers.RefundConsumer, logger *zap.Logger,
	rabbit *mq.RabbitMQ, lc fx.Lifecycle,
) {
	appCtx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rabbit.DeclareTopology([]string{"sms.refund"}); err != nil {
				logger.Error("declare topology failed", zap.Error(err))
				return err
			}
			logger.Info("queue declared", zap.String("queue", "sms.refund"))

			go func() {
				if err := refundConsumer.Consume(appCtx); err != nil {
					logger.Error("consumer exited", zap.Error(err))
				}
			}()

			logger.Info("refund consumer started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping refund consumer")
			cancel()
			return rabbit.Close()
		},
	})
}

func NewConnectionDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	ctx := context.Background()
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewPaymentGateway(cfg *config.Config) paymentgateway.PaymentGateway {
	client := httpclient.NewHTTPClient(cfg.PaymentGateway.Timeout)
	return paymentgateway.NewPaymentGateway(cfg.PaymentGateway, client)
}

func NewMQConnection(cfg *config.Config, logger *zap.Logger) (*mq.RabbitMQ, error) {
	return mq.NewConnection(cfg.RabbitMQ, logger)
}

func NewMQConsumer(rabbitMQ *mq.RabbitMQ) (mq.Consumer, error) {
	return rabbitMQ.CreateConsumer()
}
