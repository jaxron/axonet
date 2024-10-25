package circuitbreaker_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/circuitbreaker"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ErrFailed = errors.New("simulated failure")

func TestCircuitBreakerMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Success scenario", func(t *testing.T) {
		t.Parallel()

		middleware := circuitbreaker.New(5, 10*time.Second, 30*time.Second)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Circuit opens after multiple failures", func(t *testing.T) {
		t.Parallel()

		middleware := circuitbreaker.New(3, 10*time.Second, 1*time.Second)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		failingHandler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return nil, ErrFailed
		}

		// Fail 3 times to open the circuit
		for range 3 {
			_, err := middleware.Process(context.Background(), &http.Client{}, req, failingHandler)
			require.Error(t, err)
		}

		// The next call should return ErrCircuitOpen
		_, err := middleware.Process(context.Background(), &http.Client{}, req, failingHandler)
		require.Error(t, err)
		assert.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
	})

	t.Run("Circuit half-open state", func(t *testing.T) {
		t.Parallel()

		middleware := circuitbreaker.New(3, 10*time.Second, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		failingHandler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return nil, ErrFailed
		}

		// Fail 3 times to open the circuit
		for range 3 {
			_, err := middleware.Process(context.Background(), &http.Client{}, req, failingHandler)
			require.Error(t, err)
		}

		// Wait for the circuit to enter half-open state
		time.Sleep(200 * time.Millisecond)

		successHandler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// The circuit should now be half-open and allow one request
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, successHandler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The circuit should now be closed and allow more requests
		resp, err = middleware.Process(context.Background(), &http.Client{}, req, successHandler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
