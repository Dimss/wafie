package applogger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger

func NewLogger() *zap.Logger {
	if logger != nil {
		return logger
	}
	config := zap.NewDevelopmentConfig()
	//logger, err := zap.NewProduction()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewLoggerToFile() *zap.Logger {
	rotatingWriter := &lumberjack.Logger{
		Filename:   "relay.log",
		MaxSize:    5,
		MaxBackups: 1,
		MaxAge:     3,
	}
	config := zap.NewDevelopmentEncoderConfig()
	encoder := zapcore.NewConsoleEncoder(config)
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(rotatingWriter),
		zapcore.DebugLevel,
	)
	return zap.New(core)
}
