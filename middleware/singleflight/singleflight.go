package singleflight

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cespare/xxhash"
	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"golang.org/x/sync/singleflight"
)

// SingleFlightMiddleware implements the singleflight pattern to deduplicate concurrent identical requests.
type SingleFlightMiddleware struct {
	sfGroup *singleflight.Group
	logger  logger.Logger
}

// NewSingleFlightMiddleware creates a new SingleFlightMiddleware instance.
func NewSingleFlightMiddleware() *SingleFlightMiddleware {
	return &SingleFlightMiddleware{
		sfGroup: &singleflight.Group{},
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies the singleflight pattern before passing the request to the next middleware.
func (m *SingleFlightMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	m.logger.Debug("Processing request with singleflight middleware")

	// Generate a unique key for the request
	key, err := m.generateRequestKey(ctx.Req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrSingleFlight, err)
	}

	// Use singleflight to execute the request
	result, err, _ := m.sfGroup.Do(key, func() (interface{}, error) {
		return ctx.Next(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrSingleFlight, err)
	}

	// Type assertion to get the response
	resp, ok := result.(*http.Response)
	if !ok {
		return nil, errors.ErrUnreachable
	}

	return resp, nil
}

// generateRequestKey generates a unique key for the request based on the method, URL, headers, and body.
func (m *SingleFlightMiddleware) generateRequestKey(req *http.Request) (string, error) {
	h := xxhash.New()

	// Helper function to write to hash and check error
	writeHash := func(s string) error {
		_, err := io.WriteString(h, s)
		return err
	}

	// Hash method and URL
	if err := writeHash(req.Method + req.URL.String()); err != nil {
		return "", fmt.Errorf("failed to hash method and URL: %w", err)
	}

	// Hash headers (excluding Authorization)
	for key, values := range req.Header {
		if key != "Authorization" {
			if err := writeHash(key + fmt.Sprint(values)); err != nil {
				return "", fmt.Errorf("failed to hash header: %w", err)
			}
		}
	}

	// Hash body if it exists
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read request body: %w", err)
		}
		if _, err := h.Write(body); err != nil {
			return "", fmt.Errorf("failed to hash body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return strconv.FormatUint(h.Sum64(), 16), nil
}

// SetLogger sets the logger for the middleware.
func (m *SingleFlightMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
