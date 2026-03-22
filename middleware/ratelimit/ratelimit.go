package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jaxron/axonet/pkg/client/errs"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
	"golang.org/x/time/rate"
)

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

// Process waits until the rate limit allows the request to proceed.
func (m *RateLimiterMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Wait for rate limiter permission
	if err := m.limiter.Wait(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errs.ErrTimeout
		}
		if strings.Contains(err.Error(), "would exceed context deadline") {
			return nil, errs.ErrTimeout
		}
		return nil, err
	}

	// Execute the next middleware in the chain
	return next(ctx, httpClient, req)
}

// SetLogger sets the logger for the middleware.
func (m *RateLimiterMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
