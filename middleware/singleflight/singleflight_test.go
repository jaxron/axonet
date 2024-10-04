package singleflight_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/singleflight"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleFlightMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Deduplicate concurrent identical requests", func(t *testing.T) {
		t.Parallel()

		middleware := singleflight.New()
		middleware.SetLogger(logger.NewBasicLogger())

		requestCount := 0
		var mu sync.Mutex

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			mu.Lock()
			requestCount++
			mu.Unlock()
			time.Sleep(100 * time.Millisecond) // Simulate work
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		makeRequest := func() (*http.Response, error) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			return middleware.Process(context.Background(), &http.Client{}, req, handler)
		}

		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := makeRequest()
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}()
		}
		wg.Wait()

		assert.Equal(t, 1, requestCount, "Expected only one request to be processed")
	})

	t.Run("Different requests are not deduplicated", func(t *testing.T) {
		t.Parallel()

		middleware := singleflight.New()
		middleware.SetLogger(logger.NewBasicLogger())

		requestCount := 0
		var mu sync.Mutex

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			mu.Lock()
			requestCount++
			mu.Unlock()
			time.Sleep(100 * time.Millisecond) // Simulate work
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		makeRequest := func(url string) (*http.Response, error) {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			return middleware.Process(context.Background(), &http.Client{}, req, handler)
		}

		var wg sync.WaitGroup
		urls := []string{"http://example.com/1", "http://example.com/2", "http://example.com/3"}
		for _, url := range urls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				resp, err := makeRequest(url)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}(url)
		}
		wg.Wait()

		assert.Equal(t, len(urls), requestCount, "Expected each different request to be processed")
	})

	t.Run("Requests with different bodies are not deduplicated", func(t *testing.T) {
		t.Parallel()

		middleware := singleflight.New()
		middleware.SetLogger(logger.NewBasicLogger())

		requestCount := 0
		var mu sync.Mutex

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			mu.Lock()
			requestCount++
			mu.Unlock()
			time.Sleep(100 * time.Millisecond) // Simulate work
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		makeRequest := func(body string) (*http.Response, error) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader(body))
			return middleware.Process(context.Background(), &http.Client{}, req, handler)
		}

		var wg sync.WaitGroup
		bodies := []string{"body1", "body2", "body3"}
		for _, body := range bodies {
			wg.Add(1)
			go func(body string) {
				defer wg.Done()
				resp, err := makeRequest(body)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}(body)
		}
		wg.Wait()

		assert.Equal(t, len(bodies), requestCount, "Expected each request with different body to be processed")
	})

	t.Run("Error handling", func(t *testing.T) {
		t.Parallel()

		middleware := singleflight.New()
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return nil, errors.ErrNetwork
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, errors.ErrNetwork)
	})

	t.Run("Request body can be read after key generation", func(t *testing.T) {
		t.Parallel()

		middleware := singleflight.New()
		middleware.SetLogger(logger.NewBasicLogger())

		body := "test body"
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			assert.Equal(t, body, string(bodyBytes), "Request body should be readable")
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		req := httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader(body))
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
