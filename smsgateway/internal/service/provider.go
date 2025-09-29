package service

import (
	"context"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"go.uber.org/zap"
)

type ProviderService interface {
	SendWithRetry(ctx context.Context, fromMSISDN, toMSISDN, text string) (smsprovider.Response, error)
}

type Provider struct {
	provider smsprovider.Provider
	logger   *zap.Logger
	config   smsprovider.Config
}

func NewProviderService(provider smsprovider.Provider, logger *zap.Logger, config *config.Config) ProviderService {
	return &Provider{provider: provider, logger: logger, config: config.Provider}
}

func (p *Provider) SendWithRetry(ctx context.Context, fromMSISDN, toMSISDN, text string) (smsprovider.Response, error) {
	var lastErr error

	for attempt := 1; attempt <= p.config.MaxRetry; attempt++ {
		p.logger.Debug("Attempting to send SMS",
			zap.Int("attempt", attempt),
			zap.String("to", toMSISDN),
			zap.String("from", fromMSISDN))

		providerCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)

		response, err := p.provider.Send(providerCtx, fromMSISDN, toMSISDN, text)
		cancel()

		if err == nil {
			p.logger.Info("SMS sent successfully",
				zap.String("messageId", response.MessageID),
				zap.String("status", response.Status),
				zap.Int("attempt", attempt))
			return response, nil
		}

		lastErr = err
		p.logger.Warn("SMS send attempt failed",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.String("to", toMSISDN))

		if err.Error() == smsprovider.ErrorCodeInvalidNumber {
			p.logger.Error("Non-retryable error encountered",
				zap.Error(err),
				zap.String("to", toMSISDN))
			return smsprovider.Response{}, err
		}

		if attempt < p.config.MaxRetry {
			delay := time.Duration(attempt) * 100 * time.Millisecond
			p.logger.Debug("Waiting before retry", zap.Duration("delay", delay))

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return smsprovider.Response{}, ctx.Err()
			}
		}
	}

	p.logger.Error("All retry attempts exhausted",
		zap.Error(lastErr),
		zap.Int("maxRetries", p.config.MaxRetry),
		zap.String("to", toMSISDN))

	return smsprovider.Response{}, lastErr
}
