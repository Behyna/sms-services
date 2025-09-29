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
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
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
			NewSMSProvider,
			service.NewProviderService,
			service.NewSendService,

			consumers.NewSendConsumer,
		),
		fx.Invoke(runSendConsumer),
	).Run()
}

func runSendConsumer(cfg *config.Config, sendConsumer consumers.SendConsumer, logger *zap.Logger,
	rabbit *mq.RabbitMQ, lc fx.Lifecycle,
) {
	appCtx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rabbit.DeclareTopology([]string{"sms.send"}); err != nil {
				logger.Error("declare topology failed", zap.Error(err))
				return err
			}
			logger.Info("queue declared", zap.String("queue", "sms.send"))

			go func() {
				if err := sendConsumer.Consume(appCtx); err != nil {
					logger.Error("consumer exited", zap.Error(err))
				}
			}()

			logger.Info("send consumer started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping send consumer")
			cancel()
			return rabbit.Close()
		},
	})
}

func NewConnectionDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	ctx := context.Background()
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewSMSProvider(cfg *config.Config) smsprovider.Provider {
	client := httpclient.NewHTTPClient(cfg.Provider.Timeout)
	return smsprovider.NewSMSProvider(cfg.Provider, client)
}

func NewMQConnection(cfg *config.Config, logger *zap.Logger) (*mq.RabbitMQ, error) {
	return mq.NewConnection(cfg.RabbitMQ, logger)
}

func NewMQConsumer(rabbitMQ *mq.RabbitMQ) (mq.Consumer, error) {
	return rabbitMQ.CreateConsumer()
}
