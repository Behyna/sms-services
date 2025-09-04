package consumers

import (
	"context"
	"encoding/json"

	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	service service.MessageService
	logger  *zap.Logger
}

func NewHandler(service service.MessageService, logger *zap.Logger) *Handler {
	return &Handler{service, logger}
}

func (h *Handler) Handle(ctx context.Context, body []byte) error {
	h.logger.Info("received send command", zap.ByteString("body", body))
	var cmd service.SendMessageCommand
	if err := json.Unmarshal(body, &cmd); err != nil {
		h.logger.Warn("invalid send command", zap.Error(err))
		return err
	}

	return h.service.SendMessage(ctx, cmd)
}
