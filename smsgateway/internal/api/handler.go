package api

import "github.com/gofiber/fiber/v2"

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Pong(c *fiber.Ctx) error {
	return c.SendString("pong")
}
