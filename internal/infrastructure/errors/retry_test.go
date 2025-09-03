package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts to be 3, got %d", config.MaxAttempts)
	}

	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected InitialDelay to be 100ms, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("Expected MaxDelay to be 5s, got %v", config.MaxDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}

	if !config.Jitter {
		t.Error("Expected Jitter to be true")
	}

	expectedCodes := []ErrorCode{ErrCodeConnection, ErrCodeTimeout, ErrCodeTransaction}
	if len(config.RetryableErrors) != len(expectedCodes) {
		t.Errorf("Expected %d retryable error codes, got %d", len(expectedCodes), len(config.RetryableErrors))
	}
}

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	callCount := 0
	operation := func() error {
		callCount++
		return nil // Success on first try
	}

	err := WithRetry(ctx, config, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}
}

func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond // Speed up test
	config.Jitter = false                      // Remove jitter for predictable timing

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
		}
		return nil // Success on third try
	}

	err := WithRetry(ctx, config, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected operation to be called 3 times, got %d", callCount)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	callCount := 0
	operation := func() error {
		callCount++
		return NewRepositoryError("test", errors.New("not found"), ErrCodeNotFound)
	}

	err := WithRetry(ctx, config, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}

	if !IsNotFound(err) {
		t.Error("Expected NotFound error")
	}
}

func TestWithRetry_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond // Speed up test
	config.Jitter = false                      // Remove jitter for predictable timing

	callCount := 0
	operation := func() error {
		callCount++
		return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
	}

	err := WithRetry(ctx, config, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != config.MaxAttempts {
		t.Errorf("Expected operation to be called %d times, got %d", config.MaxAttempts, callCount)
	}

	expectedMsg := "operation failed after 3 attempts"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	config.InitialDelay = 100 * time.Millisecond // Longer delay to allow cancellation

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context after first failure
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		}
		return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
	}

	err := WithRetry(ctx, config, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}
}

func TestWithRetry_NilConfig(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := WithRetry(ctx, nil, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}
}

func TestShouldRetry(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable connection error",
			err:      NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection),
			expected: true,
		},
		{
			name:     "retryable timeout error",
			err:      NewRepositoryError("test", errors.New("timeout"), ErrCodeTimeout),
			expected: true,
		},
		{
			name:     "retryable transaction error",
			err:      NewRepositoryError("test", errors.New("deadlock"), ErrCodeTransaction),
			expected: true,
		},
		{
			name:     "non-retryable not found error",
			err:      NewRepositoryError("test", errors.New("not found"), ErrCodeNotFound),
			expected: false,
		},
		{
			name:     "non-retryable validation error",
			err:      NewRepositoryError("test", errors.New("validation failed"), ErrCodeValidation),
			expected: false,
		},
		{
			name:     "non-repository error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "retryable disk space error not in config",
			err:      NewRepositoryError("test", errors.New("disk full"), ErrCodeDiskSpace),
			expected: false, // Not in default config's RetryableErrors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err, config)
			if result != tt.expected {
				t.Errorf("shouldRetry() = %v, expected %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	config := &RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false, // Disable jitter for predictable testing
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at MaxDelay
		{5, 1 * time.Second}, // Still capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calculateDelay(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, expected %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	config := &RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}

	// Test that jitter adds some variation within ±25%
	baseDelay := calculateDelay(0, &RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
	})

	jitteredDelay := calculateDelay(0, config)

	// Calculate bounds for ±25% variability
	lowerBound := baseDelay - time.Duration(float64(baseDelay)*0.25)
	upperBound := baseDelay + time.Duration(float64(baseDelay)*0.25)

	// Ensure lowerBound is not negative
	if lowerBound < 0 {
		lowerBound = 0
	}

	// Jittered delay should be within ±25% of base delay
	if jitteredDelay < lowerBound {
		t.Errorf("Jittered delay %v should be >= lower bound %v", jitteredDelay, lowerBound)
	}
	if jitteredDelay > upperBound {
		t.Errorf("Jittered delay %v should be <= upper bound %v", jitteredDelay, upperBound)
	}
}

func TestWithRetryContext(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond // Speed up test
	config.Jitter = false

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
		}
		return nil
	}

	err := WithRetryContext(ctx, config, operation, "test-operation")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected operation to be called 2 times, got %d", callCount)
	}
}

