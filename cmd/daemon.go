package cmd

import (
	"context"
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
	"github.com/tejiriaustin/savannah-assessment/logger"
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

	logCfg := logger.Config{
		LogLevel: "info",
		DevMode:  true,
	}
	log, err := logger.NewLogger(logCfg)
	if err != nil {
		log.Errorf("Failed to create logger: %v", err)
	}
	defer log.Sync()

	if err := os.WriteFile(cfg.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Fatal("Failed to write PID file", "error", err)
	}
	defer func() {
		if err := os.Remove(cfg.PidFile); err != nil {
			log.Error("Failed to remove PID file", "error", err)
		}
	}()

	monitorClient, err := monitoring.NewOsQueryFIMClient(cfg.OsquerySocket)
	if err != nil {
		log.Error("Failed to create monitoring client", "error", err)
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
		startServer(ctx, log, cfg, monitorClient, cmdChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startDaemon(ctx, log, cfg, monitorClient, cmdChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	log.Info("Shutdown signal received")

	cancel()

	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		log.Info("All goroutines finished")
	case <-time.After(3 * time.Second):
		log.Warn("Shutdown timed out")
	}

	log.Info("Daemon service stopped")
}

func startServer(ctx context.Context, log *logger.Logger, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Server shutting down")
			return
		default:
			s := server.New(cfg)
			s.Start(log, monitorClient, cmdChan)
		}
	}
}

func startDaemon(ctx context.Context, log *logger.Logger, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan <-chan daemon.Command) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Daemon shutting down")
			return
		default:
			d, err := daemon.New(cfg, log, monitorClient, cmdChan)
			if err != nil {
				log.Error("Failed to create daemon", "error", err)
				return
			}

			go func() {
				if err := d.StartDaemon(ctx); err != nil {
					log.Error("Failed to start daemon", "error", err)
				}
			}()
		}
	}
}

func stopDaemon(cmd *cobra.Command, args []string) {
	logCfg := logger.Config{
		LogLevel: "info",
		DevMode:  true,
	}
	log, err := logger.NewLogger(logCfg)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	switch runtime.GOOS {
	case "darwin", "linux":
		stopUnixDaemon(log)
	case "windows":
		stopWindowsDaemon(log)
	default:
		log.Error("Unsupported operating system", "os", runtime.GOOS)
		os.Exit(1)
	}
}

func stopUnixDaemon(log *logger.Logger) {
	pkillCmd := exec.Command("pkill", "-f", "filemodtracker daemon")
	if err := pkillCmd.Run(); err != nil {
		log.Warn("Failed to stop daemon using pkill", "error", err)
		stopUsingPgrep(log)
	} else {
		log.Info("Daemon stopped successfully using pkill")
	}
}

func stopUsingPgrep(log *logger.Logger) {
	pgrepCmd := exec.Command("pgrep", "-f", "filemodtracker daemon")
	output, err := pgrepCmd.Output()
	if err != nil {
		log.Error("Failed to find daemon process", "error", err)
		os.Exit(1)
	}

	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		log.Info("No running daemon found")
		return
	}

	for _, pid := range pids {
		killCmd := exec.Command("kill", pid)
		if err := killCmd.Run(); err != nil {
			log.Error("Failed to stop daemon process", "pid", pid, "error", err)
		} else {
			log.Info("Sent termination signal to process", "pid", pid)
		}
	}
	log.Info("Daemon stop command executed")
}

func stopWindowsDaemon(log *logger.Logger) {
	cmd := exec.Command("taskkill", "/F", "/IM", "filemodtracker.exe")
	if err := cmd.Run(); err != nil {
		log.Error("Failed to stop daemon", "error", err)
		os.Exit(1)
	}
	log.Info("Daemon stopped successfully")
}
