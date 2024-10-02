package circuitbreaker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/sony/gobreaker"
)

// CircuitBreakerMiddleware implements the circuit breaker pattern to prevent cascading failures.
type CircuitBreakerMiddleware struct {
	breaker *gobreaker.CircuitBreaker
	logger  logger.Logger
}

// New creates a new CircuitBreakerMiddleware instance.
func New(maxRequests uint32, interval, timeout time.Duration) *CircuitBreakerMiddleware {
	middleware := &CircuitBreakerMiddleware{
		breaker: nil,
		logger:  &logger.NoOpLogger{},
	}

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "HTTPCircuitBreaker",
		MaxRequests: maxRequests,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			middleware.logger.WithFields(
				logger.String("name", name),
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			).Warn("Circuit breaker state changed")
		},
		IsSuccessful: nil,
	})
	middleware.breaker = breaker

	return middleware
}

// Process applies the circuit breaker before passing the request to the next middleware.
func (m *CircuitBreakerMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	m.logger.Debug("Processing request with circuit breaker middleware")

	// Execute the request with the circuit breaker
	result, err := m.breaker.Execute(func() (interface{}, error) {
		return ctx.Next(ctx)
	})
	if err != nil {
		switch err {
		case gobreaker.ErrOpenState:
			return nil, fmt.Errorf("%w: %w", errors.ErrCircuitOpen, err)
		case gobreaker.ErrTooManyRequests:
			return nil, fmt.Errorf("%w: %w", errors.ErrCircuitExhausted, err)
		default:
			return nil, err
		}
	}

	// Type assertion to get the response
	resp, ok := result.(*http.Response)
	if !ok {
		return nil, errors.ErrUnreachable
	}

	return resp, nil
}

// SetLogger sets the logger for the middleware.
func (m *CircuitBreakerMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
