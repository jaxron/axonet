package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	clientErrors "github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

var ErrRetryFailed = errors.New("retry failed")

// RetryMiddleware implements retry logic for HTTP requests with exponential backoff.
type RetryMiddleware struct {
	maxAttempts     uint64
	initialInterval time.Duration
	maxInterval     time.Duration
	logger          logger.Logger
}

// New creates a new RetryMiddleware instance.
func New(maxAttempts uint64, initialInterval, maxInterval time.Duration) *RetryMiddleware {
	return &RetryMiddleware{
		maxAttempts:     maxAttempts,
		initialInterval: initialInterval,
		maxInterval:     maxInterval,
		logger:          &logger.NoOpLogger{},
	}
}

// Process applies retry logic before passing the request to the next middleware.
func (m *RetryMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	m.logger.Debug("Processing request with retry middleware")

	// Create an exponential backoff strategy with a maximum number of retries
	expBackoff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(m.initialInterval),
		backoff.WithMaxInterval(m.maxInterval),
	), m.maxAttempts)
	backoffStrategy := backoff.WithContext(expBackoff, ctx)

	var resp *http.Response
	var err error

	// Retry the request using the backoff strategy
	retryErr := backoff.RetryNotify(
		func() error {
			resp, err = next(ctx, httpClient, req)
			return m.handleRetryError(err)
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			m.logger.WithFields(
				logger.String("error", err.Error()),
				logger.Duration("retry_in", duration),
			).Warn("Retrying request")
		},
	)
	if retryErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrRetryFailed, retryErr)
	}

	return resp, nil
}

// handleRetryError determines whether to retry the request based on the error type.
func (m *RetryMiddleware) handleRetryError(err error) error {
	if err != nil {
		if clientErrors.IsTemporary(err) {
			return err // This will trigger a retry for temporary errors
		}
		return backoff.Permanent(err) // This will stop retries for permanent errors
	}
	return nil // Success, stop retrying
}

// SetLogger sets the logger for the middleware.
func (m *RetryMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
