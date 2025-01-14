package singleflight

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cespare/xxhash"
	clientErrors "github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
	"golang.org/x/sync/singleflight"
)

var (
	ErrKeyGeneration = errors.New("failed to generate request key")
	ErrHashMethod    = errors.New("failed to hash method and URL")
	ErrHashHeader    = errors.New("failed to hash header")
	ErrReadBody      = errors.New("failed to read request body")
	ErrHashBody      = errors.New("failed to hash body")
)

// SingleFlightMiddleware implements the singleflight pattern to deduplicate concurrent identical requests.
type SingleFlightMiddleware struct {
	sfGroup *singleflight.Group
	logger  logger.Logger
}

// New creates a new SingleFlightMiddleware instance.
func New() *SingleFlightMiddleware {
	return &SingleFlightMiddleware{
		sfGroup: &singleflight.Group{},
		logger:  &logger.NoOpLogger{},
	}
}

// Process applies the singleflight pattern before passing the request to the next middleware.
func (m *SingleFlightMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Generate a unique key for the request
	key, err := m.generateRequestKey(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyGeneration, err)
	}

	// Use singleflight to execute the request
	result, err, _ := m.sfGroup.Do(key, func() (interface{}, error) {
		return next(ctx, httpClient, req)
	})

	// Type assertion to get the response
	resp, ok := result.(*http.Response)
	if !ok {
		return nil, clientErrors.ErrUnreachable
	}

	// Note: we let the user handle response
	return resp, err
}

// generateRequestKey generates a unique key for the request based on the method, URL, headers, and body.
func (m *SingleFlightMiddleware) generateRequestKey(req *http.Request) (string, error) {
	h := xxhash.New()

	// Helper function to write to hash and handle errors
	writeToHash := func(data []byte, errType error) error {
		if _, err := h.Write(data); err != nil {
			return fmt.Errorf("%w: %w", errType, err)
		}
		return nil
	}

	// Hash method and URL
	if err := writeToHash([]byte(req.Method+req.URL.String()), ErrHashMethod); err != nil {
		return "", fmt.Errorf("%w: %w", ErrKeyGeneration, err)
	}

	// Hash headers (excluding Authorization)
	for key, values := range req.Header {
		if key != "Authorization" {
			if err := writeToHash([]byte(key+fmt.Sprint(values)), ErrHashHeader); err != nil {
				return "", fmt.Errorf("%w: %w", ErrKeyGeneration, err)
			}
		}
	}

	// Hash body if it exists
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrKeyGeneration, err)
		}
		if err := writeToHash(body, ErrHashBody); err != nil {
			return "", fmt.Errorf("%w: %w", ErrKeyGeneration, err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return strconv.FormatUint(h.Sum64(), 16), nil
}

// SetLogger sets the logger for the middleware.
func (m *SingleFlightMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
