package rediscache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cespare/xxhash"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/jaxron/axonet/pkg/client/middleware"
	"github.com/redis/rueidis"
)

// RedisCacheMiddleware implements a caching middleware using Redis.
type RedisCacheMiddleware struct {
	client     rueidis.Client
	logger     logger.Logger
	expiration time.Duration
}

// cachedResponse represents the structure of a cached HTTP response.
type cachedResponse struct {
	Status           string      `json:"status"`
	StatusCode       int         `json:"statusCode"`
	Header           http.Header `json:"header"`
	Body             []byte      `json:"body"`
	ContentLength    int64       `json:"contentLength"`
	TransferEncoding []string    `json:"transferEncoding"`
	Uncompressed     bool        `json:"uncompressed"`
	Trailer          http.Header `json:"trailer"`
}

// New creates a new RedisCacheMiddleware instance.
func New(clientOptions rueidis.ClientOption, expiration time.Duration) (*RedisCacheMiddleware, error) {
	client, err := rueidis.NewClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	return &RedisCacheMiddleware{
		client:     client,
		logger:     &logger.NoOpLogger{},
		expiration: expiration,
	}, nil
}

// Process implements the middleware.Middleware interface.
func (m *RedisCacheMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	key := m.generateKey(req)

	// Try to get the cached response
	cachedResp, err := m.getFromCache(ctx, key)
	if err == nil {
		m.logger.Debug("Cache hit")
		return m.reconstructResponse(cachedResp), nil
	}

	// Cache miss, proceed with the request
	m.logger.Debug("Cache miss")
	resp, err := next(ctx, httpClient, req)
	if err != nil {
		return resp, err
	}

	// Only cache successful responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Clone the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			m.logger.WithFields(logger.String("error", err.Error())).Error("Failed to read response body")
			return resp, nil
		}
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Cache the response
		go m.cacheResponse(ctx, key, resp, bodyBytes)
	}

	return resp, nil
}

// SetLogger sets the logger for the middleware.
func (m *RedisCacheMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}

// generateKey creates a unique cache key based on the request method, URL, headers, and body.
func (m *RedisCacheMiddleware) generateKey(req *http.Request) string {
	h := xxhash.New()
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.String()))
	for key, values := range req.Header {
		h.Write([]byte(key))
		for _, value := range values {
			h.Write([]byte(value))
		}
	}

	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			m.logger.WithFields(logger.String("error", err.Error())).Error("Failed to read request body for caching")
		}

		h.Write(body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return fmt.Sprintf("cache:%x", h.Sum64())
}

// getFromCache retrieves a cached response from Redis.
func (m *RedisCacheMiddleware) getFromCache(ctx context.Context, key string) (*cachedResponse, error) {
	cmd := m.client.B().Get().Key(key).Build()
	result, err := m.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		return nil, err
	}

	var cachedResp cachedResponse
	err = sonic.Unmarshal(result, &cachedResp)
	if err != nil {
		return nil, err
	}

	return &cachedResp, nil
}

// cacheResponse stores the HTTP response in Redis.
func (m *RedisCacheMiddleware) cacheResponse(ctx context.Context, key string, resp *http.Response, bodyBytes []byte) {
	// Create a cached response
	cachedResp := cachedResponse{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Header:           resp.Header,
		Body:             bodyBytes,
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Uncompressed:     resp.Uncompressed,
		Trailer:          resp.Trailer,
	}

	jsonData, err := sonic.Marshal(cachedResp)
	if err != nil {
		m.logger.WithFields(logger.String("error", err.Error())).Error("Failed to marshal cached response")
		return
	}

	cmd := m.client.B().Set().Key(key).Value(string(jsonData)).Ex(m.expiration).Build()
	err = m.client.Do(ctx, cmd).Error()
	if err != nil {
		m.logger.WithFields(logger.String("error", err.Error())).Error("Failed to cache response")
	}
}

// reconstructResponse creates an http.Response from a cached response.
func (m *RedisCacheMiddleware) reconstructResponse(cachedResp *cachedResponse) *http.Response {
	return &http.Response{
		Status:           cachedResp.Status,
		StatusCode:       cachedResp.StatusCode,
		Header:           cachedResp.Header,
		Body:             io.NopCloser(bytes.NewReader(cachedResp.Body)),
		ContentLength:    cachedResp.ContentLength,
		TransferEncoding: cachedResp.TransferEncoding,
		Uncompressed:     cachedResp.Uncompressed,
		Trailer:          cachedResp.Trailer,
	} //exhaustruct:ignore
}
