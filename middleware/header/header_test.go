package header_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaxron/axonet/middleware/header"
	clientContext "github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderMiddleware(t *testing.T) {
	t.Run("Apply headers to request", func(t *testing.T) {
		headers := http.Header{
			"User-Agent": []string{"TestAgent/1.0"},
			"X-Custom":   []string{"Value1", "Value2"},
		}

		middleware := header.NewHeaderMiddleware(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, "TestAgent/1.0", req.Header.Get("User-Agent"))
		assert.Equal(t, []string{"Value1", "Value2"}, req.Header["X-Custom"])
	})

	t.Run("Append to existing headers", func(t *testing.T) {
		headers := http.Header{
			"X-Existing": []string{"NewValue"},
		}

		middleware := header.NewHeaderMiddleware(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.Header.Set("X-Existing", "OriginalValue")
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, []string{"OriginalValue", "NewValue"}, req.Header["X-Existing"])
	})

	t.Run("Empty headers", func(t *testing.T) {
		middleware := header.NewHeaderMiddleware(http.Header{})
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		originalHeaderLen := len(req.Header)
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, originalHeaderLen, len(req.Header))
	})

	t.Run("Multiple values for same header", func(t *testing.T) {
		headers := http.Header{
			"X-Multi": []string{"Value1", "Value2", "Value3"},
		}

		middleware := header.NewHeaderMiddleware(headers)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, []string{"Value1", "Value2", "Value3"}, req.Header["X-Multi"])
	})
}
