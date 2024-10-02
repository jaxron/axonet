package cookie

import (
	"net/http"
	"sync"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// CookieMiddleware manages cookie rotation for HTTP requests.
type CookieMiddleware struct {
	cookies [][]*http.Cookie
	current int
	logger  logger.Logger
	mu      sync.RWMutex
}

// NewCookieMiddleware creates a new CookieMiddleware instance.
func NewCookieMiddleware(cookies [][]*http.Cookie) *CookieMiddleware {
	return &CookieMiddleware{
		cookies: cookies,
		current: 0,
		logger:  &logger.NoOpLogger{},
		mu:      sync.RWMutex{},
	}
}

// Process applies cookie logic before passing the request to the next middleware.
func (m *CookieMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.cookies) > 0 {
		// Get the next set of cookies to use
		cookies := m.cookies[m.current]
		m.current = (m.current + 1) % len(m.cookies)

		m.logger.WithFields(logger.Int("cookies", len(cookies))).Debug("Using Cookie")

		// Apply the cookies to the request
		for _, cookie := range cookies {
			ctx.Req.AddCookie(cookie)
		}
	}
	return ctx.Next(ctx)
}

// UpdateCookies updates the list of cookies at runtime.
func (m *CookieMiddleware) UpdateCookies(cookies [][]*http.Cookie) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Replace the existing cookie list with the new one
	m.cookies = cookies
	m.current = 0

	m.logger.WithFields(logger.Int("cookies", len(cookies))).Debug("Cookies updated")
}

// GetCookieCount returns the current number of cookies in the list.
func (m *CookieMiddleware) GetCookieCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.cookies)
}

// SetLogger sets the logger for the middleware.
func (m *CookieMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
