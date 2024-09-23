package monitoring

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

func New(configPath string, opts ...Options) (*OsQueryFIMClient, error) {
	client := &OsQueryFIMClient{
		configPath:    configPath,
		osqueryBinary: "osqueryi",
		databasePath:  "/var/tmp/osquery_data/osquery.db",
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
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

//func (c *OsQueryFIMClient) clearDatabaseLock() error {
//	lockFile := filepath.Join(c.databasePath, "LOCK")
//	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
//		return fmt.Errorf("failed to remove database lock file: %w", err)
//	}
//	return nil
//}

func (c *OsQueryFIMClient) Start() error {
	if err := c.createConfig(); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(c.databasePath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	//if err := c.clearDatabaseLock(); err != nil {
	//	return err
	//}

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
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start osqueryi: %w", err)
	}

	go c.handleStderr()

	time.Sleep(2 * time.Second)

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return fmt.Errorf("osquery exited unexpectedly")
	}

	return nil
}

func (c *OsQueryFIMClient) handleStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(os.Stderr, "osqueryi: %s\n", line)

		if strings.Contains(line, "IO error: While lock file") {
			fmt.Println("Detected lock file error. Attempting to clear lock and restart...")
			if err := c.Restart(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to restart osquery: %v\n", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stderr: %v\n", err)
	}
}

func (c *OsQueryFIMClient) Query(query string) ([]map[string]interface{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return sendCommand(c.stdin, c.stdout, query)
}

func sendCommand(stdin io.Writer, stdout io.Reader, command string) ([]map[string]interface{}, error) {
	if _, err := fmt.Fprintln(stdin, command); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Create a JSON decoder
	decoder := json.NewDecoder(stdout)

	// Read the JSON array from the output
	var results []map[string]interface{}
	for {
		var result []map[string]interface{}
		if err := decoder.Decode(&result); err == io.EOF {
			// End of input
			break
		} else if err != nil {
			return nil, err
		}

		// Append the results to our results slice
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

func (c *OsQueryFIMClient) Restart() error {
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop osquery: %w", err)
	}

	//if err := c.clearDatabaseLock(); err != nil {
	//	return err
	//}

	return c.Start()
}

func (c *OsQueryFIMClient) Stop() error {
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill osquery process: %w", err)
		}
	}
	return nil
}

func (c *OsQueryFIMClient) Wait() error {
	return c.cmd.Wait()
}

func (c *OsQueryFIMClient) Close() error {
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop osqueryi: %w", err)
	}

	if c.stdin != nil {
		if err := c.stdin.Close(); err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	}
	if c.stdout != nil {
		if err := c.stdout.Close(); err != nil {
			return fmt.Errorf("failed to close stdout: %w", err)
		}
	}
	if c.stderr != nil {
		if err := c.stderr.Close(); err != nil {
			return fmt.Errorf("failed to close stderr: %w", err)
		}
	}

	return c.Wait()
}
