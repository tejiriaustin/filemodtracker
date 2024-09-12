package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	ConfigPath        string
	PidFile           string
	APIAddress        string        `json:"api_address"`
	MonitorDir        string        `mapstructure:"monitor_dir"`
	CheckFrequency    time.Duration `mapstructure:"check_frequency"`
	OsquerySocket     string        `mapstructure:"osquery_socket"`
	OsqueryConfigPath string        `mapstructure:"osquery_config"`
	DataDir           string        `mapstructure:"data_dir"`
}

var appConfig Config

func GetConfig() *Config {
	return &appConfig
}

func InitConfig(validator *validator.Validate) func() {
	return func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.filemodtracker")
		viper.AddConfigPath("/etc/filemodtracker")

		viper.SetDefault("api_address", "http://localhost:8080")
		viper.SetDefault("monitor_dir", ".")
		viper.SetDefault("check_frequency", "1m")
		viper.SetDefault("api_endpoint", "http://localhost:8080/api/report")
		viper.SetDefault("osquery_socket", "/usr/local/var/osquery/osquery.em")
		viper.SetDefault("osquery_config", "/usr/local/var/osquery/osquery.em")

		if err := viper.ReadInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				fmt.Println("No config file found. Using defaults.")
			}
		}

		if err := viper.Unmarshal(&appConfig); err != nil {
			fmt.Println("unable to decode config into struct: %w", err)
		}

		if err := validator.Struct(&appConfig); err != nil {
			fmt.Printf("Invalid config: %v\n", err)
			os.Exit(1)
		}

		appConfig.ConfigPath = viper.ConfigFileUsed()

		return
	}
}
