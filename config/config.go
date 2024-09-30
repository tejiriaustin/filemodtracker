package config

import (
	"errors"
	"os"
	"sync"
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

var (
	appConfig     Config
	configOnce    sync.Once
	configRWMutex sync.RWMutex
)

func GetConfig() *Config {
	configRWMutex.RLock()
	defer configRWMutex.RUnlock()
	return &appConfig
}

func InitConfig(validator *validator.Validate, logger *logger.Logger) func() {
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
				logger.Warn("No config file found. Using defaults.")
			}
		}

		configRWMutex.Lock()
		defer configRWMutex.Unlock()

		if err := viper.Unmarshal(&appConfig); err != nil {
			logger.Error("Unable to decode config into struct", "error", err)
		}

		if err := validator.Struct(&appConfig); err != nil {
			logger.Error("Invalid config", "error", err)
			os.Exit(1)
		}

		appConfig.ConfigPath = viper.ConfigFileUsed()
		appConfig.PidFile = "/var/run/filemodtracker.pid"
	}
}

func UpdateConfig(newConfig Config) error {
	configRWMutex.Lock()
	defer configRWMutex.Unlock()

	validate := validator.New()
	if err := validate.Struct(newConfig); err != nil {
		return err
	}

	appConfig = newConfig
	return nil
}
