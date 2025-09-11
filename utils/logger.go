package utils

import "go.uber.org/zap"

var Logger *zap.Logger

func InitLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("this is an info message")
	return logger
}
