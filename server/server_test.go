package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/tejiriaustin/savannah-assessment/models"
)

// MockDB is a mock implementation of db.Repository
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Close() error {
	return nil
}

func (m *MockDB) CreateFileEventsTable() error {
	return nil
}

func (m *MockDB) InsertFileEvent(event models.FileEvent) error {
	return nil
}

func (m *MockDB) GetFileEvents() ([]models.FileEvent, error) {
	args := m.Called()
	return args.Get(0).([]models.FileEvent), args.Error(1)
}

func TestRetrieveEvents(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Test cases
	testCases := []struct {
		name           string
		setupMock      func(*MockDB)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "Successful retrieval",
			setupMock: func(mockDB *MockDB) {
				mockDB.On("GetFileEvents").Return([]models.FileEvent{
					{ID: 1, Path: "/test/file1", Operation: "CREATE"},
					{ID: 2, Path: "/test/file2", Operation: "MODIFY"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []models.FileEvent{
				{ID: 1, Path: "/test/file1", Operation: "CREATE"},
				{ID: 2, Path: "/test/file2", Operation: "MODIFY"},
			},
		},
		{
			name: "Database error",
			setupMock: func(mockDB *MockDB) {
				mockDB.On("GetFileEvents").Return([]models.FileEvent{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   gin.H{"error": "database error"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock database
			mockDB := new(MockDB)
			tc.setupMock(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			s := &Server{}

			handler := s.retrieveEvents(mockDB)
			handler(c)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, response)

			mockDB.AssertExpectations(t)
		})
	}
}
