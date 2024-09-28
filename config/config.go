package config

import (
	"errors"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"github.com/tejiriaustin/savannah-assessment/logger"
)

type Config struct {
	ConfigPath         string
	PidFile            string
	Port               string        `mapstructure:"port"`
	MonitoredDirectory string        `mapstructure:"monitored_directory"`
	CheckFrequency     time.Duration `mapstructure:"check_frequency"`
	OsqueryConfig      string        `mapstructure:"osquery_config"`
	OsquerySocket      string        `mapstructure:"osquery_socket"`
}

var appConfig Config

func GetConfig() *Config {
	return &appConfig
}

func InitConfig(validator *validator.Validate, log *logger.Logger) func() {
	return func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.filemodtracker")
		viper.AddConfigPath("/etc/filemodtracker")
		viper.AddConfigPath("/usr/local/etc/filemodtracker")

		viper.SetDefault("port", ":80")
		viper.SetDefault("monitored_directory", "/Users/%%")
		viper.SetDefault("check_frequency", "1m")
		viper.SetDefault("api_endpoint", "http://localhost:80")
		viper.SetDefault("osquery_config", "osquery_fim.conf")
		viper.SetDefault("osquery_socket", "/var/osquery/osquery.em")

		if err := viper.ReadInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				log.Warn("No config file found. Using defaults.")
			} else {
				log.Error("Error reading config file", "error", err)
			}
		} else {
			log.Info("Config file used", "path", viper.ConfigFileUsed())
		}

		if err := viper.Unmarshal(&appConfig); err != nil {
			log.Error("Unable to decode config into struct", "error", err)
		}

		if err := validator.Struct(&appConfig); err != nil {
			log.Error("Invalid config", "error", err)
			os.Exit(1)
		}

		appConfig.ConfigPath = viper.ConfigFileUsed()
		appConfig.PidFile = "/var/run/filemodtracker.pid"

		log.Info("Configuration initialized", "config", appConfig)
	}
}
