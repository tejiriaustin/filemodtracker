package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/tejiriaustin/savannah-assessment/clients"
	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/daemon"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/server"
)

func start() {
	cfg := config.GetConfig()

	osqClient, err := clients.New(cfg.OsquerySocket, cfg.OsqueryConfigPath, clients.WithMonitorDir(cfg.MonitorDir))
	if err != nil {
		log.Fatalf("Failed to create osquery client: %v", err)
	}
	defer osqClient.Close()

	if err := osqClient.EnsureFileEventMonitoring(cfg.OsquerySocket, cfg.MonitorDir); err != nil {
		log.Fatalf("Failed to ensure osquery is monitoring file events: %v", err)
	}

	dbClient, err := db.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create database client: %v", err)
	}
	defer dbClient.Close()

	// Create command channel for communicating between server and daemon
	cmdChan := make(chan string, 100)

	// Create daemon
	d, err := daemon.New(cfg, dbClient, osqClient, cmdChan)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Create server
	s := server.New(cfg)

	// Use a WaitGroup to manage goroutines
	var wg sync.WaitGroup
	wg.Add(2) // One for daemon, one for server

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start daemon
	go func() {
		defer wg.Done()
		if err := d.Start(); err != nil {
			log.Printf("Daemon stopped with error: %v", err)
			cancel()
		}
	}()

	// Start server
	s.Start(dbClient, cmdChan)
	log.Println("Server has stopped")
	cancel() // Cancel context when server stops

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either a signal or context cancellation
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Initiate graceful shutdown
	log.Println("Initiating graceful shutdown...")

	// Stop the daemon
	if err := d.Stop(); err != nil {
		log.Printf("Error stopping daemon: %v", err)
	}

	// Wait for both goroutines to finish
	wg.Wait()

	log.Println("Graceful shutdown completed")
}
