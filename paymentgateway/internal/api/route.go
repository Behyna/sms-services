package api

import (
	v1 "github.com/Behyna/sms-services/paymentgateway/internal/api/v1"
	"github.com/gofiber/fiber/v2"
)

const prefixV1 = "api/v1/"

func SetupRoutes(app *fiber.App, handler *v1.Handler) {
	app.Get("/ping", handler.Pong)
	app.Post(prefixV1+"users", handler.CreateUsersBalance)
	app.Post(prefixV1+"users/balance", handler.GetUserBalance)
	app.Post(prefixV1+"user/increase/balance", handler.UpdateUserBalance)
	app.Post(prefixV1+"user/decrease/balance", handler.DecreaseUserBalance)
}
