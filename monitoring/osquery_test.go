package monitoring

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockServer struct {
	mock.Mock
}

func (m *MockServer) RegisterPlugin(plugin *table.Plugin) {
	m.Called(plugin)
}

func (m *MockServer) Run() error {
	args := m.Called()
	return args.Error(0)
}

type OsQueryClientTestSuite struct {
	suite.Suite
	mockServer *MockServer
	client     OsQueryClient
}

func (suite *OsQueryClientTestSuite) SetupTest() {
	suite.mockServer = new(MockServer)

	suite.client = OsQueryClient{
		monitorDir: "/tmp",
		resultsBuilder: func(file os.FileInfo) map[string]string {
			return map[string]string{
				"name":      file.Name(),
				"action":    strings.ToLower(filepath.Ext(file.Name())),
				"time":      file.ModTime().Format(time.RFC3339),
				"directory": filepath.Dir(file.Name()),
				"size":      strconv.FormatInt(file.Size(), 10),
			}
		},
	}
}

func (suite *OsQueryClientTestSuite) TestStart() {
	// Define the test cases
	testCases := []struct {
		name       string
		mockRunErr error
		expectErr  string
	}{
		{
			name:       "Successful StartMonitoring",
			mockRunErr: nil,
			expectErr:  "",
		},
		{
			name:       "Server Run Error",
			mockRunErr: errors.New("server run error"),
			expectErr:  "server run error",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockServer.On("RegisterPlugin", mock.AnythingOfType("*table.Plugin")).Return()
			suite.mockServer.On("Run").Return(tc.mockRunErr)

			err := suite.client.StartMonitoring("test_table", map[string]string{"column1": "TEXT", "column2": "TEXT"})

			if tc.expectErr != "" {
				suite.EqualError(err, tc.expectErr)
			} else {
				suite.NoError(err)
			}

			suite.mockServer.AssertCalled(suite.T(), "RegisterPlugin", mock.AnythingOfType("*table.Plugin"))
			suite.mockServer.AssertCalled(suite.T(), "Run")
		})
	}
}

func (suite *OsQueryClientTestSuite) TearDownTest() {
	suite.mockServer.ExpectedCalls = nil
}

func TestOsQueryClientTestSuite(t *testing.T) {
	suite.Run(t, new(OsQueryClientTestSuite))
}
