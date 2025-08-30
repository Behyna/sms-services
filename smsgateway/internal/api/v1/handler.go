package v1

import (
	error2 "github.com/Behyna/sms-services/smsgateway/internal/error"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	logger *zap.Logger
}

func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) Pong(c *fiber.Ctx) error {
	return c.SendString("pong")
}

func (h *Handler) Message(c *fiber.Ctx) error {
	var request SendMessageRequest

	// TODO: add validation to request struct

	if err := c.BodyParser(&request); err != nil {
		h.logger.Warn("Failed to parse body",
			zap.Error(err),
			zap.String("body", string(c.Body())))
		return c.Status(fiber.StatusBadRequest).JSON(error2.Response{
			Error:   "Invalid request body",
			Message: "Request body could not be parsed",
		})
	}

	h.logger.Info("Message received successfully",
		zap.String("from", request.From),
		zap.String("to", request.To),
		zap.String("messageID", request.MessageID),
	)

	mockResponse := SendMessageResponse{Status: "QUEUED", MessageID: "12345"}
	return c.Status(fiber.StatusCreated).JSON(mockResponse)
}
