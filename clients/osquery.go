package clients

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/models"
)

var fileEvents []models.FileEvent

func StartExtension(socketPath string) (*osquery.ExtensionManagerClient, error) {
	server, err := osquery.NewExtensionManagerServer("file_monitor", socketPath)
	if err != nil {
		return nil, fmt.Errorf("error creating extension: %w", err)
	}

	fileEventsTable := table.NewPlugin("file_events", FileEventsColumns(), FileEventsGenerate)
	server.RegisterPlugin(fileEventsTable)

	go func() {
		if err := server.Run(); err != nil {
			fmt.Printf("Error running server: %v\n", err)
		}
	}()

	// Wait for server to become available
	client, err := osquery.NewClient(socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error creating osquery client: %w", err)
	}

	return client, nil
}

func FileEventsColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("path"),
		table.TextColumn("operation"),
		table.TextColumn("timestamp"),
	}
}

func FileEventsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	for _, event := range fileEvents {
		results = append(results, map[string]string{
			"path":      event.Path,
			"operation": event.Operation,
			"timestamp": event.Timestamp.Format(time.RFC3339),
		})
	}

	return results, nil
}

func AddFileEvent(event models.FileEvent) {
	fileEvents = append(fileEvents, event)
}

func EnsureOsqueryRunning() error {
	cmd := exec.Command("pgrep", "osqueryd")
	if err := cmd.Run(); err != nil {
		fmt.Println("Osquery is not running. Attempting to start...")
		startCmd := exec.Command("sudo", "systemctl", "start", "osqueryd")
		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start osquery: %v", err)
		}
		fmt.Println("Osquery started successfully.")
	} else {
		fmt.Println("Osquery is already running.")
	}
	return nil
}

func ConnectToOsquery(cfg *config.Config) (*osquery.ExtensionManagerClient, error) {
	timeout := time.After(30 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timed out waiting for osquery socket")
		case <-tick:
			if _, err := os.Stat(cfg.OsquerySocket); os.IsNotExist(err) {
				fmt.Printf("Waiting for osquery socket at %s...\n", cfg.OsquerySocket)
				continue
			}

			client, err := osquery.NewClient(cfg.OsquerySocket, 3*time.Second)
			if err != nil {
				fmt.Printf("Error connecting to osquery: %v. Retrying...\n", err)
				continue
			}

			return client, nil
		}
	}
}
