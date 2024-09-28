package logger

import (
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

type Config struct {
	DevMode     bool
	LogLevel    string
	ServiceName string
}

func NewLogger(config Config) (*Logger, error) {
	if config.ServiceName == "" {
		return nil, errors.New("application name cannot be empty")
	}

	level, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, err
	}

	zapConfig := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(level),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
	}

	if config.DevMode {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	zapLogger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
		zap.Fields(
			zap.String("service_name", config.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	sugar := zapLogger.Sugar()
	return &Logger{sugar}, nil
}

func (l *Logger) Sync() error {
	return l.SugaredLogger.Sync()
}
