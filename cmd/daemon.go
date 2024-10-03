// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
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
	log.Info("Starting File Modification Tracker daemon")

	pid := os.Getpid()
	if err := cfg.WritePidFile(pid); err != nil {
		log.Error("Failed to write PID file", "error", err)
		os.Exit(1)
	}

	monitorClient, err := monitoring.New(cfg.OsqueryConfig, monitoring.WithLogger(log))
	if err != nil {
		log.Fatal("Failed to create monitoring client", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var (
		wg      sync.WaitGroup
		errChan = make(chan error, 2)
		cmdChan = make(chan daemon.Command, 100)
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := startServer(ctx, log, cfg, monitorClient, cmdChan); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := startDaemon(ctx, log, cfg, monitorClient, cmdChan); err != nil {
			errChan <- fmt.Errorf("daemon error: %w", err)
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil {
			log.Info("Error in daemon service", "error", err)
		}
	case sig := <-sigChan:
		log.Info("Shutdown signal received", "signal", sig)
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All goroutines finished")
	case <-shutdownCtx.Done():
		log.Info("Shutdown timed out")
	}

	log.Info("Daemon service stopped")
}

func startServer(ctx context.Context, log *logger.Logger, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan chan daemon.Command) error {

	h := server.NewHandler(log).SetupHandler(monitorClient, cmdChan)

	err := server.New(cfg, log).Start(h)
	if err != nil {
		return err
	}

	<-ctx.Done()
	log.Info("Server shutting down")
	return nil
}

func startDaemon(ctx context.Context, log *logger.Logger, cfg *config.Config, monitorClient monitoring.Monitor, cmdChan <-chan daemon.Command) error {
	d, err := daemon.New(cfg, log, monitorClient, cmdChan)
	if err != nil {
		log.Error("Failed to create daemon", "error", err)
		return fmt.Errorf("failed to create daemon: %v", err)
	}

	if err := d.StartDaemon(ctx); err != nil {
		log.Info("Failed to start daemon", "error", err)
		return fmt.Errorf("failed to start daemon: %v", err)
	}
	return nil
}

func stopDaemon(cmd *cobra.Command, args []string) {
	cfg := config.GetConfig()
	switch runtime.GOOS {
	case "darwin", "linux":
		stopUnixDaemon(cfg, log)
	case "windows":
		stopWindowsDaemon(cfg, log)
	default:
		log.Error("Unsupported operating system", "os", runtime.GOOS)
		os.Exit(1)
	}
}

func stopUnixDaemon(cfg *config.Config, log *logger.Logger) {
	pid, err := cfg.ReadPidFile()
	if err != nil {
		log.Info("Failed to read PID file", "error", err)
		os.Exit(1)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		log.Error("Failed to find process", "pid", pid, "error", err)
		err = cfg.RemovePidFile()
		if err != nil {
			log.Info("Failed to remove PidFile", err.Error())
			return
		}
		os.Exit(1)
	}

	// Try SIGTERM first
	if err := process.Signal(os.Interrupt); err != nil {
		log.Info("Failed to stop daemon using SIGTERM", "pid", pid, "error", err)

		// If SIGTERM fails, try SIGKILL
		if err := process.Kill(); err != nil {
			log.Info("Failed to stop daemon using SIGKILL", "pid", pid, "error", err)
			os.Exit(1)
		}
		log.Info("Daemon stopped using SIGKILL", "pid", pid)
	} else {
		log.Info("Daemon stopped using SIGTERM", "pid", pid)
	}

	err = cfg.RemovePidFile()
	if err != nil {
		log.Info("Failed to remove PidFile", err.Error())
		return
	}
}

func stopWindowsDaemon(cfg *config.Config, log *logger.Logger) {
	pid, err := cfg.ReadPidFile()
	if err != nil {
		log.Error("Failed to read PID file", "error", err)
		os.Exit(1)
	}

	cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to stop daemon", "pid", pid, "error", err, "output", string(output))
		os.Exit(1)
	}
	log.Info("Daemon stopped successfully", "pid", pid, "output", string(output))

	err = cfg.RemovePidFile()
	if err != nil {
		log.Info("Failed to remove PidFile", err.Error())
		return
	}
}
