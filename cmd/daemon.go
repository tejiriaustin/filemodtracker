package cmd

import (
	"github.com/tejiriaustin/savannah-assessment/clients"
	"log"

	"github.com/spf13/cobra"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/daemon"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/server"
)

var serviceCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the File Modification Tracker daemon",
	Run:   startDaemonService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}

func startDaemonService(cmd *cobra.Command, args []string) {
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

	cmdChan := make(chan string, 100)

	d, err := daemon.New(cfg, dbClient, cmdChan)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	go func() {
		if err := d.Start(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}
	}()

	s := server.New(cfg)
	s.Start(dbClient, cmdChan)

	if err := d.Stop(); err != nil {
		log.Printf("Failed to stop daemon: %v", err)
	}
}
