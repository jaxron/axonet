package redis

import (
	"bytes"
	"context"
	"errors"
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

type SkipCacheKey struct{}

// RedisMiddleware implements a caching middleware using Redis.
type RedisMiddleware struct {
	client     rueidis.Client
	logger     logger.Logger
	expiration time.Duration
}

// CachedResponse represents the structure of a cached HTTP response.
type CachedResponse struct {
	Status           string      `json:"status"`
	StatusCode       int         `json:"statusCode"`
	Header           http.Header `json:"header"`
	Body             []byte      `json:"body"`
	ContentLength    int64       `json:"contentLength"`
	TransferEncoding []string    `json:"transferEncoding"`
	Uncompressed     bool        `json:"uncompressed"`
	Trailer          http.Header `json:"trailer"`
}

// New creates a new RedisMiddleware instance.
func New(redisClient rueidis.Client, expiration time.Duration) *RedisMiddleware {
	return &RedisMiddleware{
		client:     redisClient,
		logger:     &logger.NoOpLogger{},
		expiration: expiration,
	}
}

// Process implements the middleware.Middleware interface.
func (m *RedisMiddleware) Process(ctx context.Context, httpClient *http.Client, req *http.Request, next middleware.NextFunc) (*http.Response, error) {
	// Check if caching should be skipped
	if skipCache, ok := ctx.Value(SkipCacheKey{}).(bool); ok && skipCache {
		return next(ctx, httpClient, req)
	}

	key := m.GenerateKey(req)

	// Try to get the cached response
	cachedResp, err := m.getFromCache(ctx, key)
	if err == nil {
		m.logger.Debug("Cache hit")
		return m.ReconstructResponse(cachedResp), nil
	}

	// Cache miss, proceed with the request
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
func (m *RedisMiddleware) SetLogger(l logger.Logger) {
	m.logger = l
}

// getFromCache retrieves a cached response from Redis.
func (m *RedisMiddleware) getFromCache(ctx context.Context, key string) (*CachedResponse, error) {
	cmd := m.client.B().Get().Key(key).Build()
	result, err := m.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		return nil, err
	}

	var cachedResp CachedResponse
	err = sonic.Unmarshal(result, &cachedResp)
	if err != nil {
		return nil, err
	}

	return &cachedResp, nil
}

// cacheResponse stores the HTTP response in Redis.
func (m *RedisMiddleware) cacheResponse(ctx context.Context, key string, resp *http.Response, bodyBytes []byte) {
	// Create a cached response
	cachedResp := CachedResponse{
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
	if err != nil && !errors.Is(err, context.Canceled) {
		m.logger.WithFields(logger.String("error", err.Error())).Error("Failed to cache response")
	}
}

// GenerateKey creates a unique cache key based on the request method, URL, headers, and body.
func (m *RedisMiddleware) GenerateKey(req *http.Request) string {
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

// ReconstructResponse creates an http.Response from a cached response.
func (m *RedisMiddleware) ReconstructResponse(cachedResp *CachedResponse) *http.Response {
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
