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

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport, ok := httpClient.Transport.(*http.Transport)
			require.True(t, ok)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			assert.Contains(t, []string{proxy1.String(), proxy2.String()}, proxyURL.String())
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// Multiple requests to check rotation
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("FIFO rotation", func(t *testing.T) {
		t.Parallel()

		proxy1, _ := url.Parse("http://proxy1.example.com")
		proxy2, _ := url.Parse("http://proxy2.example.com")
		proxy3, _ := url.Parse("http://proxy3.example.com")
		proxies := []*url.URL{proxy1, proxy2, proxy3}

		middleware := proxy.New(proxies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport, ok := httpClient.Transport.(*http.Transport)
			require.True(t, ok)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Proxy-Used": []string{proxyURL.String()}}}, nil
		}

		expectedOrder := []string{
			proxy1.String(), proxy2.String(), proxy3.String(),
			proxy1.String(), proxy2.String(), proxy3.String(),
		}

		for i, expected := range expectedOrder {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)
			assert.Equal(t, expected, resp.Header.Get("Proxy-Used"), "Iteration %d", i)
		}
	})

	t.Run("Update proxies at runtime", func(t *testing.T) {
		t.Parallel()

		initialProxy, _ := url.Parse("http://initial.example.com")
		middleware := proxy.New([]*url.URL{initialProxy})
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			transport, ok := httpClient.Transport.(*http.Transport)
			require.True(t, ok)
			assert.NotNil(t, transport.Proxy)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Proxy-Used": []string{proxyURL.String()}}}, nil
		}

		// Initial request
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, initialProxy.String(), resp.Header.Get("Proxy-Used"))

		// Update proxies
		newProxy1, _ := url.Parse("http://new1.example.com")
		newProxy2, _ := url.Parse("http://new2.example.com")
		middleware.UpdateProxies([]*url.URL{newProxy1, newProxy2})

		// Check that both new proxies are used
		usedProxies := make(map[string]bool)
		for i := 0; i < 20; i++ {
			req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err = middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)
			usedProxies[resp.Header.Get("Proxy-Used")] = true
		}
		assert.True(t, usedProxies[newProxy1.String()])
		assert.True(t, usedProxies[newProxy2.String()])
		assert.False(t, usedProxies[initialProxy.String()])
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

		originalClient := &http.Client{}
		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, originalClient, httpClient)
			if transport, ok := httpClient.Transport.(*http.Transport); ok {
				assert.Nil(t, transport.Proxy)
			}
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		resp, err := middleware.Process(context.Background(), originalClient, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Disable proxy", func(t *testing.T) {
		t.Parallel()

		proxyURL, _ := url.Parse("http://proxy.example.com")
		proxies := []*url.URL{proxyURL}

		middleware := proxy.New(proxies)
		middleware.SetLogger(logger.NewBasicLogger())

		t.Run("Middleware enabled (default)", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

			handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
				transport, ok := httpClient.Transport.(*http.Transport)
				require.True(t, ok)
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

			originalClient := &http.Client{}
			handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
				// When proxy is disabled, the client should remain unchanged
				assert.Equal(t, originalClient, httpClient)

				// If Transport is not nil, ensure it doesn't have a Proxy set
				if transport, ok := httpClient.Transport.(*http.Transport); ok {
					assert.Nil(t, transport.Proxy)
				}

				return &http.Response{StatusCode: http.StatusOK}, nil
			}

			_, err := middleware.Process(ctx, originalClient, req, handler)
			require.NoError(t, err)
		})
	})
}
