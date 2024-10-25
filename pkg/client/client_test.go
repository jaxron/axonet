package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/pkg/client"
	"github.com/jaxron/axonet/pkg/client/logger"
	clientMiddleware "github.com/jaxron/axonet/pkg/client/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var ErrMiddleware = errors.New("middleware error")

// NewTestClient creates a new client.Client instance for testing purposes.
func NewTestClient(opts ...client.Option) *client.Client {
	return client.NewClient(
		append([]client.Option{
			client.WithLogger(logger.NewBasicLogger()),
		}, opts...)...,
	)
}

// MockMiddleware is a mock implementation of the Middleware interface.
type MockMiddleware struct {
	mock.Mock
}

func (m *MockMiddleware) Process(ctx context.Context, c *http.Client, req *http.Request, next clientMiddleware.NextFunc) (*http.Response, error) {
	args := m.Called(ctx, c, req, next)
	// Handle the case where the response might be nil
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m *MockMiddleware) SetLogger(logger logger.Logger) {
	m.Called(logger)
}

func TestClientDo(t *testing.T) { //nolint:funlen
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

	t.Run("Successful request with MarshalBody and Result", func(t *testing.T) {
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

	t.Run("Middleware error handling", func(t *testing.T) {
		t.Parallel()

		middleware := &MockMiddleware{}
		middleware.On("SetLogger", mock.AnythingOfType("*logger.BasicLogger")).Return()
		middleware.On("Process", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, ErrMiddleware)

		client := NewTestClient(client.WithMiddleware(1, middleware))

		_, err := client.NewRequest().
			Method(http.MethodGet).
			URL("http://example.com").
			Do(context.Background())

		require.Error(t, err)
		assert.Equal(t, ErrMiddleware, err)
		middleware.AssertExpectations(t)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()

		middleware := &MockMiddleware{}
		middleware.On("SetLogger", mock.AnythingOfType("*logger.BasicLogger")).Return()
		middleware.On("Process", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				<-ctx.Done() // Wait for context cancellation
			}).
			Return(nil, context.Canceled)

		client := NewTestClient(client.WithMiddleware(1, middleware))

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := client.NewRequest().
			Method(http.MethodGet).
			URL("http://example.com").
			Do(ctx)

		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		middleware.AssertExpectations(t)
	})

	t.Run("Middleware execution order", func(t *testing.T) {
		t.Parallel()

		executionOrder := []string{}

		type HighPriorityMiddleware struct{ MockMiddleware }
		type MediumPriorityMiddleware struct{ MockMiddleware }
		type LowPriorityMiddleware struct{ MockMiddleware }

		createMiddleware := func(name string, priority int) clientMiddleware.Middleware {
			var m clientMiddleware.Middleware
			switch priority {
			case 100:
				m = &HighPriorityMiddleware{}
			case 50:
				m = &MediumPriorityMiddleware{}
			case 10:
				m = &LowPriorityMiddleware{}
			default:
				m = &MockMiddleware{}
			}

			mockMiddleware := m.(interface {
				On(methodName string, args ...interface{}) *mock.Call
			})
			mockMiddleware.On("SetLogger", mock.AnythingOfType("*logger.BasicLogger")).Return()
			mockMiddleware.On("Process", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					executionOrder = append(executionOrder, name)
					next := args.Get(3).(clientMiddleware.NextFunc)
					_, err := next(args.Get(0).(context.Context), args.Get(1).(*http.Client), args.Get(2).(*http.Request))
					assert.NoError(t, err)
				}).
				Return(&http.Response{StatusCode: http.StatusOK}, nil)
			return m
		}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer mockServer.Close()

		client := NewTestClient(
			client.WithMiddleware(100, createMiddleware("High", 100)),
			client.WithMiddleware(50, createMiddleware("Medium", 50)),
			client.WithMiddleware(10, createMiddleware("Low", 10)),
		)

		_, err := client.NewRequest().
			Method(http.MethodGet).
			URL(mockServer.URL).
			Do(context.Background())

		require.NoError(t, err)
		assert.Equal(t, []string{"High", "Medium", "Low"}, executionOrder)
	})
}
