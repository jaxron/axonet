package cookie_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaxron/axonet/middleware/cookie"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookieMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Apply cookies to request", func(t *testing.T) {
		t.Parallel()

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

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// Multiple requests to check rotation
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			reqCookies := req.Cookies()
			assert.Len(t, reqCookies, 2)
			assert.Contains(t, []string{"123", "456"}, reqCookies[0].Value)
			assert.Contains(t, []string{"john", "jane"}, reqCookies[1].Value)
		}
	})

	t.Run("Cookie rotation", func(t *testing.T) {
		t.Parallel()

		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "123"}},
			{&http.Cookie{Name: "session", Value: "456"}},
			{&http.Cookie{Name: "session", Value: "789"}},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		usedValues := make(map[string]bool)
		for i := 0; i < 30; i++ {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)

			reqCookies := req.Cookies()
			assert.Len(t, reqCookies, 1)
			assert.Equal(t, "session", reqCookies[0].Name)
			usedValues[reqCookies[0].Value] = true
		}

		// All cookie values should have been used
		assert.Len(t, usedValues, 3)
		assert.True(t, usedValues["123"])
		assert.True(t, usedValues["456"])
		assert.True(t, usedValues["789"])
	})

	t.Run("Update cookies at runtime", func(t *testing.T) {
		t.Parallel()

		initialCookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "initial"}},
		}

		middleware := cookie.New(initialCookies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		// Initial request
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, "initial", req.Cookies()[0].Value)

		// Update cookies
		newCookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "updated1"}},
			{&http.Cookie{Name: "session", Value: "updated2"}},
		}
		middleware.UpdateCookies(newCookies)

		// Check that both new cookie sets are used
		usedValues := make(map[string]bool)
		for i := 0; i < 20; i++ {
			req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			_, err = middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)
			usedValues[req.Cookies()[0].Value] = true
		}
		assert.True(t, usedValues["updated1"])
		assert.True(t, usedValues["updated2"])
		assert.False(t, usedValues["initial"])
	})

	t.Run("GetCookieCount", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		middleware := cookie.New(nil)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		resp, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, req.Cookies())
	})

	t.Run("FIFO rotation", func(t *testing.T) {
		t.Parallel()

		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "1"}},
			{&http.Cookie{Name: "session", Value: "2"}},
			{&http.Cookie{Name: "session", Value: "3"}},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		expectedOrder := []string{"1", "2", "3", "1", "2", "3"}

		for i, expected := range expectedOrder {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
			require.NoError(t, err)

			reqCookies := req.Cookies()
			assert.Len(t, reqCookies, 1)
			assert.Equal(t, expected, reqCookies[0].Value, "Iteration %d", i)
		}
	})

	t.Run("Middleware enabled (default)", func(t *testing.T) {
		t.Parallel()

		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "123"}},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		_, err := middleware.Process(context.Background(), &http.Client{}, req, handler)
		require.NoError(t, err)

		reqCookies := req.Cookies()
		assert.Len(t, reqCookies, 1)
		assert.Equal(t, "session", reqCookies[0].Name)
		assert.Equal(t, "123", reqCookies[0].Value)
	})

	t.Run("Middleware disabled via context", func(t *testing.T) {
		t.Parallel()

		cookies := [][]*http.Cookie{
			{&http.Cookie{Name: "session", Value: "123"}},
		}

		middleware := cookie.New(cookies)
		middleware.SetLogger(logger.NewBasicLogger())

		handler := func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		ctx := context.WithValue(context.Background(), cookie.KeySkipCookie, true)
		_, err := middleware.Process(ctx, &http.Client{}, req, handler)
		require.NoError(t, err)

		reqCookies := req.Cookies()
		assert.Len(t, reqCookies, 0, "Expected no cookies when middleware is disabled")
	})
}
