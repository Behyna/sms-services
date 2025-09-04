package smsprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Behyna/sms-services/smsgateway/pkg/httpclient"
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
		return Response{}, err
	}

	defer resp.Body.Close()

	var res Response
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Response{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("error from status code %d", resp.StatusCode)
	}

	return res, nil
}
