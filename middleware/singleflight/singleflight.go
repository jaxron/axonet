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
	"github.com/jaxron/axonet/pkg/client/errs"
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

// sfResponse holds response data that can be safely shared across goroutines.
type sfResponse struct {
	resp      *http.Response
	bodyBytes []byte
}

// New creates a new SingleFlightMiddleware instance.
func New() *SingleFlightMiddleware {
	return &SingleFlightMiddleware{
		sfGroup: &singleflight.Group{},
		logger:  &logger.NoOpLogger{},
	}
}

// Process deduplicates concurrent identical requests.
func (m *SingleFlightMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Generate a unique key for the request
	key, err := m.generateRequestKey(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyGeneration, err)
	}

	// Use singleflight to execute the request
	result, err, _ := m.sfGroup.Do(key, func() (interface{}, error) {
		resp, err := next(ctx, httpClient, req)
		if err != nil {
			return nil, err
		}

		// Buffer the body so it can be cloned for each caller
		var bodyBytes []byte
		if resp.Body != nil {
			var readErr error
			bodyBytes, readErr = io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrReadBody, readErr)
			}
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		return &sfResponse{resp: resp, bodyBytes: bodyBytes}, nil
	})
	if err != nil {
		return nil, err
	}

	// Type assertion to get the response
	sfResp, ok := result.(*sfResponse)
	if !ok {
		return nil, errs.ErrUnreachable
	}

	// Clone response with a fresh body reader for this caller
	cloned := *sfResp.resp
	cloned.Body = io.NopCloser(bytes.NewReader(sfResp.bodyBytes))
	return &cloned, nil
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

	// Hash headers
	for key, values := range req.Header {
		if key != "Authorization" && key != "Cookie" {
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
