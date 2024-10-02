package header

import (
	"net/http"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// HeaderMiddleware adds headers to HTTP requests.
type HeaderMiddleware struct {
	headers http.Header
	logger  logger.Logger
}

// NewHeaderMiddleware creates a new HeaderMiddleware instance.
func NewHeaderMiddleware(headers http.Header) *HeaderMiddleware {
	return &HeaderMiddleware{
		headers: headers,
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies headers to the request before passing it to the next middleware.
func (m *HeaderMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	for key, values := range m.headers {
		for _, value := range values {
			ctx.Req.Header.Add(key, value)
		}
	}
	return ctx.Next(ctx)
}

// SetLogger sets the logger for the middleware.
func (m *HeaderMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
