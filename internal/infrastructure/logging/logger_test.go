package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"qwin/internal/testutils"
	"strings"
	"testing"
	"time"
)

// Mock RepositoryError for testing
type mockRepositoryError struct {
	message   string
	code      string
	retryable bool
	context   map[string]string
	timestamp time.Time
}

func (m *mockRepositoryError) Error() string {
	return m.message
}

func (m *mockRepositoryError) GetCode() string {
	return m.code
}

func (m *mockRepositoryError) IsRetryable() bool {
	return m.retryable
}

func (m *mockRepositoryError) GetContext() map[string]string {
	return m.context
}

func (m *mockRepositoryError) GetTimestamp() time.Time {
	return m.timestamp
}

// Mock Logger for testing
type mockLogger struct {
	debugCalls []logCall
	infoCalls  []logCall
	warnCalls  []logCall
	errorCalls []logCall
}

type logCall struct {
	msg    string
	fields []interface{}
}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {
	m.debugCalls = append(m.debugCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Info(msg string, fields ...interface{}) {
	m.infoCalls = append(m.infoCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Warn(msg string, fields ...interface{}) {
	m.warnCalls = append(m.warnCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Error(msg string, fields ...interface{}) {
	m.errorCalls = append(m.errorCalls, logCall{msg: msg, fields: fields})
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger() returned nil")
	}

	if _, ok := logger.(*DefaultLogger); !ok {
		t.Errorf("NewDefaultLogger() returned %T, expected *DefaultLogger", logger)
	}
}

func TestDefaultLogger_LogLevels(t *testing.T) {
	// Capture current log state
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Restore original state after test
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	logger := &DefaultLogger{}

	tests := []struct {
		name           string
		logFunc        func(string, ...interface{})
		message        string
		fields         []interface{}
		levelToken     string
		expectedMsg    string
		expectedFields map[string]interface{}
	}{
		{
			name:           "Debug",
			logFunc:        logger.Debug,
			message:        "debug message",
			fields:         []interface{}{"key", "value"},
			levelToken:     "DEBUG",
			expectedMsg:    "debug message",
			expectedFields: map[string]interface{}{"key": "value"},
		},
		{
			name:           "Info",
			logFunc:        logger.Info,
			message:        "info message",
			fields:         []interface{}{"count", 42},
			levelToken:     "INFO",
			expectedMsg:    "info message",
			expectedFields: map[string]interface{}{"count": float64(42)}, // JSON numbers are float64
		},
		{
			name:           "Warn",
			logFunc:        logger.Warn,
			message:        "warn message",
			fields:         []interface{}{},
			levelToken:     "WARN",
			expectedMsg:    "warn message",
			expectedFields: map[string]interface{}{},
		},
		{
			name:           "Error",
			logFunc:        logger.Error,
			message:        "error message",
			fields:         []interface{}{"error", "test error"},
			levelToken:     "ERROR",
			expectedMsg:    "error message",
			expectedFields: map[string]interface{}{"error": "test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message, tt.fields...)

			output := strings.TrimSpace(buf.String())

			// Find the JSON part (skip timestamp prefix if any)
			jsonStart := strings.Index(output, "{")
			if jsonStart == -1 {
				t.Fatalf("Expected JSON output, got: %q", output)
			}
			jsonPart := output[jsonStart:]

			// Parse JSON
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(jsonPart), &logEntry); err != nil {
				t.Fatalf("Failed to parse JSON log entry: %v, output: %q", err, output)
			}

			// Check required fields exist
			if logEntry["timestamp"] == nil {
				t.Error("Expected log entry to have timestamp field")
			}

			if logEntry["level"] != tt.levelToken {
				t.Errorf("Expected level %q, got %q", tt.levelToken, logEntry["level"])
			}

			if logEntry["message"] != tt.expectedMsg {
				t.Errorf("Expected message %q, got %q", tt.expectedMsg, logEntry["message"])
			}

			// Check fields
			fields, ok := logEntry["fields"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected fields to be a map, got %T", logEntry["fields"])
			}

			for key, expectedValue := range tt.expectedFields {
				actualValue, exists := fields[key]
				if !exists {
					t.Errorf("Expected field %q to exist", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected field %q to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLogRepositoryError_WithRepositoryError(t *testing.T) {
	mockLog := &mockLogger{}

	repoErr := &mockRepositoryError{
		message:   "test repository error",
		code:      "TEST_ERROR",
		retryable: true,
		context:   map[string]string{"table": "users", "id": "123"},
		timestamp: time.Now(),
	}

	context := map[string]interface{}{
		"additional": "context",
		"count":      5,
	}

	LogRepositoryError(mockLog, repoErr, "test_operation", context)

	if len(mockLog.errorCalls) != 1 {
		t.Fatalf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	call := mockLog.errorCalls[0]
	if !strings.Contains(call.msg, "Repository error: test repository error") {
		t.Errorf("Expected error message to contain repository error, got %q", call.msg)
	}

	// Check that fields contain expected values
	fieldsMap := testutils.FieldsToMap(t, call.fields)

	expectedFields := map[string]interface{}{
		"operation":  "test_operation",
		"error_code": "TEST_ERROR",
		"retryable":  true,
		"table":      "users",
		"id":         "123",
		"additional": "context",
		"count":      5,
	}

	for key, expected := range expectedFields {
		if actual, exists := fieldsMap[key]; !exists {
			t.Errorf("Expected field %q not found in log call", key)
		} else if actual != expected {
			t.Errorf("Field %q: expected %v, got %v", key, expected, actual)
		}
	}
}

func TestLogRepositoryError_WithRegularError(t *testing.T) {
	mockLog := &mockLogger{}

	err := errors.New("regular error")
	context := map[string]interface{}{
		"context": "value",
	}

	LogRepositoryError(mockLog, err, "test_operation", context)

	if len(mockLog.errorCalls) != 1 {
		t.Fatalf("Expected 1 error call, got %d", len(mockLog.errorCalls))
	}

	call := mockLog.errorCalls[0]
	if !strings.Contains(call.msg, "Unexpected error: regular error") {
		t.Errorf("Expected error message to contain unexpected error, got %q", call.msg)
	}

	// Check that fields contain expected values
	fieldsMap := testutils.FieldsToMap(t, call.fields)

	if fieldsMap["operation"] != "test_operation" {
		t.Errorf("Expected operation field to be 'test_operation', got %v", fieldsMap["operation"])
	}

	if fieldsMap["context"] != "value" {
		t.Errorf("Expected context field to be 'value', got %v", fieldsMap["context"])
	}
}

func TestLogRepositoryError_WithNilLogger(t *testing.T) {
	// Capture current log state
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Capture log output to verify default logger is used
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Restore original state after test
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	err := errors.New("test error")
	LogRepositoryError(nil, err, "test_operation", nil)

	output := strings.TrimSpace(buf.String())

	// Find the JSON part (skip timestamp prefix if any)
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		t.Fatalf("Expected JSON output, got: %q", output)
	}
	jsonPart := output[jsonStart:]

	// Parse JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log entry: %v, output: %q", err, output)
	}

	// Check level
	if logEntry["level"] != "ERROR" {
		t.Errorf("Expected level ERROR, got %q", logEntry["level"])
	}

	// Check fields contain operation
	fields, ok := logEntry["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected fields to be a map, got %T", logEntry["fields"])
	}

	if fields["operation"] != "test_operation" {
		t.Errorf("Expected operation field to be 'test_operation', got %v", fields["operation"])
	}
}

func TestLogRepositoryOperation(t *testing.T) {
	mockLog := &mockLogger{}

	duration := 150 * time.Millisecond
	context := map[string]interface{}{
		"rows_affected": 5,
		"table":         "users",
	}

	LogRepositoryOperation(mockLog, "insert_user", duration, context)

	if len(mockLog.infoCalls) != 1 {
		t.Fatalf("Expected 1 info call, got %d", len(mockLog.infoCalls))
	}

	call := mockLog.infoCalls[0]
	if !strings.Contains(call.msg, "Repository operation completed: insert_user") {
		t.Errorf("Expected info message to contain operation completion, got %q", call.msg)
	}

	// Check that fields contain expected values
	fieldsMap := testutils.FieldsToMap(t, call.fields)

	expectedFields := map[string]interface{}{
		"operation":     "insert_user",
		"duration_ms":   int64(150),
		"rows_affected": 5,
		"table":         "users",
	}

	for key, expected := range expectedFields {
		if actual, exists := fieldsMap[key]; !exists {
			t.Errorf("Expected field %q not found in log call", key)
		} else if actual != expected {
			t.Errorf("Field %q: expected %v, got %v", key, expected, actual)
		}
	}
}

func TestLogRepositoryOperation_WithNilLogger(t *testing.T) {
	// Capture current log state
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Capture log output to verify default logger is used
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Restore original state after test
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	LogRepositoryOperation(nil, "test_operation", time.Millisecond, nil)

	output := strings.TrimSpace(buf.String())

	// Find the JSON part (skip timestamp prefix if any)
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		t.Fatalf("Expected JSON output, got: %q", output)
	}
	jsonPart := output[jsonStart:]

	// Parse JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log entry: %v, output: %q", err, output)
	}

	// Check level
	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level INFO, got %q", logEntry["level"])
	}

	// Check fields contain operation
	fields, ok := logEntry["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected fields to be a map, got %T", logEntry["fields"])
	}

	if fields["operation"] != "test_operation" {
		t.Errorf("Expected operation field to be 'test_operation', got %v", fields["operation"])
	}
}

func TestBackwardCompatibilityAliases(t *testing.T) {
	mockLog := &mockLogger{}

	// Test LogError alias
	err := errors.New("test error")
	LogError(mockLog, err, "test_op", nil)

	if len(mockLog.errorCalls) != 1 {
		t.Errorf("LogError alias should call LogRepositoryError")
	}

	// Test LogOperation alias
	LogOperation(mockLog, "test_op", time.Millisecond, nil)

	if len(mockLog.infoCalls) != 1 {
		t.Errorf("LogOperation alias should call LogRepositoryOperation")
	}
}
