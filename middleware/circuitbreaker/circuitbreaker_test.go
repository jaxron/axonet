package circuitbreaker_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/middleware/circuitbreaker"
	clientContext "github.com/jaxron/axonet/pkg/client/context"
	clientErrors "github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerMiddleware(t *testing.T) {
	t.Run("Success scenario", func(t *testing.T) {
		middleware := circuitbreaker.NewCircuitBreakerMiddleware(5, 10*time.Second, 30*time.Second)
		middleware.SetLogger(logger.NewBasicLogger())

		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			Next: func(*clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Circuit opens after multiple failures", func(t *testing.T) {
		middleware := circuitbreaker.NewCircuitBreakerMiddleware(3, 10*time.Second, 1*time.Second)
		middleware.SetLogger(logger.NewBasicLogger())

		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			Next: func(*clientContext.Context) (*http.Response, error) {
				return nil, errors.New("simulated failure")
			},
		}

		// Fail 3 times to open the circuit
		for i := 0; i < 3; i++ {
			_, err := middleware.Process(ctx)
			require.Error(t, err)
		}

		// The next call should return ErrCircuitOpen
		_, err := middleware.Process(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, clientErrors.ErrCircuitOpen))
	})

	t.Run("Circuit half-open state", func(t *testing.T) {
		middleware := circuitbreaker.NewCircuitBreakerMiddleware(3, 10*time.Second, 100*time.Millisecond)
		middleware.SetLogger(logger.NewBasicLogger())

		failingCtx := &clientContext.Context{
			Client: &http.Client{},
			Req:    httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			Next: func(*clientContext.Context) (*http.Response, error) {
				return nil, errors.New("simulated failure")
			},
		}

		// Fail 3 times to open the circuit
		for i := 0; i < 3; i++ {
			_, err := middleware.Process(failingCtx)
			require.Error(t, err)
		}

		// Wait for the circuit to enter half-open state
		time.Sleep(200 * time.Millisecond)

		successCtx := &clientContext.Context{
			Client: &http.Client{},
			Req:    httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			Next: func(*clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		// The circuit should now be half-open and allow one request
		resp, err := middleware.Process(successCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The circuit should now be closed and allow more requests
		resp, err = middleware.Process(successCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
