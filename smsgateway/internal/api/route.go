package api

import "github.com/gofiber/fiber/v2"

func SetupRoutes(app *fiber.App, handler *Handler) {
	app.Get("/ping", handler.Pong)
}
