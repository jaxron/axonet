package proxy

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
)

var ErrInvalidTransport = errors.New("invalid transport")

type SkipProxyKey struct{}

// ProxyMiddleware manages proxy rotation for HTTP requests.
type ProxyMiddleware struct {
	proxies    []*url.URL
	proxyCount int
	current    atomic.Uint64
	mu         sync.RWMutex
	logger     logger.Logger
}

// New creates a new ProxyMiddleware instance.
func New(proxies []*url.URL) *ProxyMiddleware {
	m := &ProxyMiddleware{
		proxies:    proxies,
		proxyCount: len(proxies),
		current:    atomic.Uint64{},
		mu:         sync.RWMutex{},
		logger:     &logger.NoOpLogger{},
	}
	m.current.Store(0)
	return m
}

// Process applies proxy logic before passing the request to the next middleware.
func (m *ProxyMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	if skipProxy, ok := ctx.Value(SkipProxyKey{}).(bool); ok && skipProxy {
		return next(ctx, httpClient, req)
	}

	m.mu.RLock()
	proxyLen := len(m.proxies)
	m.mu.RUnlock()

	if proxyLen > 0 {
		proxy := m.selectProxy()
		m.logger.WithFields(logger.String("proxy", proxy.Host)).Debug("Using Proxy")

		var err error
		httpClient, err = m.applyProxyToClient(httpClient, proxy)
		if err != nil {
			return nil, err
		}
	}

	return next(ctx, httpClient, req)
}

// selectProxy chooses the next proxy to use.
func (m *ProxyMiddleware) selectProxy() *url.URL {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.proxyCount == 0 {
		return nil
	}

	current := m.current.Add(1) - 1
	index := current % uint64(m.proxyCount) // #nosec G115
	return m.proxies[index]
}

// applyProxyToClient applies the proxy to the given http.Client.
func (m *ProxyMiddleware) applyProxyToClient(httpClient *http.Client, proxy *url.URL) (*http.Client, error) {
	// Get the transport from the client
	transport, err := m.getTransport(httpClient)
	if err != nil {
		return nil, err
	}

	// Clone the transport
	newTransport := transport.Clone()

	// Modify only the necessary fields
	newTransport.Proxy = http.ProxyURL(proxy)
	newTransport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
		m.logger.WithFields(logger.String("proxy", proxyURL.Host)).Debug("Proxy connection established")
		return nil
	}

	// Create a new client with the modified transport
	return &http.Client{
		Transport:     newTransport,
		CheckRedirect: httpClient.CheckRedirect,
		Jar:           httpClient.Jar,
		Timeout:       httpClient.Timeout,
	}, nil
}

func (m *ProxyMiddleware) getTransport(httpClient *http.Client) (*http.Transport, error) {
	if t, ok := httpClient.Transport.(*http.Transport); ok {
		return t, nil
	}
	if httpClient.Transport == nil {
		return http.DefaultTransport.(*http.Transport), nil
	}
	return nil, ErrInvalidTransport
}

// UpdateProxies updates the list of proxies at runtime.
func (m *ProxyMiddleware) UpdateProxies(newProxies []*url.URL) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.proxies = newProxies
	m.proxyCount = len(newProxies)
	m.current.Store(0)

	m.logger.WithFields(logger.Int("proxy_count", len(newProxies))).Debug("Proxies updated")
}

// Shuffle randomizes the order of the proxies.
func (m *ProxyMiddleware) Shuffle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(m.proxies), func(i, j int) {
		m.proxies[i], m.proxies[j] = m.proxies[j], m.proxies[i]
	})

	m.logger.Debug("Proxies shuffled")
}

// GetProxyCount returns the current number of proxies in the list.
func (m *ProxyMiddleware) GetProxyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.proxyCount
}

// SetLogger sets the logger for the middleware.
func (m *ProxyMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
