package main

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/internal/api"
	"github.com/Behyna/sms-services/smsgateway/internal/api/v1"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/internal/database"
	"github.com/Behyna/sms-services/smsgateway/internal/repository"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			// TODO: wrap fiber,
			zap.NewProduction,
			fiber.New,
			v1.NewHandler,
			database.NewConnection,
			repository.NewMessageRepository,
			repository.NewTxLogRepository,
			service.NewMessageService,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(app *fiber.App, handler *v1.Handler, cfg *config.Config, logger *zap.Logger, lc fx.Lifecycle) {
	api.SetupRoutes(app, handler)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting server", zap.String("port", cfg.API.Port))
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
