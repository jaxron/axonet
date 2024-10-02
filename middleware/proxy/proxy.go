package proxy

import (
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/jaxron/axonet/pkg/client/context"
	"github.com/jaxron/axonet/pkg/client/logger"
)

var ErrInvalidTransport = errors.New("invalid transport")

// ProxyMiddleware manages proxy rotation for HTTP requests.
type ProxyMiddleware struct {
	proxies atomic.Value // stores []*url.URL
	current atomic.Uint64
	logger  logger.Logger
}

// New creates a new ProxyMiddleware instance.
func New(proxies []*url.URL) *ProxyMiddleware {
	m := &ProxyMiddleware{
		proxies: atomic.Value{},
		current: atomic.Uint64{},
		logger:  &logger.NoOpLogger{},
	}
	m.proxies.Store(proxies)
	return m
}

// Process applies proxy logic before passing the request to the next middleware.
func (m *ProxyMiddleware) Process(ctx *context.Context) (*http.Response, error) {
	proxies := m.proxies.Load().([]*url.URL)
	proxyLen := len(proxies)

	if proxyLen > 0 {
		current := m.current.Add(1) - 1          // Subtract 1 to start from 0
		index := int(current % uint64(proxyLen)) // #nosec G115
		proxy := proxies[index]

		m.logger.WithFields(logger.String("proxy", proxy.Host)).Debug("Using Proxy")

		// Clone the client to avoid modifying the original
		clonedClient := &http.Client{
			Transport:     ctx.Client.Transport,
			CheckRedirect: ctx.Client.CheckRedirect,
			Jar:           ctx.Client.Jar,
			Timeout:       ctx.Client.Timeout,
		}
		ctx.Client = clonedClient

		// Apply the proxy to the request
		transport, ok := ctx.Client.Transport.(*http.Transport)
		if !ok {
			return nil, ErrInvalidTransport
		}
		transport.Proxy = http.ProxyURL(proxy)
		ctx.Client.Transport = transport
	}

	return ctx.Next(ctx)
}

// UpdateProxies updates the list of proxies at runtime.
func (m *ProxyMiddleware) UpdateProxies(newProxies []*url.URL) {
	m.proxies.Store(newProxies)
	m.current.Store(0)

	m.logger.WithFields(logger.Int("proxy_count", len(newProxies))).Debug("Proxies updated")
}

// GetProxyCount returns the current number of proxies in the list.
func (m *ProxyMiddleware) GetProxyCount() int {
	proxies := m.proxies.Load().([]*url.URL)
	return len(proxies)
}

// SetLogger sets the logger for the middleware.
func (m *ProxyMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
