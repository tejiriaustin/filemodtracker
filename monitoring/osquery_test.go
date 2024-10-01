package monitoring

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/tejiriaustin/savannah-assessment/logger"
)

// MockWriter is a mock implementation of io.WriteCloser
type MockWriter struct {
	mock.Mock
	bytes.Buffer
}

func (m *MockWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

// MockReader is a mock implementation of io.ReadCloser
type MockReader struct {
	mock.Mock
	*bytes.Reader
}

func (m *MockReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestNew tests the New function
func TestNew(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := "/tmp/test_config.json"

	client, err := New(configPath, WithLogger(mockLogger))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, configPath, client.configPath)
	assert.Equal(t, "osqueryi", client.osqueryBinary)
	assert.Equal(t, "/var/tmp/osquery_data/osquery.db", client.databasePath)
	assert.Equal(t, 3, client.maxRetries)
}

// TestCreateConfig tests the createConfig method
func TestCreateConfig(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithMonitorDirs([]string{"/home/user"}))
	assert.NoError(t, err)

	err = client.createConfig()
	assert.NoError(t, err)

	// Read and parse the created config file
	configData, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	var config map[string]interface{}
	err = json.Unmarshal(configData, &config)
	assert.NoError(t, err)

	// Check if the config contains expected keys
	assert.Contains(t, config, "schedule")
	assert.Contains(t, config, "file_paths")
}

// TestQuery tests the Query method
func TestQuery(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithOsqueryBinary("echo"))
	assert.NoError(t, err)

	// Mock stdin
	mockStdin := new(MockWriter)
	mockStdin.On("Write", mock.Anything).Return(len("SELECT * FROM users LIMIT 1"), nil)
	client.stdin = mockStdin

	// Mock stdout
	mockResult := `[{"username":"test_user","uid":"1000"}]`
	mockStdout := NewMockReader([]byte(mockResult))
	client.stdout = mockStdout

	results, err := client.Query("SELECT * FROM users LIMIT 1")
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, "test_user", results[0]["username"])

	mockStdin.AssertExpectations(t)
}

// TestGetFileEvents tests the GetFileEvents method
func TestGetFileEvents(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithOsqueryBinary("echo"))
	assert.NoError(t, err)

	// Mock stdin
	mockStdin := new(MockWriter)
	mockStdin.On("Write", mock.Anything).Return(len("SELECT * FROM file_events;"), nil)
	client.stdin = mockStdin

	// Mock stdout
	mockResult := `[{"path":"/test/file","action":"CREATED"}]`
	mockStdout := NewMockReader([]byte(mockResult))
	client.stdout = mockStdout

	events, err := client.GetFileEvents()
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 1)
	assert.Equal(t, "/test/file", events[0]["path"])

	mockStdin.AssertExpectations(t)
}

// TestGetFileEventsByPath tests the GetFileEventsByPath method
func TestGetFileEventsByPath(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithOsqueryBinary("echo"))
	assert.NoError(t, err)

	// Mock stdin
	mockStdin := new(MockWriter)
	mockStdin.On("Write", mock.Anything).Return(100, nil) // Assume query length is 100
	client.stdin = mockStdin

	// Mock stdout
	mockResult := `[{"path":"/test/path/file","action":"MODIFIED"}]`
	mockStdout := NewMockReader([]byte(mockResult))
	client.stdout = mockStdout

	path := "/test/path"
	since := time.Now().Add(-1 * time.Hour)
	events, err := client.GetFileEventsByPath(path, since)
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.Len(t, events, 1)
	assert.Equal(t, "/test/path/file", events[0]["path"])

	mockStdin.AssertExpectations(t)
}

// TestGetFileChangesSummary tests the GetFileChangesSummary method
func TestGetFileChangesSummary(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithOsqueryBinary("echo"))
	assert.NoError(t, err)

	// Mock stdin
	mockStdin := new(MockWriter)
	mockStdin.On("Write", mock.Anything).Return(100, nil) // Assume query length is 100
	client.stdin = mockStdin

	mockResult := `[{"action":"CREATED","count":10}]`
	mockStdout := NewMockReader([]byte(mockResult))
	client.stdout = mockStdout

	since := time.Now().Add(-24 * time.Hour)
	summary, err := client.GetFileChangesSummary(since)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Len(t, summary, 1)
	assert.Equal(t, float64(10), summary[0]["count"])

	mockStdin.AssertExpectations(t)
}

func TestClose(t *testing.T) {
	mockLogger, err := logger.NewLogger(logger.Config{})
	assert.NoError(t, err)

	configPath := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configPath)

	client, err := New(configPath, WithLogger(mockLogger), WithOsqueryBinary("echo"))
	assert.NoError(t, err)

	client.cmd = exec.Command("true")

	mockStdin := new(MockWriter)
	mockStdin.On("Close").Return(nil)
	client.stdin = mockStdin

	mockStdout := new(MockReader)
	mockStdout.On("Close").Return(nil)
	client.stdout = mockStdout

	mockStderr := new(MockReader)
	mockStderr.On("Close").Return(nil)
	client.stderr = mockStderr

	err = client.Close()
	assert.NoError(t, err)

	mockStdin.AssertExpectations(t)
	mockStdout.AssertExpectations(t)
	mockStderr.AssertExpectations(t)
}

// Helper function to create a MockReader
func NewMockReader(data []byte) *MockReader {
	return &MockReader{
		Reader: bytes.NewReader(data),
	}
}
