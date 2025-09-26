package applogger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
