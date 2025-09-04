package main

import (
	"context"

	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/consumers"
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
			NewMQConnection,
			NewMQConsumer,
			NewMQPublisher,
			NewConnectionDB,
			repository.NewMessageRepository,
			repository.NewTxLogRepository,
			service.NewMessageService,
			consumers.NewHandler,
		),
		fx.Invoke(runSendConsumer),
	).Run()
}

func runSendConsumer(cfg *config.Config, handler *consumers.Handler, logger *zap.Logger,
	rabbit *mq.RabbitMQ, consumer mq.Consumer, lc fx.Lifecycle,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rabbit.DeclareTopology([]string{"sms.send"}); err != nil {
				logger.Error("declare topology failed", zap.Error(err))
				return err
			}
			logger.Info("queue declared", zap.String("queue", "sms.send"))

			go func() {
				if err := consumer.Consume(ctx, 1, "sms.send", handler.Handle); err != nil {
					logger.Error("consumer exited", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping worker-send")
			return rabbit.Close()
		},
	})
}

func NewConnectionDB(ctx context.Context, cfg config.Config, logger *zap.Logger) (*gorm.DB, error) {
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewMQConnection(cfg config.Config, logger *zap.Logger) (*mq.RabbitMQ, error) {
	return mq.NewConnection(cfg.RabbitMQ, logger)
}

func NewMQPublisher(rabbitMQ *mq.RabbitMQ) (mq.Publisher, error) {
	return rabbitMQ.CreatePublisher()
}

func NewMQConsumer(rabbitMQ *mq.RabbitMQ) (mq.Consumer, error) {
	return rabbitMQ.CreateConsumer()
}
