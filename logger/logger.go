package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
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

	return &Logger{Logger: logger}, nil
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.Logger.Sugar().Infow(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.Logger.Sugar().Errorw(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.Logger.Sugar().Fatalw(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.Logger.Sugar().Warnw(msg, fields...)
}
