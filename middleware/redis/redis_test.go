package redis_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/jaxron/axonet/middleware/redis"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Generate unique keys for different requests", func(t *testing.T) {
		t.Parallel()

		middleware := redis.RedisMiddleware{}
		middleware.SetLogger(logger.NewBasicLogger())

		req1 := httptest.NewRequest(http.MethodGet, "http://example.com/path1", nil)
		req2 := httptest.NewRequest(http.MethodGet, "http://example.com/path2", nil)
		req3 := httptest.NewRequest(http.MethodPost, "http://example.com/path1", nil)

		key1 := middleware.GenerateKey(req1)
		key2 := middleware.GenerateKey(req2)
		key3 := middleware.GenerateKey(req3)

		assert.NotEqual(t, key1, key2, "Keys for different paths should be different")
		assert.NotEqual(t, key1, key3, "Keys for different methods should be different")
		assert.NotEqual(t, key2, key3, "Keys for different methods and paths should be different")
	})

	t.Run("Cache and reconstruct response", func(t *testing.T) {
		t.Parallel()

		middleware := redis.RedisMiddleware{}
		middleware.SetLogger(logger.NewBasicLogger())

		originalResp := &http.Response{
			Status:           "200 OK",
			StatusCode:       http.StatusOK,
			Header:           http.Header{"Content-Type": []string{"application/json"}},
			Body:             io.NopCloser(bytes.NewBufferString(`{"message":"Hello, World!"}`)),
			ContentLength:    25,
			TransferEncoding: []string{"gzip"},
			Uncompressed:     false,
			Trailer:          http.Header{"X-Trailer": []string{"trailer-value"}},
		}

		// Simulate caching the response
		body, err := io.ReadAll(originalResp.Body)
		require.NoError(t, err)
		originalResp.Body.Close()
		originalResp.Body = io.NopCloser(bytes.NewReader(body))

		cachedResp := &redis.CachedResponse{
			Status:           originalResp.Status,
			StatusCode:       originalResp.StatusCode,
			Header:           originalResp.Header,
			Body:             body,
			ContentLength:    originalResp.ContentLength,
			TransferEncoding: originalResp.TransferEncoding,
			Uncompressed:     originalResp.Uncompressed,
			Trailer:          originalResp.Trailer,
		}

		// Reconstruct the response
		reconstructedResp := middleware.ReconstructResponse(cachedResp)

		assert.Equal(t, originalResp.Status, reconstructedResp.Status)
		assert.Equal(t, originalResp.StatusCode, reconstructedResp.StatusCode)
		assert.Equal(t, originalResp.Header, reconstructedResp.Header)
		assert.Equal(t, originalResp.ContentLength, reconstructedResp.ContentLength)
		assert.Equal(t, originalResp.TransferEncoding, reconstructedResp.TransferEncoding)
		assert.Equal(t, originalResp.Uncompressed, reconstructedResp.Uncompressed)
		assert.Equal(t, originalResp.Trailer, reconstructedResp.Trailer)

		reconstructedBody, err := io.ReadAll(reconstructedResp.Body)
		require.NoError(t, err)
		assert.Equal(t, body, reconstructedBody)
	})
}

func TestCachedResponseSerialization(t *testing.T) {
	t.Parallel()

	t.Run("Marshal and unmarshal cachedResponse", func(t *testing.T) {
		t.Parallel()

		original := &redis.CachedResponse{
			Status:           "200 OK",
			StatusCode:       http.StatusOK,
			Header:           http.Header{"Content-Type": []string{"application/json"}},
			Body:             []byte(`{"message":"Hello, World!"}`),
			ContentLength:    25,
			TransferEncoding: []string{"gzip"},
			Uncompressed:     false,
			Trailer:          http.Header{"X-Trailer": []string{"trailer-value"}},
		}

		// Marshal
		data, err := sonic.Marshal(original)
		require.NoError(t, err)

		// Unmarshal
		var reconstructed redis.CachedResponse
		err = sonic.Unmarshal(data, &reconstructed)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.Status, reconstructed.Status)
		assert.Equal(t, original.StatusCode, reconstructed.StatusCode)
		assert.Equal(t, original.Header, reconstructed.Header)
		assert.Equal(t, original.Body, reconstructed.Body)
		assert.Equal(t, original.ContentLength, reconstructed.ContentLength)
		assert.Equal(t, original.TransferEncoding, reconstructed.TransferEncoding)
		assert.Equal(t, original.Uncompressed, reconstructed.Uncompressed)
		assert.Equal(t, original.Trailer, reconstructed.Trailer)
	})
}

func TestShouldCacheResponse(t *testing.T) {
	t.Parallel()

	middleware := &redis.RedisMiddleware{}
	middleware.SetLogger(logger.NewBasicLogger())

	tests := []struct {
		name        string
		contentType string
		body        []byte
		want        bool
	}{
		{
			name:        "non-JSON response should always cache",
			contentType: "text/html",
			body:        []byte("<html>test</html>"),
			want:        true,
		},
		{
			name:        "valid JSON response should cache",
			contentType: "application/json",
			body:        []byte(`{"test": "value"}`),
			want:        true,
		},
		{
			name:        "empty JSON object should cache",
			contentType: "application/json",
			body:        []byte(`{}`),
			want:        true,
		},
		{
			name:        "empty JSON array should cache",
			contentType: "application/json",
			body:        []byte(`[]`),
			want:        true,
		},
		{
			name:        "JSON null should cache",
			contentType: "application/json",
			body:        []byte(`null`),
			want:        true,
		},
		{
			name:        "empty response with JSON content-type should not cache",
			contentType: "application/json",
			body:        []byte{},
			want:        false,
		},
		{
			name:        "invalid JSON with JSON content-type should not cache",
			contentType: "application/json",
			body:        []byte("plain text"),
			want:        false,
		},
		{
			name:        "plain text with non-JSON content-type should cache",
			contentType: "text/plain",
			body:        []byte("plain text"),
			want:        true,
		},
		{
			name:        "empty response with non-JSON content-type should cache",
			contentType: "text/plain",
			body:        []byte{},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}

			got := middleware.ShouldCacheResponse(resp, tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}
