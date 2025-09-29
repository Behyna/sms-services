package main

import (
	"context"
	"time"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/publishers"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
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
			NewMQPublisher,

			repository.NewTxLogRepository,

			service.NewMessageQueueService,

			publishers.NewRefundPublisher,
		),
		fx.Invoke(runRefundPublisher),
	).Run()
}

func runRefundPublisher(cfg *config.Config, publisher publishers.RefundPublisher, logger *zap.Logger,
	rabbit *mq.RabbitMQ, lc fx.Lifecycle) {
	appCtx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rabbit.DeclareTopology([]string{"sms.refund"}); err != nil {
				logger.Error("declare topology failed", zap.Error(err))
				return err
			}

			logger.Info("queue declared", zap.String("queue", "sms.refund"))

			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						if err := publisher.Publish(appCtx); err != nil {
							logger.Error("failed to publish refund tx", zap.Error(err))
						}
					case <-appCtx.Done():
						logger.Info("publisher context cancelled")
						return
					}
				}
			}()

			logger.Info("refund publisher started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping send publisher")
			cancel()
			return rabbit.Close()
		},
	})
}

func NewConnectionDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	ctx := context.Background()
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewMQConnection(cfg *config.Config, logger *zap.Logger) (*mq.RabbitMQ, error) {
	return mq.NewConnection(cfg.RabbitMQ, logger)
}

func NewMQPublisher(rabbitMQ *mq.RabbitMQ) (mq.Publisher, error) {
	return rabbitMQ.CreatePublisher()
}
