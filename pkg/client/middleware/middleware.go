package middleware

import (
	"context"
	"net/http"

	"github.com/jaxron/axonet/pkg/client/logger"
)

// NextFunc is a function type that represents the next middleware in the chain.
type NextFunc func(context.Context, *http.Client, *http.Request) (*http.Response, error)

// Middleware interface for all HTTP middleware components.
type Middleware interface {
	Process(ctx context.Context, httpClient *http.Client, req *http.Request, next NextFunc) (*http.Response, error)
	SetLogger(l logger.Logger)
}
