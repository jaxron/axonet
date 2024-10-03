package retry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/retry"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Successful request without retries", func(t *testing.T) {
		t.Parallel()

		middleware := retry.New(3, 10*time.Millisecond, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Retry on temporary error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		maxAttempts := uint64(3)
		middleware := retry.New(maxAttempts, 10*time.Millisecond, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			attempts++
			if attempts < int(maxAttempts) {
				return nil, errors.ErrTemporary
			}
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int(maxAttempts), attempts)
	})

	t.Run("Fail after max retries", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		maxAttempts := uint64(3)
		middleware := retry.New(maxAttempts, 10*time.Millisecond, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.ErrTemporary
		}

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, retry.ErrRetryFailed)
		assert.Equal(t, int(maxAttempts)+1, attempts) // The middleware makes one more attempt than maxAttempts
	})

	t.Run("No retry on permanent error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		middleware := retry.New(3, 10*time.Millisecond, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.ErrPermanent
		}

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, retry.ErrRetryFailed)
		assert.Equal(t, 1, attempts)
	})

	t.Run("Respect context cancellation", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		middleware := retry.New(5, 10*time.Millisecond, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 2 {
				cancel()
			}
			return nil, errors.ErrTemporary
		}

		resp, err := middleware.Process(ctx, &http.Client{}, req, handler)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, retry.ErrRetryFailed)
		assert.Equal(t, 2, attempts)
	})
}
