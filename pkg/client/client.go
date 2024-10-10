// package client provides HTTP request handling functionality with various middleware options.
package client

import (
	"context"
	"net/http"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

// Client manages HTTP requests with various middleware options.
type Client struct {
	middlewareChain *middleware.Chain
	httpClient      *http.Client
}

// NewClient creates a new Client instance with default settings.
func NewClient(opts ...Option) *Client {
	client := &Client{
		middlewareChain: middleware.NewChain(&logger.NoOpLogger{}),
		httpClient: &http.Client{
			Transport:     http.DefaultTransport,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       0,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Do performs an HTTP request with the specified options.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.middlewareChain.Process(ctx, c.httpClient, req)
}
