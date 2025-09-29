package paymentgateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Behyna/sms-services/smsgateway/pkg/mocks"
	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func matchRequestBody(request paymentgateway.UpdateUserBalanceRequest) interface{} {
	return mock.MatchedBy(func(body interface{}) bool {
		buf, ok := body.(*bytes.Buffer)
		if !ok {
			return false
		}

		var req paymentgateway.UpdateUserBalanceRequest
		if err := json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&req); err != nil {
			return false
		}

		return req.UserID == request.UserID && req.Amount == request.Amount
	})
}

func TestPaymentGateway_Refund(t *testing.T) {
	cfg := paymentgateway.Config{
		BaseURL: "https://api.payment.test",
		Timeout: 30 * time.Second,
	}

	refundURL := "https://api.payment.test/user/increase/balance"
	headers := map[string]string{"Content-Type": "application/json"}

	request := paymentgateway.UpdateUserBalanceRequest{
		UserID:         "user123",
		Amount:         1,
		IdempotencyKey: "abc-123-def-456",
	}

	t.Run("successful refund", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		body := `{
			"code": "success",
			"message": "user balance updated successfully",
			"x_track_id": "",
			"result": {}
		}`

		successResponse := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		mockClient.On("Post", context.Background(), refundURL, matchRequestBody(request),
			headers).Return(successResponse, nil)

		response, err := pg.Refund(context.Background(), request)

		assert.NoError(t, err)
		assert.Equal(t, "success", response.Code)
		assert.Equal(t, "user balance updated successfully", response.Message)
		mockClient.AssertExpectations(t)
	})

	t.Run("timeout error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		mockClient.On("Post", context.Background(), refundURL, matchRequestBody(request),
			headers).Return((*http.Response)(nil), context.DeadlineExceeded)

		ctx := context.Background()
		response, err := pg.Refund(ctx, request)

		assert.Error(t, err)
		assert.Equal(t, paymentgateway.ErrTimeout, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("network error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		networkErr := errors.New("network connection failed")
		resp := &http.Response{}

		mockClient.On("Post", context.Background(), refundURL, matchRequestBody(request),
			headers).Return(resp, networkErr)

		response, err := pg.Refund(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, networkErr, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		invalidJSON := `{"code": "SUCCESS", "message":`
		successResponse := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(invalidJSON)),
		}

		mockClient.On("Post", context.Background(), refundURL,
			matchRequestBody(request), headers).Return(successResponse, nil)

		response, err := pg.Refund(context.Background(), request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding error")
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("server error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}

		mockClient.On("Post", context.Background(), refundURL,
			matchRequestBody(request), headers).Return(resp, nil)

		response, err := pg.Refund(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, paymentgateway.ErrServerError, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})
}

func TestPaymentGateway_Charge(t *testing.T) {
	cfg := paymentgateway.Config{
		BaseURL: "https://api.payment.test",
		Timeout: 30 * time.Second,
	}

	chargeURL := cfg.BaseURL + paymentgateway.DecreaseBalanceEndpoint
	headers := map[string]string{"Content-Type": "application/json"}

	request := paymentgateway.UpdateUserBalanceRequest{
		UserID:         "user123",
		Amount:         1,
		IdempotencyKey: "abc-123-def-456",
	}

	t.Run("successful charge", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		body := `{
			"code": "success",
			"message": "user balance updated successfully",
			"x_track_id": "",
			"result": {}
		}`

		successResponse := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		mockClient.On("Post", context.Background(), chargeURL, matchRequestBody(request),
			headers).Return(successResponse, nil)

		response, err := pg.Charge(context.Background(), request)

		assert.NoError(t, err)
		assert.Equal(t, "success", response.Code)
		assert.Equal(t, "user balance updated successfully", response.Message)
		mockClient.AssertExpectations(t)
	})

	t.Run("timeout error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		mockClient.On("Post", context.Background(), chargeURL, matchRequestBody(request),
			headers).Return((*http.Response)(nil), context.DeadlineExceeded)

		ctx := context.Background()
		response, err := pg.Charge(ctx, request)

		assert.Error(t, err)
		assert.Equal(t, paymentgateway.ErrTimeout, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("network error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		networkErr := errors.New("network connection failed")
		resp := &http.Response{}

		mockClient.On("Post", context.Background(), chargeURL, matchRequestBody(request),
			headers).Return(resp, networkErr)

		response, err := pg.Charge(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, networkErr, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		invalidJSON := `{"code": "SUCCESS", "message":`
		successResponse := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(invalidJSON)),
		}

		mockClient.On("Post", context.Background(), chargeURL,
			matchRequestBody(request), headers).Return(successResponse, nil)

		response, err := pg.Charge(context.Background(), request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding error")
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})

	t.Run("server error", func(t *testing.T) {
		mockClient := &mocks.HTTPClient{}
		pg := paymentgateway.NewPaymentGateway(cfg, mockClient)

		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}

		mockClient.On("Post", context.Background(), chargeURL,
			matchRequestBody(request), headers).Return(resp, nil)

		response, err := pg.Charge(context.Background(), request)

		assert.Error(t, err)
		assert.Equal(t, paymentgateway.ErrServerError, err)
		assert.Empty(t, response)
		mockClient.AssertExpectations(t)
	})
}
