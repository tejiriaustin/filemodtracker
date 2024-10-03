package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"github.com/tejiriaustin/savannah-assessment/logger"
)

type Config struct {
	ConfigPath         string
	Port               string        `mapstructure:"port"`
	MonitoredDirectory string        `mapstructure:"monitored_directory"`
	CheckFrequency     time.Duration `mapstructure:"check_frequency"`
	OsqueryConfig      string        `mapstructure:"osquery_config"`
	OsquerySocket      string        `mapstructure:"osquery_socket"`
	PidFilePath        string        `mapstructure:"pid_file_path"`
	mutex              sync.RWMutex
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
		viper.SetDefault("pid_file_path", filepath.Join(os.TempDir(), "filemodtracker.pid"))

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
	}
}

func (c *Config) WritePidFile(pid int) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.PidFilePath == "" {
		c.PidFilePath = filepath.Join(os.TempDir(), "filemodtracker.pid")
	}

	if err := ioutil.WriteFile(c.PidFilePath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	return nil
}

func (c *Config) ReadPidFile() (int, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.PidFilePath == "" {
		return 0, fmt.Errorf("PID file path not set")
	}

	content, err := ioutil.ReadFile(c.PidFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("daemon not running")
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}
	return pid, nil
}

func (c *Config) RemovePidFile() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.PidFilePath == "" {
		return nil // No PID file path set, nothing to remove
	}

	err := os.Remove(c.PidFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}
