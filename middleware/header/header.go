package header

import (
	"context"
	"net/http"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

// HeaderMiddleware adds headers to HTTP requests.
type HeaderMiddleware struct {
	headers http.Header
	logger  logger.Logger
}

// New creates a new HeaderMiddleware instance.
func New(headers http.Header) *HeaderMiddleware {
	return &HeaderMiddleware{
		headers: headers,
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies headers to the request before passing it to the next middleware.
func (m *HeaderMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	m.logger.Debug("Processing request with header middleware")

	for key, values := range m.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return next(ctx, httpClient, req)
}

// SetLogger sets the logger for the middleware.
func (m *HeaderMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
