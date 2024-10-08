package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

type Config struct {
	LogLevel    string
	DevMode     bool
	ServiceName string
}

func NewLogger(cfg Config) (*Logger, error) {
	var zapCfg zap.Config
	if cfg.DevMode {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	level, err := zapcore.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := zapCfg.Build(zap.Fields(zap.String("service", cfg.ServiceName)))
	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: logger.Sugar()}, nil
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.SugaredLogger.Infow(fmt.Sprintf(msg, fields...))
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.SugaredLogger.Errorf(fmt.Sprintf(msg, fields...))
}

func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.SugaredLogger.Fatalw(fmt.Sprintf(msg, fields...))
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.SugaredLogger.Warnw(fmt.Sprintf(msg, fields...))
}

func (l *Logger) Debug(fields ...interface{}) {
	l.SugaredLogger.Debug(fields...)
}