func TestWithRetryContext_Failure(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond // Speed up test
	config.Jitter = false

	callCount := 0
	operation := func() error {
		callCount++
		return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
	}

	err := WithRetryContext(ctx, config, operation, "test-operation")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg := "operation 'test-operation' failed after 3 attempts"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestWithRetryContext_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	config.InitialDelay = 100 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		}
		return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
	}

	err := WithRetryContext(ctx, config, operation, "test-operation")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg := "operation 'test-operation' cancelled during retry"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRetryWithBackoff(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
		}
		return nil
	}

	err := RetryWithBackoff(ctx, 3, 1*time.Millisecond, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected operation to be called 2 times, got %d", callCount)
	}
}

func TestRetryQuick(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return NewRepositoryError("test", errors.New("timeout"), ErrCodeTimeout)
		}
		return nil
	}

	err := RetryQuick(ctx, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected operation to be called 2 times, got %d", callCount)
	}
}

func TestRetryQuick_MaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return NewRepositoryError("test", errors.New("timeout"), ErrCodeTimeout)
	}

	err := RetryQuick(ctx, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 2 { // RetryQuick has MaxAttempts = 2
		t.Errorf("Expected operation to be called 2 times, got %d", callCount)
	}
}

func TestRetryPersistent(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
		}
		return nil
	}

	err := RetryPersistent(ctx, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected operation to be called 3 times, got %d", callCount)
	}
}

func TestRetryPersistent_NonRetryableError(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return NewRepositoryError("test", errors.New("validation failed"), ErrCodeValidation)
	}

	err := RetryPersistent(ctx, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}
}

// mockRetryLogger implements RetryLogger for testing
type mockRetryLogger struct {
	messages []string
}

func (m *mockRetryLogger) Printf(format string, v ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf(format, v...))
}

func TestSetRetryLogger(t *testing.T) {
	// Save original logger
	originalLogger := retryLogger
	defer func() {
		retryLogger = originalLogger
	}()

	// Test with mock logger
	mockLogger := &mockRetryLogger{}
	SetRetryLogger(mockLogger)

	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond
	config.Jitter = false

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return NewRepositoryError("test", errors.New("connection failed"), ErrCodeConnection)
		}
		return nil
	}

	err := WithRetryContext(ctx, config, operation, "test-operation")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify logging occurred
	if len(mockLogger.messages) != 2 {
		t.Errorf("Expected 2 log messages, got %d", len(mockLogger.messages))
	}

	// Check that messages contain expected content
	if !strings.Contains(mockLogger.messages[0], "test-operation") {
		t.Errorf("Expected first message to contain operation name, got: %s", mockLogger.messages[0])
	}
	if !strings.Contains(mockLogger.messages[1], "succeeded after 2 attempts") {
		t.Errorf("Expected second message to contain success message, got: %s", mockLogger.messages[1])
	}
}

func TestLogRetryMessage_NilLogger(t *testing.T) {
	// Save original logger
	originalLogger := retryLogger
	defer func() {
		retryLogger = originalLogger
	}()

	// Set logger to nil
	SetRetryLogger(nil)

	// This should not panic
	logRetryMessage("test message %s", "param")
}

func TestCustomRetryLogger(t *testing.T) {
	// Save original logger
	originalLogger := retryLogger
	defer func() {
		retryLogger = originalLogger
	}()

	// Create a custom logger
	customLogger := &mockRetryLogger{}
	SetRetryLogger(customLogger)

	// Test logging through the custom logger
	logRetryMessage("test message %s", "param")

	// Verify the message was logged
	if len(customLogger.messages) != 1 {
		t.Errorf("Expected 1 log message, got %d", len(customLogger.messages))
	}

	expectedMessage := "test message param"
	if customLogger.messages[0] != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, customLogger.messages[0])
	}
}
