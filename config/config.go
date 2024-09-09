// config/config.go
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	MonitorDir     string        `mapstructure:"monitor_dir"`
	CheckFrequency time.Duration `mapstructure:"check_frequency"`
	APIEndpoint    string        `mapstructure:"api_endpoint"`
	OsquerySocket  string        `mapstructure:"osquery_socket"`
}

var appConfig Config

func GetConfig() Config {
	return appConfig
}

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.filemodtracker")
	viper.AddConfigPath("/etc/filemodtracker")

	viper.SetDefault("monitor_dir", ".")
	viper.SetDefault("check_frequency", "1m")
	viper.SetDefault("api_endpoint", "http://localhost:8080/api/report")
	viper.SetDefault("osquery_socket", "/var/osquery/osquery.em")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fmt.Println("No config file found. Using defaults.")
		}
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		fmt.Println("unable to decode config into struct: %w", err)
	}

	return
}
