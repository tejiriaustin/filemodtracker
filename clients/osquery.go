package clients

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/osquery/osquery-go"
)

type (
	OsQueryClient struct {
		monitorDir        string
		osquerySocketPath string
		osqueryConfigPath string
		osQuery           *osquery.ExtensionManagerClient
	}

	Option func(s *OsQueryClient) error
)

func New(osquerySocketPath, osqueryConfigPath string, opts ...Option) (*OsQueryClient, error) {
	client, err := osquery.NewClient(osquerySocketPath, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create osquery client: %v", err)
	}

	s := &OsQueryClient{
		osQuery:           client,
		osquerySocketPath: osquerySocketPath,
		osqueryConfigPath: osqueryConfigPath,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			client.Close()
			return nil, err
		}
	}

	return s, nil
}

func WithMonitorDir(dir string) Option {
	return func(s *OsQueryClient) error {
		s.monitorDir = dir
		return nil
	}
}

func (s *OsQueryClient) Close() {
	if s.osQuery != nil {
		s.osQuery.Close()
	}
}

func (s *OsQueryClient) MonitorFiles(db *sql.DB) error {
	query := fmt.Sprintf("SELECT path, action, time FROM file_events WHERE path LIKE '%s%%'", s.monitorDir)
	resp, err := s.osQuery.Query(query)
	if err != nil {
		return fmt.Errorf("error querying osquery: %v", err)
	}

	for _, row := range resp.Response {
		path := row["path"]
		action := row["action"]
		timestamp := row["time"]

		t, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			log.Printf("Error parsing timestamp: %v", err)
			continue
		}

		_, err = db.Exec("INSERT INTO file_events (path, action, timestamp) VALUES (?, ?, ?)", path, action, t)
		if err != nil {
			log.Printf("Error inserting into database: %v", err)
		}
	}

	return nil
}

func (s *OsQueryClient) EnsureFileEventMonitoring(osquerySocketPath, osqueryConfigPath string) error {
	if _, err := os.Stat(osquerySocketPath); err == nil {
		log.Println("osquery is already running")
		return nil
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("osqueryd",
			"--pidfile", `C:\ProgramData\osquery\osqueryd.pid`,
			"--database_path", `C:\ProgramData\osquery\osquery.db`,
			"--extensions_socket", osquerySocketPath,
			"--config_path", osqueryConfigPath,
			"--force")
	default: // Unix-like systems (Linux, macOS)
		cmd = exec.Command("osqueryd",
			"--pidfile", "/var/run/osquery.pid",
			"--database_path", "/var/osquery/osquery.db",
			"--extensions_socket", osquerySocketPath,
			"--config_path", osqueryConfigPath,
			"--force")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start osquery: %v", err)
	}

	log.Println("osquery started successfully")

	for i := 0; i < 10; i++ {
		if _, err := os.Stat(osquerySocketPath); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("osquery socket file/pipe not created within the expected time")
}

func (s *OsQueryClient) GetFileEvents() ([]struct {
	Path      string
	Action    string
	Timestamp time.Time
}, error) {
	query := fmt.Sprintf("SELECT path, action, time FROM file_events WHERE path LIKE '%s%%' AND time > (SELECT unix_time FROM time) - 10", s.monitorDir)
	resp, err := s.osQuery.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying osquery: %v", err)
	}

	var events []struct {
		Path      string
		Action    string
		Timestamp time.Time
	}

	for _, row := range resp.Response {
		timestamp, err := time.Parse("2006-01-02 15:04:05", row["time"])
		if err != nil {
			log.Printf("Error parsing timestamp: %v", err)
			continue
		}

		events = append(events, struct {
			Path      string
			Action    string
			Timestamp time.Time
		}{
			Path:      row["path"],
			Action:    row["action"],
			Timestamp: timestamp,
		})
	}

	return events, nil
}
