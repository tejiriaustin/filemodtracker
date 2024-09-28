package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

type Config struct {
	LogLevel string
	DevMode  bool
}

func NewLogger(config Config) (*Logger, error) {
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

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	sugar := zapLogger.Sugar()
	return &Logger{sugar}, nil
}

func (l *Logger) Sync() error {
	return l.SugaredLogger.Sync()
}
