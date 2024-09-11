// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tejiriaustin/savannah-assessment/daemon"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/server"
	"log"
	"os"

	"github.com/tejiriaustin/savannah-assessment/config"
)

var (
	cfgFile string
	d       *daemon.Daemon
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "savannah-assessment",
	Short: "File Modification Tracker",
	Long:  `A CLI tool to track and record modifications to files in a specified directory.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the File Modification Tracker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()

		dbClient, err := db.NewClient(cfg.DataDir)
		if err != nil {
			log.Fatalf("Failed to create database client: %v", err)
		}
		defer dbClient.Close()

		cmdChan := make(chan string, 100) // Buffer for 100 commands

		d, err = daemon.New(cfg, dbClient, cmdChan)
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

		// The server.Start() method will block until the server is shut down
		// After the server is shut down, we should also stop the daemon
		if err := d.Stop(); err != nil {
			log.Printf("Failed to stop daemon: %v", err)
		}
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the File Modification Tracker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		dbClient, err := db.NewClient(cfg.DataDir)
		if err != nil {
			fmt.Printf("Error creating database client: %v\n", err)
			os.Exit(1)
		}

		d, err = daemon.New(cfg, dbClient, nil)
		if err != nil {
			fmt.Printf("Error creating daemon: %v\n", err)
			os.Exit(1)
		}

		err = d.Stop()
		if err != nil {
			fmt.Printf("Error stopping daemon: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Daemon stopped successfully.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the File Modification Tracker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking File Modification Tracker daemon status...")
		// Implement status check logic here
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration for File Modification Tracker",
	Long:  `View or modify the configuration for File Modification Tracker.`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current configuration:")
		fmt.Printf("Monitor directory: %s\n", viper.GetString("monitor_dir"))
		fmt.Printf("Check frequency: %s\n", viper.GetDuration("check_frequency"))
		fmt.Printf("API endpoint: %s\n", viper.GetString("api_endpoint"))
		fmt.Printf("Osquery socket: %s\n", viper.GetString("osquery_socket"))
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		viper.Set(key, value)
		err := viper.WriteConfig()
		if err != nil {
			fmt.Printf("Error writing config: %v\n", err)
		} else {
			fmt.Printf("Set %s to %s\n", key, value)
		}
	},
}

func init() {
	validator := validator.New()

	cobra.OnInitialize(config.InitConfig(validator))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
