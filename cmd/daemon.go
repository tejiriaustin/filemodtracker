package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/daemon"
	"github.com/tejiriaustin/savannah-assessment/monitoring"
	"github.com/tejiriaustin/savannah-assessment/server"
)

var serviceCmd = &cobra.Command{
	Use:   "daemon",
	Short: "StartMonitoring the File Modification Tracker daemon",
	Run:   startDaemonService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}

func startDaemonService(cmd *cobra.Command, args []string) {
	cfg := config.GetConfig()

	if err := os.WriteFile(cfg.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer func() {
		if err := os.Remove(cfg.PidFile); err != nil {
			log.Printf("Failed to remove PID file: %v", err)
		}
	}()

	if err := monitoring.EnsureOsqueryExists(cfg.OsquerySocket, cfg.MonitorDir); err != nil {
		log.Fatalf("Failed to ensure osquery is monitoring file events: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	cmdChan := make(chan daemon.Command, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer(ctx, cfg, cmdChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startDaemon(ctx, cfg, cmdChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	log.Println("Shutdown signal received")

	cancel()

	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		log.Println("All goroutines finished")
	case <-time.After(10 * time.Second):
		log.Println("Shutdown timed out")
	}

	log.Println("Daemon service stopped")
}

func startServer(ctx context.Context, cfg *config.Config, cmdChan chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Server shutting down")
			return
		default:
			monitorClient, err := monitoring.New("file_events", cfg.OsquerySocket)
			if err != nil {
				log.Fatalf("failed to create monitoring client: %v", err)
				return
			}
			s := server.New(cfg)
			s.Start(monitorClient, cmdChan)
		}
	}
}

func startDaemon(ctx context.Context, cfg *config.Config, cmdChan <-chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Server shutting down")
			return
		default:
			monitorClient, err := monitoring.New("file_events", cfg.OsquerySocket, monitoring.WithResultsBuilder())
			if err != nil {
				log.Fatalf("failed to create monitoring client: %v", err)
				return
			}

			go func() {
				err := monitorClient.StartConfigServer("monitoring_config", GenerateConfigs(cfg.MonitorDir))
				if err != nil {
					log.Println("StartConfigServer error: ", err)
				}
			}()

			d, err := daemon.New(cfg, monitorClient, cmdChan)
			if err != nil {
				log.Fatalf("Failed to create daemon: %v", err)
				return
			}
			if err := d.StartDaemon(); err != nil {
				log.Fatalf("Failed to start daemon: %v", err)
			}
		}
	}
}

func GenerateConfigs(dirToMonitor string) func(ctx context.Context) (map[string]string, error) {
	return func(ctx context.Context) (map[string]string, error) {
		return map[string]string{
			"config1": `
{
  "schedule": {
    "crontab": {
      "query": "SELECT * FROM crontab;",
      "interval": 300
    },
    "file_events": {
      "query": "SELECT * FROM file_events;",
      "removed": false,
      "interval": 300
    }
  },
  "file_paths": {
    "homes": [
      "` + dirToMonitor + `",
    ],
    "etc": [
      "/etc/%%"
    ],
    "tmp": [
      "/tmp/%%"
    ]
  },
  "exclude_paths": {
    "homes": [
      "/home/not_to_monitor/.ssh/%%"
    ],
    "tmp": [
      "/tmp/too_many_events/"
    ]
  }
}
`,
		}, nil
	}
}

func stopDaemon(cmd *cobra.Command, args []string) {
	cfg := config.GetConfig()

	pidBytes, err := ioutil.ReadFile(cfg.PidFile)
	if err != nil {
		fmt.Printf("Error reading PID file: %v\n", err)
		os.Exit(1)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		fmt.Printf("Error parsing PID: %v\n", err)
		os.Exit(1)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Error finding process: %v\n", err)
		os.Exit(1)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		fmt.Printf("Error sending termination signal: %v\n", err)
		os.Exit(1)
	}

	err = os.Remove(cfg.PidFile)
	if err != nil {
		fmt.Printf("Warning: Unable to remove PID file: %v\n", err)
	}

	fmt.Println("Daemon stopped successfully.")
}
