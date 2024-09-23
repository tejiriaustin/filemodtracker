package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
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

	monitorClient, err := monitoring.New(cfg.OsqueryConfig, monitoring.WithMonitorDirs([]string{cfg.MonitoredDirectory}))
	if err != nil {
		log.Fatalf("failed to create monitoring client: %v", err)
		return
	}

	err = monitorClient.Start()
	if err != nil {
		log.Fatalf("failed to start monitoring client: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg      = sync.WaitGroup{}
		cmdChan = make(chan daemon.Command, 100)
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer(ctx, cfg, monitorClient, cmdChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startDaemon(ctx, cfg, monitorClient, cmdChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	log.Println("Shutdown signal received")

	cancel()

	waitChan := make(chan struct{})
	go func() {
		if err := monitorClient.Wait(); err != nil {
			fmt.Printf("osqueryi process exited with error: %v\n", err)
			return
		}
		if err := monitorClient.Close(); err != nil {
			fmt.Printf("osqueryi process exited with error: %v\n", err)
			return
		}

		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		log.Println("All goroutines finished")
	case <-time.After(3 * time.Second):
		log.Println("Shutdown timed out")
	}

	log.Println("Daemon service stopped")
}

func startServer(ctx context.Context, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Server shutting down")
			return
		default:

			s := server.New(cfg)
			s.Start(monitorClient, cmdChan)
		}
	}
}

func startDaemon(ctx context.Context, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan <-chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Server shutting down")
			return
		default:
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

func stopDaemon(cmd *cobra.Command, args []string) {
	switch runtime.GOOS {
	case "darwin", "linux":
		stopUnixDaemon()
	case "windows":
		stopWindowsDaemon()
	default:
		fmt.Printf("Unsupported operating system: %s\n", runtime.GOOS)
		os.Exit(1)
	}
}

func stopUnixDaemon() {
	// Option 1: Using pkill
	pkillCmd := exec.Command("pkill", "-f", "filemodtracker daemon")
	if err := pkillCmd.Run(); err != nil {
		fmt.Printf("Failed to stop daemon using pkill: %v\n", err)
		// Fallback to Option 2 if pkill fails
		stopUsingPgrep()
	} else {
		fmt.Println("Daemon stopped successfully using pkill.")
	}
}

func stopUsingPgrep() {
	// Option 2: Using pgrep and kill
	pgrepCmd := exec.Command("pgrep", "-f", "filemodtracker daemon")
	output, err := pgrepCmd.Output()
	if err != nil {
		fmt.Printf("Failed to find daemon process: %v\n", err)
		os.Exit(1)
	}

	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		fmt.Println("No running daemon found.")
		return
	}

	for _, pid := range pids {
		killCmd := exec.Command("kill", pid)
		if err := killCmd.Run(); err != nil {
			fmt.Printf("Failed to stop daemon process %s: %v\n", pid, err)
		} else {
			fmt.Printf("Sent termination signal to process %s\n", pid)
		}
	}
	fmt.Println("Daemon stop command executed.")
}

func stopWindowsDaemon() {
	cmd := exec.Command("taskkill", "/F", "/IM", "filemodtracker.exe")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to stop daemon: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Daemon stopped successfully.")
}
