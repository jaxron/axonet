package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jaxron/axonet/middleware/proxy"
	clientContext "github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyMiddleware(t *testing.T) {
	t.Run("Apply proxy to request", func(t *testing.T) {
		proxy1, _ := url.Parse("http://proxy1.example.com")
		proxy2, _ := url.Parse("http://proxy2.example.com")
		proxies := []*url.URL{proxy1, proxy2}

		middleware := proxy.New(proxies)
		middleware.SetLogger(logger.NewBasicLogger())

		transport := &http.Transport{}
		client := &http.Client{Transport: transport}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: client,
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		// First request
		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assertProxyUsed(t, transport, req, proxy1)

		// Second request
		resp, err = middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assertProxyUsed(t, transport, req, proxy2)

		// Third request (should rotate back to first proxy)
		resp, err = middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assertProxyUsed(t, transport, req, proxy1)
	})

	t.Run("Update proxies at runtime", func(t *testing.T) {
		initialProxy, _ := url.Parse("http://initial.example.com")
		middleware := proxy.New([]*url.URL{initialProxy})
		middleware.SetLogger(logger.NewBasicLogger())

		transport := &http.Transport{}
		client := &http.Client{Transport: transport}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: client,
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		// Initial request
		_, err := middleware.Process(ctx)
		require.NoError(t, err)
		assertProxyUsed(t, transport, req, initialProxy)

		// Update proxies
		newProxy, _ := url.Parse("http://new.example.com")
		middleware.UpdateProxies([]*url.URL{newProxy})

		// Next request should use the new proxy
		_, err = middleware.Process(ctx)
		require.NoError(t, err)
		assertProxyUsed(t, transport, req, newProxy)
	})

	t.Run("GetProxyCount", func(t *testing.T) {
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
		middleware := proxy.New(nil)
		middleware.SetLogger(logger.NewBasicLogger())

		transport := &http.Transport{}
		client := &http.Client{Transport: transport}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: client,
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assertNoProxyUsed(t, transport, req)
	})

	t.Run("Invalid transport", func(t *testing.T) {
		proxyURL, _ := url.Parse("http://proxy.example.com")
		middleware := proxy.New([]*url.URL{proxyURL})
		middleware.SetLogger(logger.NewBasicLogger())

		client := &http.Client{Transport: nil}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: client,
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		_, err := middleware.Process(ctx)
		assert.Error(t, err)
		assert.Equal(t, proxy.ErrInvalidTransport, err)
	})
}

func assertProxyUsed(t *testing.T, transport *http.Transport, req *http.Request, expectedProxy *url.URL) {
	t.Helper()
	if transport.Proxy == nil {
		t.Fatal("Expected transport.Proxy to be set, but it was nil")
	}
	proxyURL, err := transport.Proxy(req)
	require.NoError(t, err)
	assert.Equal(t, expectedProxy, proxyURL)
}

func assertNoProxyUsed(t *testing.T, transport *http.Transport, req *http.Request) {
	t.Helper()
	if transport.Proxy != nil {
		proxyURL, err := transport.Proxy(req)
		require.NoError(t, err)
		assert.Nil(t, proxyURL, "Expected no proxy to be used, but got: %v", proxyURL)
	}
}
