package cookie

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

type contextKey int

const (
	KeySkipCookie contextKey = iota
)

// CookieMiddleware manages cookie rotation for HTTP requests.
type CookieMiddleware struct {
	cookies     [][]*http.Cookie
	cookieCount int
	current     atomic.Uint64
	mu          sync.RWMutex
	logger      logger.Logger
}

// New creates a new CookieMiddleware instance.
func New(cookies [][]*http.Cookie) *CookieMiddleware {
	m := &CookieMiddleware{
		cookies:     cookies,
		cookieCount: len(cookies),
		current:     atomic.Uint64{},
		mu:          sync.RWMutex{},
		logger:      &logger.NoOpLogger{},
	}
	m.current.Store(0)
	return m
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cookieCount == 0 {
		return nil
	}

	current := m.current.Add(1) - 1
	index := current % uint64(m.cookieCount) // #nosec G115
	return m.cookies[index]
}

// UpdateCookies updates the list of cookies at runtime.
func (m *CookieMiddleware) UpdateCookies(cookies [][]*http.Cookie) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cookies = cookies
	m.cookieCount = len(cookies)
	m.current.Store(0)

	m.logger.WithFields(logger.Int("cookie_sets", len(cookies))).Debug("Cookies updated")
}

// GetCookieCount returns the current number of cookie sets in the list.
func (m *CookieMiddleware) GetCookieCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cookieCount
}

// SetLogger sets the logger for the middleware.
func (m *CookieMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
