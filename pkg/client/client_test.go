package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/pkg/client"
	"github.com/jaxron/axonet/pkg/client/errors"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewTestClient creates a new client.Client instance for testing purposes.
func NewTestClient(opts ...client.Option) *client.Client {
	return client.NewClient(
		append([]client.Option{
			client.WithLogger(logger.NewBasicLogger()),
		}, opts...)...,
	)
}

func TestClientDo(t *testing.T) {
	t.Parallel()

	t.Run("Successful request", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"message": "success"}`))
			assert.NoError(t, err)
		}))
		defer mockServer.Close()

		resp, err := NewTestClient().
			NewRequest().
			Method(http.MethodGet).
			URL(mockServer.URL).
			Do(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]string
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["message"])
	})

	t.Run("Successful request with MarshalBody", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var receivedData map[string]string
			err = json.Unmarshal(body, &receivedData)
			assert.NoError(t, err)
			assert.Equal(t, "test", receivedData["key"])

			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{"message": "success"}`))
			assert.NoError(t, err)
		}))
		defer mockServer.Close()

		type RequestBody struct {
			Key string `json:"key"`
		}

		var result map[string]string
		resp, err := NewTestClient().
			NewRequest().
			Method(http.MethodPost).
			URL(mockServer.URL).
			MarshalBody(RequestBody{Key: "test"}).
			Result(&result).
			Do(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "success", result["message"])
	})

	t.Run("Custom marshal and unmarshal functions", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, "CUSTOM:test", string(body))

			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`CUSTOM:{"message":"success"}`))
			assert.NoError(t, err)
		}))
		defer mockServer.Close()

		customMarshal := func(v interface{}) ([]byte, error) {
			return []byte("CUSTOM:" + v.(string)), nil
		}

		customUnmarshal := func(data []byte, v interface{}) error {
			*v.(*string) = string(data[7:]) // Remove "CUSTOM:" prefix
			return nil
		}

		var result string
		resp, err := NewTestClient().
			NewRequest().
			Method(http.MethodPost).
			URL(mockServer.URL).
			MarshalWith(customMarshal).
			UnmarshalWith(customUnmarshal).
			MarshalBody("test").
			Result(&result).
			Do(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"message":"success"}`, result)
	})

	t.Run("Error when both Body and MarshalBody are set", func(t *testing.T) {
		t.Parallel()

		_, err := NewTestClient().
			NewRequest().
			Method(http.MethodPost).
			URL("http://example.com").
			Body([]byte("test")).
			MarshalBody(struct{ Key string }{Key: "value"}).
			Do(context.Background())

		require.Error(t, err)
		assert.ErrorIs(t, err, errors.ErrBodyMarshalConflict)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer mockServer.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := NewTestClient().
			NewRequest().
			Method(http.MethodGet).
			URL(mockServer.URL).
			Do(ctx)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestClientDoUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("Successful unmarshal", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"message": "success", "code": 200}`))
			assert.NoError(t, err)
		}))
		defer mockServer.Close()

		var result struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}

		resp, err := NewTestClient().
			NewRequest().
			Method(http.MethodGet).
			URL(mockServer.URL).
			Result(&result).
			Do(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "success", result.Message)
		assert.Equal(t, 200, result.Code)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		t.Parallel()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"message": "success", "code": "not a number"}`))
			assert.NoError(t, err)
		}))
		defer mockServer.Close()

		var result struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}

		_, err := NewTestClient().
			NewRequest().
			Method(http.MethodGet).
			URL(mockServer.URL).
			Result(&result).
			Do(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "json: cannot unmarshal string into Go struct field")
	})
}
