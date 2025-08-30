package main

import (
	"log"

	"github.com/Behyna/sms-services/smsgateway/internal/api"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	api.SetupRoutes(app, api.NewHandler())

	log.Fatal(app.Listen(":8080"))
}
