package pkg

import "go.uber.org/zap"

// надстройка над логгером
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Sync() error
}
