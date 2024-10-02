package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaxron/axonet/middleware/cookie"
	clientContext "github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookieMiddleware(t *testing.T) {
	t.Run("Apply cookies to request", func(t *testing.T) {
		cookies := [][]*http.Cookie{
			{
				&http.Cookie{Name: "session", Value: "123"},
				&http.Cookie{Name: "user", Value: "john"},
			},
			{
				&http.Cookie{Name: "session", Value: "456"},
				&http.Cookie{Name: "user", Value: "jane"},
			},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		// First request
		resp, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		reqCookies := req.Cookies()
		assert.Len(t, reqCookies, 2)
		assert.Equal(t, "session", reqCookies[0].Name)
		assert.Equal(t, "123", reqCookies[0].Value)
		assert.Equal(t, "user", reqCookies[1].Name)
		assert.Equal(t, "john", reqCookies[1].Value)

		// Second request (should use the next set of cookies)
		req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx.Req = req
		resp, err = middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		reqCookies = req.Cookies()
		assert.Len(t, reqCookies, 2)
		assert.Equal(t, "session", reqCookies[0].Name)
		assert.Equal(t, "456", reqCookies[0].Value)
		assert.Equal(t, "user", reqCookies[1].Name)
		assert.Equal(t, "jane", reqCookies[1].Value)
	})

	t.Run("Cookie rotation", func(t *testing.T) {
		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "123"}},
			{&http.Cookie{Name: "session", Value: "456"}},
			{&http.Cookie{Name: "session", Value: "789"}},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			ctx := &clientContext.Context{
				Client: &http.Client{},
				Req:    req,
				Next: func(ctx *clientContext.Context) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK}, nil
				},
			}

			_, err := middleware.Process(ctx)
			require.NoError(t, err)

			reqCookies := req.Cookies()
			assert.Len(t, reqCookies, 1)
			assert.Equal(t, "session", reqCookies[0].Name)
			assert.Equal(t, cookies[i%3][0].Value, reqCookies[0].Value)
		}
	})

	t.Run("Update cookies at runtime", func(t *testing.T) {
		initialCookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "initial"}},
		}

		middleware := cookie.New(initialCookies)
		middleware.SetLogger(logger.NewBasicLogger())

		// Initial request
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := &clientContext.Context{
			Client: &http.Client{},
			Req:    req,
			Next: func(ctx *clientContext.Context) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		_, err := middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, "initial", req.Cookies()[0].Value)

		// Update cookies
		newCookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "updated"}},
		}
		middleware.UpdateCookies(newCookies)

		// Next request should use updated cookies
		req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx.Req = req
		_, err = middleware.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, "updated", req.Cookies()[0].Value)
	})

	t.Run("GetCookieCount", func(t *testing.T) {
		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "123"}},
			{&http.Cookie{Name: "session", Value: "456"}},
		}

		middleware := cookie.New(cookies)
		assert.Equal(t, 2, middleware.GetCookieCount())

		newCookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "789"}},
			{&http.Cookie{Name: "session", Value: "012"}},
			{&http.Cookie{Name: "session", Value: "345"}},
		}
		middleware.UpdateCookies(newCookies)
		assert.Equal(t, 3, middleware.GetCookieCount())
	})

	t.Run("No cookies", func(t *testing.T) {
		middleware := cookie.New(nil)
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
		assert.Empty(t, req.Cookies())
	})
}
