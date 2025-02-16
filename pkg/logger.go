package pkg

import "go.uber.org/zap"

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Sync() error
}

type zapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(z *zap.Logger) Logger {
	return &zapLogger{logger: z}
}

func (z *zapLogger) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}
func (z *zapLogger) Warn(msg string, fields ...zap.Field) {
	z.logger.Warn(msg, fields...)
}
func (z *zapLogger) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}
func (z *zapLogger) Sync() error {
	return z.logger.Sync()
}
