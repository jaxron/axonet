package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

// MarshalFunc is a function type that matches standard marshal functions.
type MarshalFunc func(interface{}) ([]byte, error)

// UnmarshalFunc is a function type that matches standard unmarshal functions.
type UnmarshalFunc func([]byte, interface{}) error

// Option is a function type that modifies the Client configuration.
type Option func(*Client)

// WithMiddleware adds or updates the middleware for the Client with a specified priority.
func WithMiddleware(middleware middleware.Middleware) Option {
	return func(c *Client) {
		c.middlewareChain.Then(middleware)
	}
}

// WithTimeout sets the timeout for the Client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithLogger sets the logger for the Client and its middleware.
func WithLogger(logger logger.Logger) Option {
	return func(c *Client) {
		c.middlewareChain.SetLogger(logger)
	}
}

// WithMarshalFunc sets the marshal function for the Client.
func WithMarshalFunc(fn MarshalFunc) Option {
	return func(c *Client) {
		c.marshalFunc = fn
	}
}

// WithUnmarshalFunc sets the unmarshal function for the Client.
func WithUnmarshalFunc(fn UnmarshalFunc) Option {
	return func(c *Client) {
		c.unmarshalFunc = fn
	}
}

// Request helps build requests using method chaining.
type Request struct {
	client        *Client
	marshalFunc   MarshalFunc
	unmarshalFunc UnmarshalFunc
	result        interface{}
	method        string
	url           string
	body          []byte
	marshalBody   interface{}
	header        http.Header
	query         Query
}

// NewRequest creates a new Request with default options.
func (c *Client) NewRequest() *Request {
	return &Request{
		client:        c,
		marshalFunc:   c.marshalFunc,
		unmarshalFunc: c.unmarshalFunc,
		result:        nil,
		method:        "",
		url:           "",
		body:          nil,
		marshalBody:   nil,
		header:        make(http.Header),
		query:         make(Query),
	}
}

// Method sets the HTTP method for the request.
func (rb *Request) Method(method string) *Request {
	rb.method = method
	return rb
}

// URL sets the URL for the request.
func (rb *Request) URL(url string) *Request {
	rb.url = url
	return rb
}

// MarshalWith sets the marshal function for the request body.
func (rb *Request) MarshalWith(fn MarshalFunc) *Request {
	rb.marshalFunc = fn
	return rb
}

// UnmarshalWith sets the unmarshal function for the response.
func (rb *Request) UnmarshalWith(fn UnmarshalFunc) *Request {
	rb.unmarshalFunc = fn
	return rb
}

// Result sets the result to unmarshal the response into.
func (rb *Request) Result(result interface{}) *Request {
	rb.result = result
	return rb
}

// Body sets the body of the request.
func (rb *Request) Body(body []byte) *Request {
	rb.body = body
	return rb
}

// MarshalBody sets the body of the request after marshaling the provided struct.
func (rb *Request) MarshalBody(body interface{}) *Request {
	rb.marshalBody = body
	return rb
}

// Query adds a query parameter to the request.
func (rb *Request) Query(key, value string) *Request {
	rb.query.Add(key, value)
	return rb
}

// Header adds a header to the request.
func (rb *Request) Header(key, value string) *Request {
	rb.header.Set(key, value)
	return rb
}

// Build returns the final http.Request for execution.
func (rb *Request) Build(ctx context.Context) (*http.Request, error) {
	// Ensure only one of the body or marshalBody is set
	if rb.body != nil && rb.marshalBody != nil {
		return nil, errors.ErrBodyMarshalConflict
	}

	var bodyReader io.Reader

	// Marshal the body if provided
	if rb.marshalBody != nil {
		marshaledBody, err := rb.marshalFunc(rb.marshalBody)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errors.ErrRequestCreation, err)
		}
		bodyReader = bytes.NewReader(marshaledBody)
	}

	// Use the body if provided
	if rb.body != nil {
		bodyReader = bytes.NewReader(rb.body)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, rb.method, rb.url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrRequestCreation, err)
	}

	// Set the query parameters
	req.URL.RawQuery = rb.query.Encode()

	// Set the headers
	for key, values := range rb.header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return req, nil
}

// Do executes the request and returns the raw http.Response.
func (rb *Request) Do(ctx context.Context) (*http.Response, error) {
	// Build the request
	req, err := rb.Build(ctx)
	if err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := rb.client.Do(ctx, req)
	if err != nil {
		return resp, err
	}

	// If a result is set, unmarshal the response
	if rb.result != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, err
		}

		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		if err = rb.unmarshalFunc(body, rb.result); err != nil {
			return resp, err
		}
	}

	return resp, nil
}
