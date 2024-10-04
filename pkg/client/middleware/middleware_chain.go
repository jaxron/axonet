package middleware

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// Chain represents a chain of middleware.
type Chain struct {
	middlewares []Middleware
	logger      logger.Logger
}

// NewChain creates a new middleware chain.
func NewChain(logger logger.Logger, middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
		logger:      logger,
	}
}

// Len returns the number of middlewares in the chain.
func (c *Chain) Len() int {
	return len(c.middlewares)
}

// Middlewares returns the slice of middlewares.
func (c *Chain) Middlewares() []Middleware {
	return c.middlewares
}

// Then adds middleware to the chain, replacing any existing middleware of the same type.
func (c *Chain) Then(middlewares ...Middleware) {
	seen := make(map[reflect.Type]Middleware)

	// Add existing middlewares to the map
	for _, m := range c.middlewares {
		seen[reflect.TypeOf(m)] = m
	}

	// Add or replace middlewares
	for _, m := range middlewares {
		seen[reflect.TypeOf(m)] = m
		m.SetLogger(c.logger)
	}

	// Create a new slice with unique middlewares
	c.middlewares = make([]Middleware, 0, len(seen))
	for _, m := range seen {
		c.middlewares = append(c.middlewares, m)
	}
}

// Process runs the request through all middleware in the chain.
func (c *Chain) Process(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
	// If no middlewares are defined, perform the request immediately
	if len(c.middlewares) == 0 {
		return c.performRequest(ctx, httpClient, req)
	}

	c.logMiddlewareChain()
	return c.processMiddleware(ctx, httpClient, req, 0)
}

// logMiddlewareChain logs the available middleware in the chain.
func (c *Chain) logMiddlewareChain() {
	for i, m := range c.middlewares {
		c.logger.WithFields(
			logger.Int("index", i),
			logger.String("type", reflect.TypeOf(m).String()),
		).Debug("Middleware in chain")
	}
}

// processMiddleware recursively applies each middleware in the chain.
func (c *Chain) processMiddleware(ctx context.Context, httpClient *http.Client, req *http.Request, index int) (*http.Response, error) {
	// If we've reached the end of the middleware chain, perform the request
	if index == len(c.middlewares) {
		return c.performRequest(ctx, httpClient, req)
	}

	// Otherwise, apply the middleware and continue
	return c.middlewares[index].Process(ctx, httpClient, req, func(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
		return c.processMiddleware(ctx, client, req, index+1)
	})
}

// performRequest executes the actual HTTP request.
func (c *Chain) performRequest(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
	// Log the request details
	c.logger.WithFields(
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("len_headers", len(req.Header)),
	).Debug("Request")

	// Send the request
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %w", errors.ErrTimeout, err)
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrNetwork, err)
	}

	// Check for non-ok status codes
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("%w: %d", errors.ErrBadStatus, resp.StatusCode)
	}

	// Log the response details
	c.logger.WithFields(
		logger.Int("status", resp.StatusCode),
		logger.Int("len_headers", len(resp.Header)),
	).Debug("Response")

	return resp, nil
}

// SetLogger updates the logger for all middleware in the chain.
func (c *Chain) SetLogger(l logger.Logger) {
	for _, m := range c.middlewares {
		m.SetLogger(l)
	}
	c.logger = l
}
