package smsprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Behyna/common/pkg/httpclient"
)

type Provider interface {
	Send(ctx context.Context, from string, to string, text string) (res Response, err error)
}

type Config struct {
	Enable   bool          `mapstructure:"enable"`
	URL      string        `mapstructure:"url"`
	Timeout  time.Duration `mapstructure:"timeout"`
	MaxRetry int           `mapstructure:"max_retry"`
}

type SMSProvider struct {
	cfg    Config
	client httpclient.HTTPClient
}

func NewSMSProvider(cfg Config, client httpclient.HTTPClient) Provider {
	return &SMSProvider{cfg: cfg, client: client}
}

func (s *SMSProvider) Send(ctx context.Context, from string, to string, text string) (Response, error) {
	resp, err := s.client.Post(ctx, s.cfg.URL, nil, nil) // TODO:: Replace nil with actual body
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return Response{}, errors.New(ErrorCodeTimeout)
		}

		return Response{}, errors.New(ErrorCodeNetworkError)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 400:
			return Response{}, errors.New(ErrorCodeInvalidNumber)
		case 500, 502, 503, 504:
			return Response{}, errors.New(ErrorCodeServerError)
		default:
			return Response{}, errors.New(ErrorCodeServerError)
		}
	}

	var res Response
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Response{}, errors.New(ErrorCodeServerError)
	}

	return res, nil
}
