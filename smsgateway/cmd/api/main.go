package main

import (
	"context"

	"github.com/Behyna/common/pkg/httpclient"
	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/sms-services/smsgateway/internal/api"
	v1 "github.com/Behyna/sms-services/smsgateway/internal/api/v1"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/error"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			zap.NewProduction,
			NewFiberApp,

			NewConnectionDB,

			repository.NewMessageRepository,
			repository.NewTxLogRepository,
			repository.NewTransactionManager,

			NewSMSProvider,
			NewPaymentGateway,

			service.NewPaymentService,
			service.NewProviderService,
			service.NewMessageService,

			service.NewMessageWorkflowService,

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

func NewConnectionDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	ctx := context.Background()
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewSMSProvider(cfg *config.Config) smsprovider.Provider {
	client := httpclient.NewHTTPClient(cfg.Provider.Timeout)
	return smsprovider.NewSMSProvider(cfg.Provider, client)
}

func NewPaymentGateway(cfg *config.Config) paymentgateway.PaymentGateway {
	client := httpclient.NewHTTPClient(cfg.PaymentGateway.Timeout)
	return paymentgateway.NewPaymentGateway(cfg.PaymentGateway, client)
}

func NewFiberApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(),
	})
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	config.Encoding = "console"

	return config.Build()
}
