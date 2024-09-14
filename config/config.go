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
	ConfigPath     string
	PidFile        string
	Port           string        `mapstructure:"port"`
	MonitorDir     string        `mapstructure:"monitor_dir"`
	CheckFrequency time.Duration `mapstructure:"check_frequency"`
	Timeout        time.Duration `mapstructure:"timeout"`
	OsqueryConfig  string        `mapstructure:"osquery_config"`
	ApiEndpoint    string        `mapstructure:"api_endpoint"`
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

		viper.SetDefault("port", ":80")
		viper.SetDefault("monitor_dir", "/Users/%%")
		viper.SetDefault("check_frequency", "1m")
		viper.SetDefault("timeout", "1m")
		viper.SetDefault("api_endpoint", "http://localhost:80")
		viper.SetDefault("osquery_config", "osquery_fim.conf")

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
		appConfig.PidFile = "/var/run/filemodtracker.pid"

		return
	}
}
