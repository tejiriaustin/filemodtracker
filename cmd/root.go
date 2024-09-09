/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/tejiriaustin/savannah-assessment/config"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	s       service.Service
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "savannah-assessment",
	Short: "File Modification Tracker",
	Long:  `A CLI tool to track and record modifications to files in a specified directory.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the File Modification Tracker service",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		fmt.Printf("Starting File Modification Tracker...\n")
		fmt.Printf("Monitoring directory: %s\n", cfg.MonitorDir)
		fmt.Printf("Check frequency: %s\n", cfg.CheckFrequency)
		fmt.Printf("API endpoint: %s\n", cfg.APIEndpoint)
		fmt.Printf("Osquery socket: %s\n", cfg.OsquerySocket)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the File Modification Tracker service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping File Modification Tracker service...")
		// Implement service stop logic here
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the File Modification Tracker service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking File Modification Tracker service status...")
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
	cobra.OnInitialize(config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)

	var err error
	s, err = NewService()
	if err != nil {
		fmt.Printf("Failed to create service: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
