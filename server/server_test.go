package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/daemon"
)

// MockMonitor is a mock implementation of the monitoring.Monitor interface
type MockMonitor struct {
	mock.Mock
}

func (m *MockMonitor) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMonitor) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMonitor) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMonitor) GetFileEvents() ([]map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]map[string]interface{}), args.Error(0)
}

func (m *MockMonitor) GetFileEventsByPath(path string, since time.Time) ([]map[string]interface{}, error) {
	args := m.Called(path, since)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockMonitor) GetFileChangesSummary(since time.Time) ([]map[string]interface{}, error) {
	args := m.Called(since)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func TestNew(t *testing.T) {
	cfg := &config.Config{Port: ":8080"}
	server := New(cfg)
	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.cfg)
	assert.Equal(t, ":8080", server.server.Addr)
}

func TestServer_Endpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMonitor := new(MockMonitor)
	mockEvents := []map[string]string{{"dummy_event": "some_event"}}
	mockMonitor.On("GetFileEvents").Return(mockEvents, nil)

	cmdChan := make(chan daemon.Command, 1)
	server := &Server{cfg: &config.Config{}}
	router := server.setupRouter(mockMonitor, cmdChan)

	tests := []struct {
		name           string
		method         string
		url            string
		body           interface{}
		expectedStatus int
		expectedBody   interface{}
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Health Check",
			method:         "GET",
			url:            "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"status": "alive and well"},
		},
		{
			name:           "Valid Command",
			method:         "POST",
			url:            "/command",
			body:           gin.H{"command": "ls -l"},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"status": "command received"},
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				cmd := <-cmdChan
				assert.Equal(t, "ls", cmd.Command)
				assert.Equal(t, []string{"-l"}, cmd.Args)
			},
		},
		{
			name:           "Empty Command",
			method:         "POST",
			url:            "/command",
			body:           gin.H{"command": ""},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "Key: 'Command' Error:Field validation for 'Command' failed on the 'required' tag"},
		},
		{
			name:           "Invalid Command",
			method:         "POST",
			url:            "/command",
			body:           gin.H{"command": "rm -rf /"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "base command not allowed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request

			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.url, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.url, nil)
			}

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}

	mockMonitor.AssertExpectations(t)
}

func TestServer_setupRouter(t *testing.T) {
	server := &Server{cfg: &config.Config{}}
	mockMonitor := new(MockMonitor)
	cmdChan := make(chan daemon.Command, 1)

	router := server.setupRouter(mockMonitor, cmdChan)

	assert.NotNil(t, router)
	assert.Len(t, router.Routes(), 3)
}
