// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/ui"
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "StartMonitoring the File Modification Tracker UI",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		ui.Start(cfg)
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
