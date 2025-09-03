package errors

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// RetryLogger defines the interface for logging retry operations
type RetryLogger interface {
	Printf(format string, v ...interface{})
}

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Exponential backoff factor
	Jitter          bool          // Whether to add jitter to delays
	RetryableErrors []ErrorCode   // Specific error codes to retry
}

// Package-level logger variable that can be set by callers
var retryLogger RetryLogger

// DefaultRetryConfig returns a retry configuration with sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrCodeConnection,
			ErrCodeTimeout,
			ErrCodeTransaction,
		},
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func() error

// SetRetryLogger sets the package-level logger for retry operations
func SetRetryLogger(logger RetryLogger) {
	retryLogger = logger
}

// logRetryMessage logs a retry message using the configured logger
func logRetryMessage(format string, v ...interface{}) {
	if retryLogger != nil {
		retryLogger.Printf(format, v...)
	}
}

// withRetryImpl is the core retry implementation used by both public functions
func withRetryImpl(ctx context.Context, config *RetryConfig, operation RetryableOperation, operationName string) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the operation
		err := operation()
		if err == nil {
			// Log successful operation if it required retries and we have an operation name
			if attempt > 0 && operationName != "" {
				logRetryMessage("Repository operation '%s' succeeded after %d attempts", operationName, attempt+1)
			}
			return nil // Success
		}

		lastErr = err

		// Check if we should retry this error
		if !shouldRetry(err, config) {
			if operationName != "" {
				logRetryMessage("Repository operation '%s' failed with non-retryable error: %v", operationName, err)
			}
			return err // Non-retryable error
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay for next attempt
		delay := calculateDelay(attempt, config)

		// Log retry attempt
		if operationName != "" {
			logRetryMessage("Repository operation '%s' failed (attempt %d/%d), retrying in %v: %v",
				operationName, attempt+1, config.MaxAttempts, delay, err)
		} else {
			logRetryMessage("Repository operation failed (attempt %d/%d), retrying in %v: %v",
				attempt+1, config.MaxAttempts, delay, err)
		}

		// Wait before retrying, respecting context cancellation
		select {
		case <-ctx.Done():
			if operationName != "" {
				return fmt.Errorf("operation '%s' cancelled during retry: %w", operationName, ctx.Err())
			}
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	if operationName != "" {
		return fmt.Errorf("operation '%s' failed after %d attempts: %w", operationName, config.MaxAttempts, lastErr)
	}
	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// WithRetry executes an operation with retry logic
func WithRetry(ctx context.Context, config *RetryConfig, operation RetryableOperation) error {
	return withRetryImpl(ctx, config, operation, "")
}

// shouldRetry determines if an error should be retried based on configuration
func shouldRetry(err error, config *RetryConfig) bool {
	var repoErr *RepositoryError
	if !errors.As(err, &repoErr) {
		return false // Only retry repository errors
	}

	// Check if error is generally retryable
	if !repoErr.IsRetryable() {
		return false
	}

	// Check if error code is in the retryable list
	return slices.Contains(config.RetryableErrors, repoErr.Code)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(attempt int, config *RetryConfig) time.Duration {
	// Calculate exponential backoff
	multiplier := 1.0
	for range attempt {
		multiplier *= config.BackoffFactor
	}

	delay := time.Duration(float64(config.InitialDelay) * multiplier)

	// Add jitter if enabled (before applying max delay limit)
	if config.Jitter && delay > 0 {
		// Add up to 25% jitter
		jitterAmount := time.Duration(float64(delay) * 0.25)
		if jitterAmount > 0 {
			delay += time.Duration(time.Now().UnixNano() % int64(jitterAmount))
		}
	}

	// Apply maximum delay limit after jitter
	delay = min(delay, config.MaxDelay)

	return delay
}

// WithRetryContext executes an operation with retry logic and custom context
func WithRetryContext(ctx context.Context, config *RetryConfig, operation RetryableOperation, operationName string) error {
	return withRetryImpl(ctx, config, operation, operationName)
}

// RetryWithBackoff provides a simpler interface for common retry scenarios
func RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, operation RetryableOperation) error {
	config := &RetryConfig{
		MaxAttempts:   maxAttempts,
		InitialDelay:  initialDelay,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrCodeConnection,
			ErrCodeTimeout,
			ErrCodeTransaction,
			ErrCodeDiskSpace,
		},
	}
	return WithRetry(ctx, config, operation)
}

// RetryQuick provides a quick retry configuration for fast operations
func RetryQuick(ctx context.Context, operation RetryableOperation) error {
	config := &RetryConfig{
		MaxAttempts:   2,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorCode{
			ErrCodeConnection,
			ErrCodeTimeout,
		},
	}
	return WithRetry(ctx, config, operation)
}

// RetryPersistent provides a persistent retry configuration for critical operations
func RetryPersistent(ctx context.Context, operation RetryableOperation) error {
	config := &RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrCodeConnection,
			ErrCodeTimeout,
			ErrCodeTransaction,
			ErrCodeDiskSpace,
		},
	}
	return WithRetry(ctx, config, operation)
}
