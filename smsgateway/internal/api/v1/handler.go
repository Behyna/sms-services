package v1

import (
	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	logger  *zap.Logger
	service service.MessageWorkflowService
}

func NewHandler(logger *zap.Logger, service service.MessageWorkflowService) *Handler {
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    constants.ErrCodeInvalidRequestBody,
			"message": constants.GetErrorMessage(constants.ErrCodeInvalidRequestBody),
		})
	}

	cmd := service.CreateMessageCommand{
		ClientMessageID: request.MessageID,
		FromMSISDN:      request.From,
		ToMSISDN:        request.To,
		Text:            request.Text,
	}

	resp, err := h.service.CreateMessage(ctx, cmd)
	if err != nil {
		h.logger.Error("Failed to create message transaction",
			zap.Error(err),
			zap.String("from", request.From),
			zap.String("to", request.To),
			zap.String("messageID", request.MessageID),
		)

		return err
	}

	h.logger.Info("Message received successfully",
		zap.String("from", request.From),
		zap.String("to", request.To),
		zap.String("messageID", request.MessageID),
	)

	return c.Status(fiber.StatusCreated).JSON(
		SendMessageResponse{Status: string(model.MessageStatusCreated), MessageID: resp.MessageID})
}
