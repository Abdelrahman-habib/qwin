package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Logger interface for repository operations
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// DefaultLogger provides a simple logger implementation
type DefaultLogger struct{}

// NewDefaultLogger creates a new default logger instance
func NewDefaultLogger() Logger {
	return &DefaultLogger{}
}

// logEntry represents a structured log entry
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
}

// fieldsToMap converts the variadic fields slice to a map
// Expected format: key1, value1, key2, value2, ...
func fieldsToMap(fields []interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				result[key] = fields[i+1]
			} else {
				// If key is not a string, use index as key
				result[fmt.Sprintf("field_%d", i/2)] = fields[i]
				if i+1 < len(fields) {
					result[fmt.Sprintf("field_%d_value", i/2)] = fields[i+1]
				}
			}
		} else {
			// Odd number of fields, add the last one with an index key
			result[fmt.Sprintf("field_%d", i/2)] = fields[i]
		}
	}

	return result
}

// logStructured logs a message with structured JSON format
func (l *DefaultLogger) logStructured(level, msg string, fields []interface{}) {
	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Fields:    fieldsToMap(fields),
	}

	// Try to marshal to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to safe string representation
		fallbackFields := fmt.Sprintf("%v", fields)
		fallbackEntry := logEntry{
			Timestamp: entry.Timestamp,
			Level:     level,
			Message:   msg,
			Fields: map[string]interface{}{
				"original_fields": fallbackFields,
				"marshal_error":   err.Error(),
			},
		}

		if jsonBytes, err = json.Marshal(fallbackEntry); err != nil {
			// Last resort - simple text log
			log.Printf("[%s] %s %s", level, msg, fallbackFields)
			return
		}
	}

	log.Println(string(jsonBytes))
}

func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	l.logStructured("DEBUG", msg, fields)
}

func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	l.logStructured("INFO", msg, fields)
}

func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	l.logStructured("WARN", msg, fields)
}

func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	l.logStructured("ERROR", msg, fields)
}

// RepositoryError interface for error classification (to avoid circular imports)
type RepositoryError interface {
	Error() string
	GetCode() string
	IsRetryable() bool
	GetContext() map[string]string
	GetTimestamp() time.Time
}

// LogRepositoryError logs repository errors with appropriate context
func LogRepositoryError(logger Logger, err error, operation string, context map[string]interface{}) {
	if logger == nil {
		logger = NewDefaultLogger()
	}

	// Try to cast to our RepositoryError interface
	if repoErr, ok := err.(RepositoryError); ok {
		fields := []interface{}{
			"operation", operation,
			"error_code", repoErr.GetCode(),
			"retryable", repoErr.IsRetryable(),
			"timestamp", repoErr.GetTimestamp(),
		}

		// Add repository error context
		for k, v := range repoErr.GetContext() {
			fields = append(fields, k, v)
		}

		// Add additional context
		for k, v := range context {
			fields = append(fields, k, v)
		}

		logger.Error(fmt.Sprintf("Repository error: %s", err.Error()), fields...)
	} else {
		fields := []interface{}{
			"operation", operation,
			"error_type", fmt.Sprintf("%T", err),
		}

		// Add additional context
		for k, v := range context {
			fields = append(fields, k, v)
		}

		logger.Error(fmt.Sprintf("Unexpected error: %s", err.Error()), fields...)
	}
}

// LogRepositoryOperation logs successful repository operations for monitoring
func LogRepositoryOperation(logger Logger, operation string, duration time.Duration, context map[string]interface{}) {
	if logger == nil {
		logger = NewDefaultLogger()
	}

	fields := []interface{}{
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
	}

	// Add additional context
	for k, v := range context {
		fields = append(fields, k, v)
	}

	logger.Info(fmt.Sprintf("Repository operation completed: %s", operation), fields...)
}

// LogError is an alias for LogRepositoryError for backward compatibility
func LogError(logger Logger, err error, operation string, context map[string]interface{}) {
	LogRepositoryError(logger, err, operation, context)
}

// LogOperation is an alias for LogRepositoryOperation for backward compatibility
func LogOperation(logger Logger, operation string, duration time.Duration, context map[string]interface{}) {
	LogRepositoryOperation(logger, operation, duration, context)
}
