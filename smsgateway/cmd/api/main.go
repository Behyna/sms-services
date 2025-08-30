package main

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/internal/api"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			// TODO: wrap fiber
			fiber.New,
			api.NewHandler,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(app *fiber.App, handler *api.Handler, cfg *config.Config, lc fx.Lifecycle) {
	api.SetupRoutes(app, handler)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go app.Listen(cfg.API.Port)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
}
