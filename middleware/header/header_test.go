package header_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaxron/axonet/middleware/header"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Apply headers to request", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{
			"User-Agent": []string{"TestAgent/1.0"},
			"X-Custom":   []string{"Value1", "Value2"},
		}

		middleware := header.New(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, "TestAgent/1.0", req.Header.Get("User-Agent"))
			assert.Equal(t, []string{"Value1", "Value2"}, req.Header["X-Custom"])
			return &http.Response{StatusCode: http.StatusOK}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Append to existing headers", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{
			"X-Existing": []string{"NewValue"},
		}

		middleware := header.New(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.Header.Set("X-Existing", "OriginalValue")

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, []string{"OriginalValue", "NewValue"}, req.Header["X-Existing"])
			return &http.Response{StatusCode: http.StatusOK}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Empty headers", func(t *testing.T) {
		t.Parallel()

		middleware := header.New(http.Header{})
		middleware.SetLogger(logger.NewBasicLogger())

		originalReq := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		resp, err := middleware.Process(context.Background(), &http.Client{}, originalReq, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Len(t, originalReq.Header, len(req.Header))
			return &http.Response{StatusCode: http.StatusOK}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Multiple values for same header", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{
			"X-Multi": []string{"Value1", "Value2", "Value3"},
		}

		middleware := header.New(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, []string{"Value1", "Value2", "Value3"}, req.Header["X-Multi"])
			return &http.Response{StatusCode: http.StatusOK}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
