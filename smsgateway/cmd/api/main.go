package main

import (
	"log"

	"github.com/Behyna/sms-services/smsgateway/internal/api"
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	api.SetupRoutes(app, api.NewHandler())

	log.Fatal(app.Listen(cfg.API.Port))
}
