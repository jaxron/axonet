package cookie

import (
	"net/http"
	"sync/atomic"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// CookieMiddleware manages cookie rotation for HTTP requests.
type CookieMiddleware struct {
	cookies atomic.Value
	current atomic.Uint64
	logger  logger.Logger
}

type cookieState struct {
	cookies [][]*http.Cookie
}

// New creates a new CookieMiddleware instance.
func New(cookies [][]*http.Cookie) *CookieMiddleware {
	m := &CookieMiddleware{
		cookies: atomic.Value{},
		current: atomic.Uint64{},
		logger:  &logger.NoOpLogger{},
	}
	m.cookies.Store(&cookieState{cookies: cookies})
	return m
}

// Process applies cookie logic before passing the request to the next middleware.
func (m *CookieMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	state := m.cookies.Load().(*cookieState)
	cookiesLen := len(state.cookies)

	if cookiesLen > 0 {
		current := m.current.Add(1) - 1            // Subtract 1 to start from 0
		index := int(current % uint64(cookiesLen)) // #nosec G115
		cookies := state.cookies[index]

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
	newState := &cookieState{cookies: cookies}
	m.cookies.Store(newState)
	m.current.Store(0)

	m.logger.WithFields(logger.Int("cookies", len(cookies))).Debug("Cookies updated")
}

// GetCookieCount returns the current number of cookies in the list.
func (m *CookieMiddleware) GetCookieCount() int {
	state := m.cookies.Load().(*cookieState)
	return len(state.cookies)
}

// SetLogger sets the logger for the middleware.
func (m *CookieMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
