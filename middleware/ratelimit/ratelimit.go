package ratelimit

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
	"golang.org/x/time/rate"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// RateLimiterMiddleware implements a rate limiting middleware for HTTP requests.
type RateLimiterMiddleware struct {
	limiter *rate.Limiter
	logger  logger.Logger
}

// New creates a new RateLimiterMiddleware instance.
func New(requestsPerSecond float64, burst int) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies rate limiting before passing the request to the next middleware.
func (m *RateLimiterMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	m.logger.Debug("Processing request with rate limiter middleware")

	// Wait for rate limiter permission
	if err := m.limiter.Wait(ctx.Req.Context()); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	// Execute the next middleware in the chain
	return ctx.Next(ctx)
}

// SetLogger sets the logger for the middleware.
func (m *RateLimiterMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
