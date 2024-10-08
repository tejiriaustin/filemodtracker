// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/logger"
	"github.com/tejiriaustin/savannah-assessment/ui"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "StartMonitoring the File Modification Tracker UI",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()

		logCfg := logger.Config{
			LogLevel:    "info",
			DevMode:     true,
			ServiceName: "ui",
		}
		uiLogger, err := logger.NewLogger(logCfg)
		if err != nil {
			log.Fatal("Failed to create UI logger", "error", err)
		}
		ui.Start(cfg, uiLogger)
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
