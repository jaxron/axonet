package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

var ErrInvalidTransport = errors.New("invalid transport")

type contextKey int

const (
	KeySkipProxy contextKey = iota
)

// ProxyMiddleware manages proxy rotation for HTTP requests.
type ProxyMiddleware struct {
	proxies atomic.Value
	current atomic.Uint64
	logger  logger.Logger
}

type proxyState struct {
	proxies []*url.URL
}

// New creates a new ProxyMiddleware instance.
func New(proxies []*url.URL) *ProxyMiddleware {
	m := &ProxyMiddleware{
		proxies: atomic.Value{},
		current: atomic.Uint64{},
		logger:  &logger.NoOpLogger{},
	}
	m.proxies.Store(&proxyState{proxies: proxies})
	return m
}

// Process applies proxy logic before passing the request to the next middleware.
func (m *ProxyMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Check if the proxy should be skipped for this request
	if skipProxy, ok := ctx.Value(KeySkipProxy).(bool); ok && skipProxy {
		m.logger.Debug("Skipping proxy for this request")
		return next(ctx, httpClient, req)
	}

	m.logger.Debug("Processing request with proxy middleware")

	state := m.proxies.Load().(*proxyState)
	proxyLen := len(state.proxies)

	if proxyLen > 0 {
		current := m.current.Add(1) - 1
		index := int(current % uint64(proxyLen)) // #nosec G115
		proxy := state.proxies[index]

		m.logger.WithFields(logger.String("proxy", proxy.Host)).Debug("Using Proxy")

		// Apply the proxy to the request
		transport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return nil, ErrInvalidTransport
		}
		transport = transport.Clone()
		transport.Proxy = http.ProxyURL(proxy)
		transport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
			m.logger.WithFields(logger.String("proxy", proxyURL.Host)).Debug("Proxy connection established")
			return nil
		}

		// Shallow copy the client to avoid modifying the original because
		// it's shared across requests and is unsafe for concurrent use
		httpClient = &http.Client{
			Transport:     transport,
			CheckRedirect: httpClient.CheckRedirect,
			Jar:           httpClient.Jar,
			Timeout:       httpClient.Timeout,
		}
	}

	return next(ctx, httpClient, req)
}

// UpdateProxies updates the list of proxies at runtime.
func (m *ProxyMiddleware) UpdateProxies(newProxies []*url.URL) {
	newState := &proxyState{proxies: newProxies}
	m.proxies.Store(newState)
	m.current.Store(0)

	m.logger.WithFields(logger.Int("proxy_count", len(newProxies))).Debug("Proxies updated")
}

// GetProxyCount returns the current number of proxies in the list.
func (m *ProxyMiddleware) GetProxyCount() int {
	state := m.proxies.Load().(*proxyState)
	return len(state.proxies)
}

// SetLogger sets the logger for the middleware.
func (m *ProxyMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
