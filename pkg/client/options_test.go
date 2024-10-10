package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxron/axonet/pkg/client"
	"github.com/jaxron/axonet/pkg/client/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWithMiddleware(t *testing.T) {
	mockMiddleware := &MockMiddleware{}
	mockMiddleware.On("SetLogger", mock.AnythingOfType("*logger.BasicLogger")).Return()
	mockMiddleware.On("Process", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK}, nil)

	c := NewTestClient(client.WithMiddleware(mockMiddleware))

	_, err := c.NewRequest().
		Method(http.MethodGet).
		URL("http://example.com").
		Do(context.Background())

	require.NoError(t, err)
	mockMiddleware.AssertExpectations(t)
}

func TestWithTimeout(t *testing.T) {
	// Create a test server that sleeps for 100ms
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with a timeout shorter than the server response time
	c := NewTestClient(client.WithTimeout(50 * time.Millisecond))

	_, err := c.NewRequest().
		Method(http.MethodGet).
		URL(server.URL).
		Do(context.Background())

	// Expect a timeout error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

	// Test with a timeout longer than the server response time
	c = NewTestClient(client.WithTimeout(200 * time.Millisecond))

	_, err = c.NewRequest().
		Method(http.MethodGet).
		URL(server.URL).
		Do(context.Background())

	// Expect no error
	require.NoError(t, err)
}

func TestWithLogger(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Debug", mock.Anything).Return()

	c := NewTestClient(client.WithLogger(mockLogger))

	_, err := c.NewRequest().
		Method(http.MethodGet).
		URL("http://example.com").
		Do(context.Background())

	require.NoError(t, err)
	mockLogger.AssertExpectations(t)
}

func TestMarshalWith(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]string
		err := json.NewDecoder(r.Body).Decode(&data)
		require.NoError(t, err)
		assert.Equal(t, "custom", data["format"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	customMarshal := func(v interface{}) ([]byte, error) {
		return json.Marshal(map[string]string{"format": "custom"})
	}

	_, err := NewTestClient().NewRequest().
		Method(http.MethodPost).
		URL(server.URL).
		MarshalWith(customMarshal).
		MarshalBody(struct{}{}).
		Do(context.Background())

	require.NoError(t, err)
}

func TestUnmarshalWith(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"format":"custom"}`))
	}))
	defer server.Close()

	var result map[string]string
	customUnmarshal := func(data []byte, v interface{}) error {
		return json.Unmarshal(data, v)
	}

	_, err := NewTestClient().NewRequest().
		Method(http.MethodGet).
		URL(server.URL).
		UnmarshalWith(customUnmarshal).
		Result(&result).
		Do(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "custom", result["format"])
}

func TestQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value", r.URL.Query().Get("key"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := NewTestClient().NewRequest().
		Method(http.MethodGet).
		URL(server.URL).
		Query("key", "value").
		Do(context.Background())

	require.NoError(t, err)
}

// MockLogger implementation
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) WithFields(fields ...logger.Field) logger.Logger {
	args := m.Called(fields)
	return args.Get(0).(logger.Logger)
}

func (m *MockLogger) Debug(msg string) {
	m.Called(msg)
}

func (m *MockLogger) Info(msg string) {
	m.Called(msg)
}

func (m *MockLogger) Warn(msg string) {
	m.Called(msg)
}

func (m *MockLogger) Error(msg string) {
	m.Called(msg)
}

// Add these new methods to comply with the logger.Logger interface
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}
