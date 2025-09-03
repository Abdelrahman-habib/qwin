package errors

import (
	"qwin/internal/infrastructure/logging"
)

// LoggerBridge adapts the logging.Logger interface to RetryLogger
type LoggerBridge struct {
	logger logging.Logger
}

// NewLoggerBridge creates a new bridge from logging.Logger to RetryLogger
func NewLoggerBridge(logger logging.Logger) RetryLogger {
	return &LoggerBridge{logger: logger}
}

// Printf implements RetryLogger interface by delegating to the logging.Logger
func (b *LoggerBridge) Printf(format string, v ...interface{}) {
	if b.logger != nil {
		b.logger.Info(format, v...)
	}
}

// SetDefaultRetryLogger sets up the default retry logger using the logging package
func SetDefaultRetryLogger() {
	defaultLogger := logging.NewDefaultLogger()
	bridge := NewLoggerBridge(defaultLogger)
	SetRetryLogger(bridge)
}
