// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
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
	log.Info("Starting File Modification Tracker daemon")

	monitorClient, err := monitoring.New(cfg.OsquerySocket, monitoring.WithLogger(log))
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
			log.Error("Error in daemon service", "error", err)
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
		log.Error("Failed to start daemon", "error", err)
		return fmt.Errorf("failed to start daemon: %v", err)
	}
	return nil
}

func stopDaemon(cmd *cobra.Command, args []string) {
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
	}
	log.Info("Daemon stopped successfully using pkill")
}

func stopUsingPgrep(log *logger.Logger) {
	pgrepCmd := exec.Command("pgrep", "-f", "filemodtracker stop")
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
