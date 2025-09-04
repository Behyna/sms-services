package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Behyna/sms-services/smsgateway/pkg/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock server setup
func setupMockServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	})
	return httptest.NewServer(handler)
}

func TestHttpClient_Get(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	client := httpclient.NewHTTPClient(5 * time.Second)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL+"/test", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, `{"message": "success"}`, string(body))
}

func TestHttpClient_Post(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	client := httpclient.NewHTTPClient(5 * time.Second)
	ctx := context.Background()

	resp, err := client.Post(ctx, server.URL+"/test", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, `{"message": "success"}`, string(body))
}

func TestHttpClient_Do(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	client := httpclient.NewHTTPClient(5 * time.Second)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, `{"message": "success"}`, string(body))
}
