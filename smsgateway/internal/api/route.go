package api

import (
	"github.com/Behyna/sms-services/smsgateway/internal/api/v1"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, handler *v1.Handler) {
	app.Get("/ping", handler.Pong)
	app.Post("/v1/message", handler.Message)
}
