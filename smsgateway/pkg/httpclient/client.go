package httpclient

import (
	"context"
	"io"
	"net/http"
	"time"
)

var _ HTTPClient = (*httpClient)(nil)

type HTTPClient interface {
	Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error)
	Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
	SetTimeout(time time.Duration) *httpClient
}

type httpClient struct {
	Client *http.Client
	Time   time.Duration
}

func NewHTTPClient(timeout time.Duration) HTTPClient {
	return &httpClient{Client: &http.Client{Timeout: timeout}}
}

func (c *httpClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, headers)
	return c.Client.Do(req)
}

func (c *httpClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, headers)
	return c.Client.Do(req)
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}

func (c *httpClient) setHeaders(req *http.Request, headers map[string]string) {
	if len(headers) == 0 {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

func (c *httpClient) SetTimeout(timeout time.Duration) *httpClient {
	return &httpClient{
		Client: &http.Client{Timeout: timeout},
	}
}
