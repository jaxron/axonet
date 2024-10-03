package proxy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jaxron/axonet/middleware/proxy"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Apply proxy to request", func(t *testing.T) {
		t.Parallel()

		proxy1, _ := url.Parse("http://proxy1.example.com")
		proxy2, _ := url.Parse("http://proxy2.example.com")
		proxies := []*url.URL{proxy1, proxy2}

		middleware := proxy.New(proxies)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			assert.Contains(t, []string{proxy1.String(), proxy2.String()}, proxyURL.String())
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// First request
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Second request
		resp, err = middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update proxies at runtime", func(t *testing.T) {
		t.Parallel()

		initialProxy, _ := url.Parse("http://initial.example.com")
		middleware := proxy.New([]*url.URL{initialProxy})
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			assert.Equal(t, initialProxy.String(), proxyURL.String())
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// Initial request
		_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)

		// Update proxies
		newProxy, _ := url.Parse("http://new.example.com")
		middleware.UpdateProxies([]*url.URL{newProxy})

		// Next request should use the new proxy
		newHandler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			assert.Equal(t, newProxy.String(), proxyURL.String())
			return &http.Response{StatusCode: http.StatusOK}, nil
		}
		_, err = middleware.Process(context.Background(), &http.Client{}, req, newHandler)
		require.NoError(t, err)
	})

	t.Run("GetProxyCount", func(t *testing.T) {
		t.Parallel()

		proxy1, _ := url.Parse("http://proxy1.example.com")
		proxy2, _ := url.Parse("http://proxy2.example.com")
		proxies := []*url.URL{proxy1, proxy2}

		middleware := proxy.New(proxies)
		assert.Equal(t, 2, middleware.GetProxyCount())

		newProxy, _ := url.Parse("http://new.example.com")
		middleware.UpdateProxies([]*url.URL{newProxy})
		assert.Equal(t, 1, middleware.GetProxyCount())
	})

	t.Run("No proxies", func(t *testing.T) {
		t.Parallel()

		middleware := proxy.New(nil)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			_, ok := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.False(t, ok, "Expected no transport to be set when no proxies are configured")
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestProxyMiddlewareDisable(t *testing.T) {
	t.Parallel()

	proxyURL, _ := url.Parse("http://proxy.example.com")
	proxies := []*url.URL{proxyURL}

	middleware := proxy.New(proxies)
	middleware.SetLogger(logger.NewBasicLogger())

	t.Run("Middleware enabled (default)", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.NotNil(t, transport.Proxy)
			actualProxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			assert.Equal(t, "http://proxy.example.com", actualProxyURL.String())
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
	})

	t.Run("Middleware disabled via context", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := context.WithValue(context.Background(), proxy.KeySkipProxy, true)

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			_, ok := ctx.Value(http.DefaultTransport).(*http.Transport)
			assert.False(t, ok, "Expected no transport to be set when proxy is disabled")
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		_, err := middleware.Process(ctx, &http.Client{}, req, handler)
		require.NoError(t, err)
	})
}
