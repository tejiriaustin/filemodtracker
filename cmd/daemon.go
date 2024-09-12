package cmd

import (
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the File Modification Tracker daemon and server",
	Run:   startDaemonService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}

func startDaemonService(cmd *cobra.Command, args []string) {
	start()
}
