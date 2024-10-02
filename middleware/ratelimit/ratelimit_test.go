package ratelimit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/ratelimit"
	clientContext "github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterMiddleware(t *testing.T) {
	t.Run("Respect rate limit", func(t *testing.T) {
		requestsPerSecond := 10.0
		burst := 1
		middleware := ratelimit.New(requestsPerSecond, burst)
		middleware.SetLogger(logger.NewBasicLogger())

		makeRequest := func(ctx context.Context) error {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)
			clientCtx := &clientContext.Context{
				Client: &http.Client{},
				Req:    req,
				Next: func(ctx *clientContext.Context) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK}, nil
				},
			}
			_, err := middleware.Process(clientCtx)
			return err
		}

		// Make burst+1 requests
		for i := 0; i <= burst; i++ {
			err := makeRequest(context.Background())
			require.NoError(t, err)
		}

		// The next request should be rate limited
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := makeRequest(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, ratelimit.ErrRateLimitExceeded)

		// After waiting, we should be able to make another request
		time.Sleep(time.Second / time.Duration(requestsPerSecond))
		err = makeRequest(context.Background())
		require.NoError(t, err)
	})

	t.Run("Burst allows multiple requests", func(t *testing.T) {
		requestsPerSecond := 1.0
		burst := 3
		middleware := ratelimit.New(requestsPerSecond, burst)
		middleware.SetLogger(logger.NewBasicLogger())

		makeRequest := func(ctx context.Context) error {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)
			clientCtx := &clientContext.Context{
				Client: &http.Client{},
				Req:    req,
				Next: func(ctx *clientContext.Context) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK}, nil
				},
			}
			_, err := middleware.Process(clientCtx)
			return err
		}

		// Burst number of requests should succeed immediately
		for i := 0; i < burst; i++ {
			err := makeRequest(context.Background())
			require.NoError(t, err)
		}

		// The next request should be rate limited
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := makeRequest(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, ratelimit.ErrRateLimitExceeded)
	})
}
