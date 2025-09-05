package main

import (
	"context"
	"time"

	"github.com/Behyna/common/pkg/mysql"
	"github.com/Behyna/sms-services/paymentgateway/internal/api"
	v1 "github.com/Behyna/sms-services/paymentgateway/internal/api/v1"
	v2 "github.com/Behyna/sms-services/paymentgateway/internal/api/validator"
	"github.com/Behyna/sms-services/paymentgateway/internal/config"
	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
	"github.com/Behyna/sms-services/paymentgateway/internal/repository"
	"github.com/Behyna/sms-services/paymentgateway/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			NewConnectionDB,
			zap.NewProduction,
			NewFiberApp,
			NewValidator,
			metrics.NewMetrics,
			v2.NewXValidator,

			repository.NewUserBalanceRepository,
			repository.NewTransactionRepository,
			service.NewUserBalanceService,

			v1.NewHandler,
			metrics.NewSystemCollector,
			metrics.NewDatabaseMetricsCollector,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(
	app *fiber.App,
	handler *v1.Handler,
	cfg *config.Config,
	logger *zap.Logger,
	systemCollector *metrics.SystemCollector,
	dbCollector *metrics.DatabaseMetricsCollector,
	lc fx.Lifecycle,
) {
	api.SetupRoutes(app, handler)

	// lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start collectors (they already run goroutines internally)
			systemCollector.Start(30 * time.Second)
			dbCollector.Start(15 * time.Second)

			go func() {
				logger.Info("Starting server", zap.String("port", cfg.API.Port))
				if err := app.Listen(cfg.API.Port); err != nil {
					logger.Fatal("Failed to start server", zap.Error(err))
				}
			}()
			logger.Info("Server startup initiated with metrics enabled")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down server and metrics collectors")
			systemCollector.Stop()
			dbCollector.Stop()
			defer logger.Sync()
			return app.ShutdownWithContext(ctx)
		},
	})
}

func NewConnectionDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	ctx := context.Background()
	return mysql.NewConnection(ctx, cfg.Database, logger)
}

func NewFiberApp(m *metrics.Metrics, logger *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ServerHeader: "PaymentGateway/1.0",
		AppName:      "SMS Payment Gateway",
	})

	// Add health check middleware first
	app.Use(metrics.HealthCheckMiddleware())

	// Add custom metrics middleware
	app.Use(metrics.HTTPMetricsMiddleware(m, logger))

	// Add Prometheus metrics endpoint
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	return app
}

func NewValidator() *validator.Validate {
	return validator.New()
}
