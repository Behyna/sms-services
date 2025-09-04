package main

import (
	"context"

	"github.com/Behyna/common/pkg/httpclient"
	"github.com/Behyna/common/pkg/mq"
	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/common/pkg/smsprovider"
	"github.com/Behyna/sms-services/smsgateway/internal/api"
	v1 "github.com/Behyna/sms-services/smsgateway/internal/api/v1"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			// TODO: wrap fiber,
			zap.NewProduction,
			fiber.New,

			NewConnectionDB,
			NewMQConnection,
			NewMQPublisher,
			repository.NewMessageRepository,
			repository.NewTxLogRepository,
			service.NewMessageService,

			v1.NewHandler,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(app *fiber.App, handler *v1.Handler, cfg *config.Config, logger *zap.Logger, lc fx.Lifecycle) {
	api.SetupRoutes(app, handler)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("Starting server", zap.String("port", cfg.API.Port))
				if err := app.Listen(cfg.API.Port); err != nil {
					logger.Fatal("Failed to start server", zap.Error(err))
				}
			}()
			logger.Info("Server startup initiated")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down server")
			defer logger.Sync()
			return app.ShutdownWithContext(ctx)
		},
	})
}

func setupQueues(rabbitMQ *mq.RabbitMQ, logger *zap.Logger, lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			queues := []string{"sms.send", "sms.refund"}
			if err := rabbitMQ.DeclareTopology(queues); err != nil {
				logger.Error("Failed to declare queues", zap.Error(err))
				return err
			}

			logger.Info("Queues declared successfully for publishing")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing RabbitMQ connection")
			return rabbitMQ.Close()
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

func NewProvider(cfg config.Config) smsprovider.Provider {
	client := httpclient.NewHTTPClient(cfg.Provider.Timeout)
	return smsprovider.NewSMSProvider(cfg.Provider, client)
}
