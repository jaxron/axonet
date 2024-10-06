package cookie

import (
	"context"
	"net/http"
	"sync"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

type contextKey int

const (
	KeySkipCookie contextKey = iota
)

// CookieMiddleware manages cookie rotation for HTTP requests.
type CookieMiddleware struct {
	mu      sync.RWMutex
	cookies [][]*http.Cookie
	logger  logger.Logger
}

// New creates a new CookieMiddleware instance.
func New(cookies [][]*http.Cookie) *CookieMiddleware {
	return &CookieMiddleware{
		mu:      sync.RWMutex{},
		cookies: cookies,
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies cookie logic before passing the request to the next middleware.
func (m *CookieMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Check if the cookie middleware is disabled via context
	if isDisabled, ok := ctx.Value(KeySkipCookie).(bool); ok && isDisabled {
		m.logger.Debug("Cookie middleware disabled via context")
		return next(ctx, httpClient, req)
	}

	m.logger.Debug("Processing request with cookie middleware")

	m.mu.RLock()
	cookiesLen := len(m.cookies)
	m.mu.RUnlock()

	if cookiesLen > 0 {
		cookies := m.selectCookieSet()

		m.logger.WithFields(logger.Int("cookies", len(cookies))).Debug("Using Cookie Set")

		// Apply the cookies to the request
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	return next(ctx, httpClient, req)
}

// selectCookieSet chooses the next cookie set to use.
func (m *CookieMiddleware) selectCookieSet() []*http.Cookie {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.cookies) == 0 {
		return nil
	}

	cookieSet := m.cookies[0]
	m.cookies = append(m.cookies[1:], cookieSet)

	return cookieSet
}

// UpdateCookies updates the list of cookies at runtime.
func (m *CookieMiddleware) UpdateCookies(cookies [][]*http.Cookie) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cookies = cookies

	m.logger.WithFields(logger.Int("cookie_sets", len(cookies))).Debug("Cookies updated")
}

// GetCookieCount returns the current number of cookie sets in the list.
func (m *CookieMiddleware) GetCookieCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.cookies)
}

// SetLogger sets the logger for the middleware.
func (m *CookieMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
