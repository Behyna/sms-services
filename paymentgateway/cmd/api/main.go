package api

import (
	"github.com/Behyna/sms-services/paymentgateway/internal/config"
	"github.com/gofiber/fiber"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			zap.NewProduction,
			fiber.New,
		),
		fx.Invoke(),
	).Run()
}
