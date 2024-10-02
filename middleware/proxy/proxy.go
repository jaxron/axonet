package proxy

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
)

// ProxyMiddleware manages proxy rotation for HTTP requests.
type ProxyMiddleware struct {
	proxies []*url.URL
	current int
	logger  logger.Logger
	mu      sync.RWMutex
}

// New creates a new ProxyMiddleware instance.
func New(proxies []*url.URL) *ProxyMiddleware {
	return &ProxyMiddleware{
		proxies: proxies,
		current: 0,
		logger:  &logger.NoOpLogger{},
		mu:      sync.RWMutex{},
	}
}

// Process applies proxy logic before passing the request to the next middleware.
func (m *ProxyMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxies) > 0 {
		// Get the next proxy to use
		proxy := m.proxies[m.current]
		m.current = (m.current + 1) % len(m.proxies)

		m.logger.WithFields(logger.String("proxy", proxy.Host)).Debug("Using Proxy")

		// Apply the proxy to the request
		transport, ok := ctx.Client.Transport.(*http.Transport)
		if !ok {
			return nil, errors.ErrInvalidTransport
		}
		transport.Proxy = http.ProxyURL(proxy)
		ctx.Client.Transport = transport
	}
	return ctx.Next(ctx)
}

// UpdateProxies updates the list of proxies at runtime.
// It replaces the existing proxy list with the new one and randomizes the starting proxy.
func (m *ProxyMiddleware) UpdateProxies(newProxies []*url.URL) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.proxies = newProxies
	m.current = 0

	m.logger.WithFields(logger.Int("proxy_count", len(newProxies))).Debug("Proxies updated")
}

// GetProxyCount returns the current number of proxies in the list.
func (m *ProxyMiddleware) GetProxyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.proxies)
}

// SetLogger sets the logger for the middleware.
func (m *ProxyMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
