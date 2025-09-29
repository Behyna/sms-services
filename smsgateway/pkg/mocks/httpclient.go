package mocks

import (
	"context"
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type HTTPClient struct {
	mock.Mock
}

func (_m *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	ret := _m.Called(ctx, url, headers)
	return ret.Get(0).(*http.Response), ret.Error(1)
}

func (_m *HTTPClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	ret := _m.Called(ctx, url, body, headers)
	return ret.Get(0).(*http.Response), ret.Error(1)
}

func (_m *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	ret := _m.Called(req)
	return ret.Get(0).(*http.Response), ret.Error(1)
}
