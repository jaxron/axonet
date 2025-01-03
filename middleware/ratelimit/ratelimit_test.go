package ratelimit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/ratelimit"
	clientErrors "github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Respect rate limit", func(t *testing.T) {
		t.Parallel()

		requestsPerSecond := 10.0
		burst := 1
		middleware := ratelimit.New(requestsPerSecond, burst)
		middleware.SetLogger(logger.NewBasicLogger())

		makeRequest := func(ctx context.Context) error {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)
			_, err := middleware.Process(ctx, &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			})
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
		require.ErrorIs(t, err, clientErrors.ErrTimeout)

		// After waiting, we should be able to make another request
		time.Sleep(time.Second / time.Duration(requestsPerSecond))
		err = makeRequest(context.Background())
		require.NoError(t, err)
	})

	t.Run("Burst allows multiple requests", func(t *testing.T) {
		t.Parallel()

		requestsPerSecond := 1.0
		burst := 3
		middleware := ratelimit.New(requestsPerSecond, burst)
		middleware.SetLogger(logger.NewBasicLogger())

		makeRequest := func(ctx context.Context) error {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil).WithContext(ctx)
			_, err := middleware.Process(ctx, &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			})
			return err
		}

		// Burst number of requests should succeed immediately
		for range burst {
			err := makeRequest(context.Background())
			require.NoError(t, err)
		}

		// The next request should be rate limited
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := makeRequest(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, clientErrors.ErrTimeout)
	})
}
