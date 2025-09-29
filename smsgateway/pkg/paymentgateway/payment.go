package paymentgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Behyna/common/pkg/httpclient"
)

const (
	IncreaseBalanceEndpoint = "/user/increase/balance"
	DecreaseBalanceEndpoint = "/user/decrease/balance"
)

type PaymentGateway interface {
	Refund(ctx context.Context, request UpdateUserBalanceRequest) (Response, error)
	Charge(ctx context.Context, request UpdateUserBalanceRequest) (Response, error)
}

type paymentGateway struct {
	client httpclient.HTTPClient
	config Config
}

func NewPaymentGateway(cfg Config, client httpclient.HTTPClient) PaymentGateway {
	return &paymentGateway{config: cfg, client: client}
}

func (p *paymentGateway) Refund(ctx context.Context, request UpdateUserBalanceRequest) (Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return Response{}, fmt.Errorf("encoding error: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := p.client.Post(ctx, p.config.BaseURL+IncreaseBalanceEndpoint, &buf, headers)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return Response{}, ErrTimeout
		}

		return Response{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == StatusOK {
		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return Response{}, fmt.Errorf("decoding error: %w", err)
		}

		return response, nil
	}

	err = MapStatusToError(resp.StatusCode)

	return Response{}, err
}

func (p *paymentGateway) Charge(ctx context.Context, request UpdateUserBalanceRequest) (Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return Response{}, fmt.Errorf("encoding error: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := p.client.Post(ctx, p.config.BaseURL+DecreaseBalanceEndpoint, &buf, headers)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return Response{}, ErrTimeout
		}

		return Response{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == StatusOK {

		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return Response{}, fmt.Errorf("decoding error: %w", err)
		}

		return response, nil
	}

	err = MapStatusToError(resp.StatusCode)

	return Response{}, err
}
