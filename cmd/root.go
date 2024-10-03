// Copyright © 2024 NAME HERE tejiriaustin123@gmail.com

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/logger"
)

var (
	cfgFile string
	log     *logger.Logger
	err     error
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "filemodtracker",
	Short: "File Modification Tracker",
	Long:  `A CLI tool to track and record modifications to files in a specified directory.`,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the File Modification Tracker daemon",
	Run:   stopDaemon,
}

func checkHealthEndpoint(url string) string {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "Running"
	} else {
		return fmt.Sprintf("Unexpected status: %s", resp.Status)
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the File Modification Tracker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		healthURL := fmt.Sprintf("http://localhost%s/health", cfg.Port)

		status := checkHealthEndpoint(healthURL)
		fmt.Printf("Application Status: %s\n", status)
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
		}
		fmt.Printf("Set %s to %s\n", key, value)
	},
}

func buildLogger() {
	logCfg := logger.Config{
		LogLevel:    "info",
		DevMode:     true,
		ServiceName: "filemodtracker",
	}
	log, err = logger.NewLogger(logCfg)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %v", err))
	}
}

func init() {
	validate := validator.New()

	cobra.OnInitialize(buildLogger, config.InitConfig(validate, log))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

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
