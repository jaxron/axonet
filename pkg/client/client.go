// package client provides HTTP request handling functionality with various middleware options.
package client

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

// Client manages HTTP requests with various middleware options.
type Client struct {
	middlewares       []middleware.Middleware
	defaultHTTPClient *http.Client
	Logger            logger.Logger
}

// NewClient creates a new Client instance with default settings.
func NewClient(opts ...Option) *Client {
	// Create a new client with default settings
	noOpLogger := &logger.NoOpLogger{}
	client := &Client{
		middlewares: []middleware.Middleware{},
		defaultHTTPClient: &http.Client{
			Transport:     http.DefaultTransport,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       0, // No client timeout as context timeout is used
		},
		Logger: noOpLogger,
	}

	// Apply all provided options to customize the client
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Do performs an HTTP request with the specified options.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Log the available middleware in the chain
	for i, m := range c.middlewares {
		c.Logger.WithFields(
			logger.Int("index", i),
			logger.String("type", reflect.TypeOf(m).String()),
		).Debug("Middleware in chain")
	}

	return c.executeMiddlewareChain(ctx, c.defaultHTTPClient, req, 0)
}

// executeMiddlewareChain recursively executes the middleware chain.
func (c *Client) executeMiddlewareChain(ctx context.Context, httpClient *http.Client, req *http.Request, index int) (*http.Response, error) {
	if index == len(c.middlewares) {
		// All middleware processed, execute the actual request
		return c.performRequest(ctx, httpClient, req)
	}

	// Execute the current middleware
	return c.middlewares[index].Process(ctx, httpClient, req, func(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
		// Move to the next middleware in the chain
		return c.executeMiddlewareChain(ctx, httpClient, req, index+1)
	})
}

// performRequest executes the actual HTTP request.
func (c *Client) performRequest(ctx context.Context, httpClient *http.Client, req *http.Request) (*http.Response, error) {
	// Log the request details
	c.Logger.WithFields(
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("len_headers", len(req.Header)),
	).Debug("Request")

	// Send the request
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request timed out: %w: %w", errors.ErrTimeout, err)
		}
		return nil, fmt.Errorf("network error occurred: %w: %w", errors.ErrNetwork, err)
	}

	// Check for non-ok status codes
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %w: %d", errors.ErrBadStatus, resp.StatusCode)
	}

	// Log the response details
	c.Logger.WithFields(
		logger.Int("status", resp.StatusCode),
		logger.Int("len_headers", len(resp.Header)),
	).Debug("Response")

	return resp, nil
}

// SetLogger updates the client's logger and propagates it to all middleware.
func (c *Client) SetLogger(l logger.Logger) {
	// Update all middleware loggers
	for _, m := range c.middlewares {
		m.SetLogger(l)
	}
	c.Logger = l
}

// updateMiddleware adds or replaces a middleware in the client's middleware chain.
func (c *Client) updateMiddleware(newMiddleware middleware.Middleware) {
	for i, m := range c.middlewares {
		if reflect.TypeOf(m) == reflect.TypeOf(newMiddleware) {
			c.middlewares[i] = newMiddleware
			newMiddleware.SetLogger(c.Logger)
			return
		}
	}
	c.middlewares = append(c.middlewares, newMiddleware)
	newMiddleware.SetLogger(c.Logger)
}
