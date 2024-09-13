package monitoring

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/config"
	"github.com/osquery/osquery-go/plugin/table"
)

type (
	OsQueryClient struct {
		monitorDir     string
		socket         string
		extensionName  string
		timeout        time.Duration
		interval       time.Duration
		resultsBuilder func(file os.FileInfo) map[string]string
	}
	Options func(*OsQueryClient) error
)

const (
	defaultTimeout  = 3
	defaultInterval = 3
)

func WithTimeout(timeout time.Duration) Options {
	return func(o *OsQueryClient) error {
		o.timeout = timeout
		return nil
	}
}

func WithInterval(interval time.Duration) Options {
	return func(o *OsQueryClient) error {
		o.interval = interval
		return nil
	}
}

func WithResultsBuilder() Options {
	return func(o *OsQueryClient) error {
		o.resultsBuilder = func(file os.FileInfo) map[string]string {
			return map[string]string{
				"name":      file.Name(),
				"action":    strings.ToLower(filepath.Ext(file.Name())),
				"time":      file.ModTime().Format(time.RFC3339),
				"directory": filepath.Dir(file.Name()),
				"size":      strconv.FormatInt(file.Size(), 10),
			}
		}
		return nil
	}
}

func newOsQueryClient() *OsQueryClient {
	return &OsQueryClient{
		timeout:  defaultTimeout,
		interval: defaultInterval,
	}
}

func New(extensionName, socket string, opts ...Options) (*OsQueryClient, error) {
	osQueryClient := newOsQueryClient()

	osQueryClient.socket = socket
	osQueryClient.extensionName = extensionName

	for _, option := range opts {
		err := option(osQueryClient)
		if err != nil {
			return nil, err
		}
	}

	return osQueryClient, nil
}

func EnsureOsqueryExists(osquerySocketPath, osqueryConfigPath string) error {
	//if _, err := os.Stat(osquerySocketPath); err == nil {
	//	log.Println("osquery is already running")
	//	return nil
	//}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("osqueryi",
			"--pidfile", `C:\ProgramData\osquery\osqueryi.pid`,
			"--database_path", `C:\ProgramData\osquery\osquery.db`,
			"--extensions_socket", osquerySocketPath,
			"--config_path", osqueryConfigPath,
			"--disable_events=false",
			"--enable_file_events=true",
			"--disable_audit=false",
			"--enable_ntfs_event_publisher=true",
			"--force")
	default: // Unix-like systems (Linux, macOS)
		cmd = exec.Command("osqueryi",
			"--pidfile", "/var/run/osquery.pid",
			"--database_path", "/var/osquery/osquery.db",
			"--extensions_socket", osquerySocketPath,
			"--config_path", osqueryConfigPath,
			"--disable_events=false",
			"--enable_file_events=true",
			"--disable_audit=false",
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

func GenerateColumns(columnDefs map[string]string) func() []table.ColumnDefinition {
	return func() []table.ColumnDefinition {
		columns := make([]table.ColumnDefinition, 0, len(columnDefs))

		for colName, colType := range columnDefs {
			switch strings.ToLower(colType) {
			case "text":
				columns = append(columns, table.TextColumn(colName))
			case "integer":
				columns = append(columns, table.IntegerColumn(colName))
			case "bigint":
				columns = append(columns, table.BigIntColumn(colName))
			case "double":
				columns = append(columns, table.DoubleColumn(colName))
			default:
				columns = append(columns, table.TextColumn(colName))
			}
		}

		return columns
	}
}

func TableGenerator(dirToScan string,
	resultsBuilder func(file os.FileInfo) map[string]string,
) func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {

	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		var results []map[string]string

		err := filepath.Walk(dirToScan, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			results = append(results, resultsBuilder(info))

			return nil
		})

		if err != nil {
			return nil, err
		}

		return results, nil
	}
}

func (c *OsQueryClient) StartMonitoring(tableName string, columnDefinitions map[string]string, opts ...Options) error {
	serverTimeout := osquery.ServerTimeout(
		time.Second * c.timeout,
	)

	serverPingInterval := osquery.ServerPingInterval(
		time.Second * c.interval,
	)

	server, err := osquery.NewExtensionManagerServer(
		c.extensionName,
		c.socket,
		serverTimeout,
		serverPingInterval,
	)
	if err != nil {
		return err
	}

	columns := GenerateColumns(columnDefinitions)
	generator := TableGenerator(c.monitorDir, c.resultsBuilder)

	log.Println("starting monitoring service...")
	server.RegisterPlugin(table.NewPlugin(tableName, columns(), generator))
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (c *OsQueryClient) Query(query string) ([]map[string]string, error) {
	client, err := osquery.NewClient(c.extensionName, c.timeout*time.Second)
	if err != nil {
		return nil, err
	}

	response, err := client.Query(query)
	if err != nil {
		log.Printf("Error querying osquery: %v", err)
		return nil, err
	}

	for _, row := range response.Response {
		log.Println(row)
	}
	return response.Response, nil
}

func (c *OsQueryClient) StartConfigServer(configName string, GenerateConfigs func(ctx context.Context) (map[string]string, error)) error {
	serverTimeout := osquery.ServerTimeout(
		time.Second * c.timeout,
	)

	serverPingInterval := osquery.ServerPingInterval(
		time.Second * c.interval,
	)

	server, err := osquery.NewExtensionManagerServer(
		c.extensionName,
		c.socket,
		serverTimeout,
		serverPingInterval,
	)
	if err != nil {
		return err
	}

	log.Println("starting config server...")
	server.RegisterPlugin(config.NewPlugin(configName, GenerateConfigs))
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}
