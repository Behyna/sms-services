package v1

import (
	error2 "github.com/Behyna/sms-services/smsgateway/internal/error"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	logger  *zap.Logger
	service service.MessageService
}

func NewHandler(logger *zap.Logger, service service.MessageService) *Handler {
	return &Handler{logger: logger, service: service}
}

func (h *Handler) Pong(c *fiber.Ctx) error {
	return c.SendString("pong")
}

func (h *Handler) Message(c *fiber.Ctx) error {
	ctx := c.UserContext()

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

	cmd := service.CreateMessageCommand{
		ClientMessageID: request.MessageID,
		FromMSISDN:      request.From,
		ToMSISDN:        request.To,
		Text:            request.Text,
	}
	if err := h.service.CreateMessageTransaction(ctx, cmd); err != nil {
		h.logger.Error("Failed to create message transaction",
			zap.Error(err),
			zap.String("from", request.From),
			zap.String("to", request.To),
			zap.String("messageID", request.MessageID),
		)

		return c.Status(fiber.StatusInternalServerError).JSON(error2.Response{
			Error:   "Internal server error",
			Message: "Could not process the request",
		})
	}

	h.logger.Info("Message received successfully",
		zap.String("from", request.From),
		zap.String("to", request.To),
		zap.String("messageID", request.MessageID),
	)

	return c.Status(fiber.StatusCreated).JSON(SendMessageResponse{Status: string(model.MessageStatusQueued), MessageID: request.MessageID})
}
