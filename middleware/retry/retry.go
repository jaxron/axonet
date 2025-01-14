package retry

import (
	"context"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	clientErrors "github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

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
	// Create an exponential backoff strategy with a maximum number of retries
	expBackoff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(m.initialInterval),
		backoff.WithMaxInterval(m.maxInterval),
	), m.maxAttempts)
	backoffStrategy := backoff.WithContext(expBackoff, ctx)

	var resp *http.Response

	// Retry the request using the backoff strategy
	err := backoff.RetryNotify(
		func() error {
			var err error
			resp, err = next(ctx, httpClient, req)
			return m.handleRetryError(resp, err)
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			m.logger.WithFields(
				logger.String("error", err.Error()),
				logger.String("url", req.URL.String()),
				logger.Duration("retry_in", duration),
			).Warn("Retrying request")
		},
	)

	// Note: we let the user handle response
	return resp, err
}

// handleRetryError determines whether to retry the request based on the status code and error type.
func (m *RetryMiddleware) handleRetryError(resp *http.Response, err error) error {
	if resp != nil {
		switch {
		case resp.StatusCode >= 500:
			// Server errors are typically temporary
			return clientErrors.ErrBadStatus
		case resp.StatusCode == http.StatusTooManyRequests:
			// Too Many Requests - should be retried
			return clientErrors.ErrBadStatus
		case resp.StatusCode >= 400:
			// Client errors are typically permanent
			return backoff.Permanent(clientErrors.ErrBadStatus)
		}
	}

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
