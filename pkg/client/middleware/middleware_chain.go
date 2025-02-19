package middleware

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

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
	return append([]Middleware(nil), c.middlewares...)
}

// Then adds middleware to the chain, replacing any existing middleware of the same type.
func (c *Chain) Then(middlewares ...Middleware) {
	for _, m := range middlewares {
		c.addOrReplace(m)
	}
}

// Process runs the request through all middleware in the chain.
func (c *Chain) Process(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
	// If no middlewares are defined, perform the request immediately
	if len(c.middlewares) == 0 {
		return c.performRequest(ctx, httpClient, req)
	}

	return c.processMiddleware(ctx, httpClient, req, 0)
}

// processMiddleware recursively applies each middleware in the chain.
func (c *Chain) processMiddleware(ctx context.Context, httpClient *http.Client, req *http.Request, index int) (*http.Response, error) {
	// If we've reached the end of the middleware chain, perform the request
	if index == len(c.middlewares) {
		return c.performRequest(ctx, httpClient, req)
	}

	start := time.Now()
	middleware := c.middlewares[index]

	// Otherwise, apply the middleware and continue
	resp, err := middleware.Process(ctx, httpClient, req, func(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
		c.logger.WithFields(
			logger.Int("index", index),
			logger.String("middleware", reflect.TypeOf(middleware).String()),
			logger.Duration("duration", time.Since(start)),
		).Debug("Middleware executed")
		return c.processMiddleware(ctx, client, req, index+1)
	})

	return resp, err
}

// performRequest executes the actual HTTP request.
func (c *Chain) performRequest(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log the request details
	c.logger.WithFields(
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("len_headers", len(req.Header)),
	).Debug("Request started")

	// Send the request
	resp, err := httpClient.Do(req.WithContext(ctx))
	duration := time.Since(start)
	if err != nil {
		c.logger.WithFields(
			logger.String("error", err.Error()),
			logger.Duration("duration", duration),
		).Debug("Request failed")

		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %w", errors.ErrTimeout, err)
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrNetwork, err)
	}

	// Log the response details
	c.logger.WithFields(
		logger.Int("status", resp.StatusCode),
		logger.Int("len_headers", len(resp.Header)),
		logger.Duration("duration", duration),
	).Debug("Request completed")

	return resp, nil
}

// addOrReplace adds a new middleware or replaces an existing one of the same type.
func (c *Chain) addOrReplace(m Middleware) {
	for i, existing := range c.middlewares {
		if reflect.TypeOf(existing) == reflect.TypeOf(m) {
			c.middlewares[i] = m
			m.SetLogger(c.logger)
			return
		}
	}
	c.middlewares = append(c.middlewares, m)
	m.SetLogger(c.logger)
}

// SetLogger updates the logger for all middleware in the chain.
func (c *Chain) SetLogger(l logger.Logger) {
	for _, m := range c.middlewares {
		m.SetLogger(l)
	}
	c.logger = l
}
