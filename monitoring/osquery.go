package monitoring

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tejiriaustin/savannah-assessment/logger"
)

type (
	OsQueryFIMClient struct {
		monitorDirs   []string
		configPath    string
		osqueryBinary string
		databasePath  string
		cmd           *exec.Cmd
		stdin         io.WriteCloser
		stdout        io.ReadCloser
		stderr        io.ReadCloser
		mutex         sync.Mutex
		log           *logger.Logger
		maxRetries    int
	}

	Config struct {
		Options   map[string]interface{} `json:"options,omitempty"`
		Schedule  map[string]interface{} `json:"schedule"`
		FilePaths map[string][]string    `json:"file_paths"`
	}

	Options func(*OsQueryFIMClient) error
)

var _ Monitor = (*OsQueryFIMClient)(nil)

func WithMonitorDirs(dirs []string) Options {
	return func(o *OsQueryFIMClient) error {
		o.monitorDirs = dirs
		return nil
	}
}

func WithOsqueryBinary(path string) Options {
	return func(o *OsQueryFIMClient) error {
		o.osqueryBinary = path
		return nil
	}
}

func WithDatabasePath(path string) Options {
	return func(o *OsQueryFIMClient) error {
		o.databasePath = path
		return nil
	}
}

func WithLogger(log *logger.Logger) Options {
	return func(o *OsQueryFIMClient) error {
		o.log = log
		return nil
	}
}

func WithMaxRetries(maxRetries int) Options {
	return func(o *OsQueryFIMClient) error {
		o.maxRetries = maxRetries
		return nil
	}
}

func New(configPath string, opts ...Options) (*OsQueryFIMClient, error) {
	client := &OsQueryFIMClient{
		configPath:    configPath,
		osqueryBinary: "osqueryi",
		databasePath:  "/var/tmp/osquery_data/osquery.db",
		maxRetries:    3,
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}
	if client.log == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return client, nil
}

func (c *OsQueryFIMClient) createConfig() error {
	config := map[string]interface{}{
		"schedule": map[string]interface{}{
			"file_events": map[string]interface{}{
				"query":    "SELECT * FROM file_events;",
				"interval": 300,
			},
		},
		"file_paths": map[string][]string{
			"homes": c.monitorDirs,
		},
		"etc": []string{
			"/etc/%%",
		},
		"tmp": []string{
			"/tmp/%%",
		},
	}

	jsonConfig, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(c.configPath, jsonConfig, 0644)
}

func (c *OsQueryFIMClient) Start(ctx context.Context) error {
	c.log.Info("Started file tracking...")

	if err := os.MkdirAll(filepath.Dir(c.databasePath), 0755); err != nil {
		c.log.Error("Failed to create database directory", "error", err)
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	c.cmd = exec.Command(c.osqueryBinary,
		"--config_path="+c.configPath,
		"--database_path="+c.databasePath,
		"--disable_events=false",
		"--enable_file_events=true",
		"--force",
		"--json")

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		c.log.Error("Failed to create stdin pipe", "error", err)
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		c.log.Error("Failed to create stdout pipe", "error", err)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		c.log.Error("Failed to create stderr pipe", "error", err)
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		c.log.Error("Failed to start osqueryi", "error", err)
		return fmt.Errorf("failed to start osqueryi: %w", err)
	}

	readyChan := make(chan struct{})
	go c.waitForStartup(ctx, readyChan)

	select {
	case <-readyChan:
		c.log.Info("Osquery started successfully")
	case <-ctx.Done():
		if err := c.cmd.Process.Kill(); err != nil {
			c.log.Error("Failed to kill osquery process on context cancellation", "error", err)
		}
		return ctx.Err()
	}

	go c.handleStderr(ctx)

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		c.log.Error("Osquery exited unexpectedly")
		return fmt.Errorf("osquery exited unexpectedly")
	}

	return nil
}

func (c *OsQueryFIMClient) waitForStartup(ctx context.Context, readyChan chan struct{}) {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Osquery started successfully") {
			close(readyChan)
			return
		}

		if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
			break
		}
	}

	close(readyChan)
}

func (c *OsQueryFIMClient) handleStderr(ctx context.Context) {
	scanner := bufio.NewScanner(c.stderr)
	retries := 0
	backoffSchedule := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	for scanner.Scan() {
		line := scanner.Text()
		c.log.Warn("osqueryi stderr output", "message", line)

		// Detect specific lock file error
		if strings.Contains(line, "IO error: While lock file") {
			c.log.Warn("Detected lock file error. Attempting to clear lock and restart...")

			// Retry logic for restarting osquery
			for retries < c.maxRetries {
				if err := c.Restart(ctx); err != nil {
					retries++
					c.log.Warn("Failed to restart osquery", "error", err, "retry", retries)

					// Determine appropriate backoff duration
					backoff := backoffSchedule[min(retries-1, len(backoffSchedule)-1)]
					c.log.Debug("Waiting before next retry", "backoff", backoff)

					select {
					case <-time.After(backoff):
						// Continue retrying after backoff
						continue
					case <-ctx.Done():
						c.log.Error("Context cancelled, stopping retry attempts", "retries", retries)
						return
					}
				} else {
					// Successfully restarted, reset retries and break the loop
					c.log.Info("Successfully restarted osquery after detecting lock file error", "retries", retries)
					retries = 0
					break
				}
			}

			// If maximum retries reached, log error and exit
			if retries == c.maxRetries {
				c.log.Error("Failed to restart osquery after maximum retries", "maxRetries", c.maxRetries)
				return
			}
		}
	}

	// Handle error from the scanner itself
	if err := scanner.Err(); err != nil {
		c.log.Error("Error reading from stderr", "error", err)
	}
}

