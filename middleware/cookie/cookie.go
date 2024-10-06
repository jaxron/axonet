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
	mu     sync.RWMutex
	state  *cookieState
	logger logger.Logger
}

type cookieState struct {
	cookies [][]*http.Cookie
}

// New creates a new CookieMiddleware instance.
func New(cookies [][]*http.Cookie) *CookieMiddleware {
	return &CookieMiddleware{
		mu:     sync.RWMutex{},
		state:  &cookieState{cookies: cookies},
		logger: &logger.NoOpLogger{},
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
	cookiesLen := len(m.state.cookies)
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

	if len(m.state.cookies) == 0 {
		return nil
	}

	cookieSet := m.state.cookies[0]
	m.state.cookies = append(m.state.cookies[1:], cookieSet)

	return cookieSet
}

// UpdateCookies updates the list of cookies at runtime.
func (m *CookieMiddleware) UpdateCookies(cookies [][]*http.Cookie) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.cookies = cookies

	m.logger.WithFields(logger.Int("cookie_sets", len(cookies))).Debug("Cookies updated")
}

// GetCookieCount returns the current number of cookie sets in the list.
func (m *CookieMiddleware) GetCookieCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.state.cookies)
}

// SetLogger sets the logger for the middleware.
func (m *CookieMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
