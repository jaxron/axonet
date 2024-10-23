package middleware

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"

	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// PrioritizedMiddleware wraps a Middleware with a priority value.
type PrioritizedMiddleware struct {
	Middleware Middleware
	Priority   int
}

// Chain represents a chain of middleware.
type Chain struct {
	middlewares []PrioritizedMiddleware
	logger      logger.Logger
}

// NewChain creates a new middleware chain.
func NewChain(logger logger.Logger, middlewares ...PrioritizedMiddleware) *Chain {
	chain := &Chain{
		middlewares: middlewares,
		logger:      logger,
	}
	chain.sortMiddlewares()
	return chain
}

// Len returns the number of middlewares in the chain.
func (c *Chain) Len() int {
	return len(c.middlewares)
}

// Middlewares returns the slice of middlewares.
func (c *Chain) Middlewares() []Middleware {
	result := make([]Middleware, len(c.middlewares))
	for i, pm := range c.middlewares {
		result[i] = pm.Middleware
	}
	return result
}

// Then adds middleware to the chain, replacing any existing middleware of the same type.
func (c *Chain) Then(priority int, middlewares ...Middleware) {
	for _, m := range middlewares {
		c.addOrReplace(priority, m)
	}
	c.sortMiddlewares()
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
	for i, pm := range c.middlewares {
		c.logger.WithFields(
			logger.Int("index", i),
			logger.String("type", reflect.TypeOf(pm.Middleware).String()),
			logger.Int("priority", pm.Priority),
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
	return c.middlewares[index].Middleware.Process(ctx, httpClient, req, func(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
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

	// Log the response details
	c.logger.WithFields(
		logger.Int("status", resp.StatusCode),
		logger.Int("len_headers", len(resp.Header)),
	).Debug("Response")

	return resp, nil
}

// addOrReplace adds a new middleware or replaces an existing one of the same type.
func (c *Chain) addOrReplace(priority int, m Middleware) {
	for i, pm := range c.middlewares {
		if reflect.TypeOf(pm.Middleware) == reflect.TypeOf(m) {
			c.middlewares[i] = PrioritizedMiddleware{Middleware: m, Priority: priority}
			m.SetLogger(c.logger)
			return
		}
	}
	c.middlewares = append(c.middlewares, PrioritizedMiddleware{Middleware: m, Priority: priority})
	m.SetLogger(c.logger)
}

// sortMiddlewares sorts the middlewares by priority (descending order).
func (c *Chain) sortMiddlewares() {
	sort.Slice(c.middlewares, func(i, j int) bool {
		return c.middlewares[i].Priority > c.middlewares[j].Priority
	})
}

// SetLogger updates the logger for all middleware in the chain.
func (c *Chain) SetLogger(l logger.Logger) {
	for _, pm := range c.middlewares {
		pm.Middleware.SetLogger(l)
	}
	c.logger = l
}
