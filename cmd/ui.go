// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/monitoring"
	"github.com/tejiriaustin/savannah-assessment/ui"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "StartMonitoring the File Modification Tracker UI",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()

		if err := monitoring.EnsureOsqueryExists(cfg.OsquerySocket, cfg.MonitorDir); err != nil {
			log.Fatalf("Failed to ensure osquery is monitoring file events: %v", err)
		}

		monitorClient, err := monitoring.New("file_events", cfg.OsquerySocket)
		if err != nil {
			log.Fatalf("failed to create monitoring client: %v", err)
			return
		}

		ui.Start(cfg, monitorClient)
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