func (c *OsQueryFIMClient) Query(query string) ([]map[string]interface{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stdin == nil {
		c.log.Error("stdin is nil, osquery may not be properly initialized")
		return nil, fmt.Errorf("stdin is nil, osquery may not be properly initialized")
	}

	results, err := sendCommand(c.stdin, c.stdout, query)
	if err != nil {
		c.log.Error("Failed to execute query", "query", query, "error", err)
		return nil, err
	}
	c.log.Info("Query executed successfully", "query", query, "results_count", len(results))
	return results, nil
}

func sendCommand(stdin io.Writer, stdout io.Reader, command string) ([]map[string]interface{}, error) {
	if _, err := fmt.Fprintln(stdin, command); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	decoder := json.NewDecoder(stdout)

	var results []map[string]interface{}
	for {
		var result []map[string]interface{}
		if err := decoder.Decode(&result); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		results = append(results, result...)
		break
	}

	return results, nil
}

func (c *OsQueryFIMClient) GetFileEvents() ([]map[string]interface{}, error) {
	return c.Query("SELECT * FROM file_events;")
}

func (c *OsQueryFIMClient) GetFileEventsByPath(path string, since time.Time) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM file_events WHERE path LIKE '%s%%' AND time > %d;", path, since.Unix())
	return c.Query(query)
}

func (c *OsQueryFIMClient) GetFileChangesSummary(since time.Time) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT 
			action, 
			COUNT(*) as count, 
			MIN(time) as first_occurrence, 
			MAX(time) as last_occurrence
		FROM file_events 
		WHERE time > %d 
		GROUP BY action;
	`, since.Unix())
	return c.Query(query)
}

func (c *OsQueryFIMClient) Restart(ctx context.Context) error {
	c.log.Info("Restarting osquery")
	if err := c.Stop(); err != nil {
		c.log.Error("Failed to stop osquery during restart", "error", err)
		return fmt.Errorf("failed to stop osquery: %w", err)
	}

	return c.Start(ctx)
}

func (c *OsQueryFIMClient) Stop() error {
	c.log.Info("Stopping osquery")
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			c.log.Error("Failed to kill osquery process", "error", err)
			return fmt.Errorf("failed to kill osquery process: %w", err)
		}
	}
	c.log.Info("Osquery stopped successfully")
	return nil
}

func (c *OsQueryFIMClient) UpdateOrCreateJSONFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	var config Config

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}

	if fileInfo.Size() > 0 {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("error parsing JSON: %v", err)
		}
	}

	// Update or add the "schedule" key with "file_events"
	if config.Schedule == nil {
		config.Schedule = make(map[string]interface{})
	}
	config.Schedule["file_events"] = map[string]interface{}{
		"query":    "SELECT * FROM file_events;",
		"interval": 300,
	}

	// Update or add the "file_paths" key
	if config.FilePaths == nil {
		config.FilePaths = make(map[string][]string)
	}

	// Preserve existing "homes" entries and add new one if not present
	existingHomes := config.FilePaths["homes"]
	for _, newEntry := range c.monitorDirs {
		entryExists := false
		for _, existingEntry := range existingHomes {
			if existingEntry == newEntry {
				entryExists = true
				break
			}
		}
		if !entryExists {
			existingHomes = append(existingHomes, newEntry)
		}
	}
	config.FilePaths["homes"] = existingHomes

	config.FilePaths["etc"] = []string{"/etc/%%"}
	config.FilePaths["tmp"] = []string{"/tmp/%%"}

	// Seek to the beginning of the file before writing
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking file: %v", err)
	}

	// Truncate the file to ensure we overwrite any existing content
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("error truncating file: %v", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("error writing JSON to file: %v", err)
	}

	c.log.Info(fmt.Sprintf("File %s has been updated or created successfully.\n", filePath))
	return nil
}
func (c *OsQueryFIMClient) Close() error {
	c.log.Info("Closing osquery client")
	if err := c.Stop(); err != nil {
		c.log.Error("Failed to stop osqueryi during close", "error", err)
		return fmt.Errorf("failed to stop osqueryi: %w", err)
	}

	if c.stdin != nil {
		if err := c.stdin.Close(); err != nil {
			c.log.Error("Failed to close stdin", "error", err)
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	}
	if c.stdout != nil {
		if err := c.stdout.Close(); err != nil {
			c.log.Error("Failed to close stdout", "error", err)
			return fmt.Errorf("failed to close stdout: %w", err)
		}
	}
	if c.stderr != nil {
		if err := c.stderr.Close(); err != nil {
			c.log.Error("Failed to close stderr", "error", err)
			return fmt.Errorf("failed to close stderr: %w", err)
		}
	}

	c.log.Info("Osquery client closed successfully")
	return nil
}
