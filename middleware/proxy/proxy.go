package proxy

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

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
	mu     sync.RWMutex
	state  *proxyState
	logger logger.Logger
}

type proxyState struct {
	proxies  []*url.URL
	lastUsed []time.Time
}

// New creates a new ProxyMiddleware instance.
func New(proxies []*url.URL) *ProxyMiddleware {
	lastUsed := make([]time.Time, len(proxies))
	for i := range lastUsed {
		lastUsed[i] = time.Now().Add(-24 * time.Hour) // Initialize with a past time
	}
	return &ProxyMiddleware{
		mu:     sync.RWMutex{},
		state:  &proxyState{proxies: proxies, lastUsed: lastUsed},
		logger: &logger.NoOpLogger{},
	}
}

// Process applies proxy logic before passing the request to the next middleware.
func (m *ProxyMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	if skipProxy, ok := ctx.Value(KeySkipProxy).(bool); ok && skipProxy {
		m.logger.Debug("Skipping proxy for this request")
		return next(ctx, httpClient, req)
	}

	m.logger.Debug("Processing request with proxy middleware")

	m.mu.RLock()
	proxyLen := len(m.state.proxies)
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

// selectProxy chooses the next proxy to use based on a weighted random selection.
func (m *ProxyMiddleware) selectProxy() *url.URL {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := 0.0
	weights := make([]float64, len(m.state.proxies))
	now := time.Now()

	for i, lastUsed := range m.state.lastUsed {
		timeSinceUse := now.Sub(lastUsed).Hours()
		weight := 1.0 + timeSinceUse // Add 1 to avoid zero weight
		weights[i] = weight
		total += weight
	}

	r := rand.Float64() * total
	for i, weight := range weights {
		r -= weight
		if r <= 0 {
			m.state.lastUsed[i] = now
			return m.state.proxies[i]
		}
	}

	// Fallback to last proxy
	lastIndex := len(m.state.proxies) - 1
	m.state.lastUsed[lastIndex] = now
	return m.state.proxies[lastIndex]
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

	newLastUsed := make([]time.Time, len(newProxies))
	for i := range newLastUsed {
		if i < len(m.state.lastUsed) {
			newLastUsed[i] = m.state.lastUsed[i]
		} else {
			newLastUsed[i] = time.Now().Add(-24 * time.Hour)
		}
	}
	m.state = &proxyState{proxies: newProxies, lastUsed: newLastUsed}

	m.logger.WithFields(logger.Int("proxy_count", len(newProxies))).Debug("Proxies updated")
}

// GetProxyCount returns the current number of proxies in the list.
func (m *ProxyMiddleware) GetProxyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.state.proxies)
}

// SetLogger sets the logger for the middleware.
func (m *ProxyMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}
