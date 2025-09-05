package errors

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ErrorCode represents different types of repository errors
type ErrorCode int

const (
	ErrCodeUnknown ErrorCode = iota
	ErrCodeNotFound
	ErrCodeDuplicate
	ErrCodeConstraint
	ErrCodeConnection
	ErrCodeTransaction
	ErrCodeTimeout
	ErrCodeRetryable
	ErrCodeNonRetryable
	ErrCodeValidation
	ErrCodePermission
	ErrCodeDiskSpace
	ErrCodeCorruption
	ErrCodeInternal
	ErrCodeBusy
	ErrCodeSchema
)

// String returns a string representation of the error code
func (e ErrorCode) String() string {
	switch e {
	case ErrCodeNotFound:
		return "NOT_FOUND"
	case ErrCodeDuplicate:
		return "DUPLICATE"
	case ErrCodeConstraint:
		return "CONSTRAINT"
	case ErrCodeConnection:
		return "CONNECTION"
	case ErrCodeTransaction:
		return "TRANSACTION"
	case ErrCodeTimeout:
		return "TIMEOUT"
	case ErrCodeRetryable:
		return "RETRYABLE"
	case ErrCodeNonRetryable:
		return "NON_RETRYABLE"
	case ErrCodeValidation:
		return "VALIDATION"
	case ErrCodePermission:
		return "PERMISSION"
	case ErrCodeDiskSpace:
		return "DISK_SPACE"
	case ErrCodeCorruption:
		return "CORRUPTION"
	case ErrCodeInternal:
		return "INTERNAL"
	case ErrCodeBusy:
		return "BUSY"
	case ErrCodeSchema:
		return "SCHEMA"
	default:
		return "UNKNOWN"
	}
}

// RepositoryError represents a repository-specific error with context and retry information
type RepositoryError struct {
	Op        string            // operation name
	Err       error             // underlying error
	Code      ErrorCode         // error classification
	Retryable bool              // whether the error is retryable
	Context   map[string]string // additional context information
	Timestamp time.Time         // when the error occurred
}

func (e *RepositoryError) Error() string {
	// Guard against nil receiver
	if e == nil {
		return "repository error"
	}

	var parts []string

	if e.Op != "" {
		parts = append(parts, fmt.Sprintf("op=%s", e.Op))
	}

	if e.Code != ErrCodeUnknown {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code.String()))
	}

	if e.Retryable {
		parts = append(parts, "retryable=true")
	}

	// Add context with deterministic order (treat nil Context as empty)
	if e.Context != nil && len(e.Context) > 0 {
		// Collect keys and sort them for deterministic output
		keys := make([]string, 0, len(e.Context))
		for k := range e.Context {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Append key=value pairs in sorted order
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, e.Context[k]))
		}
	}

	contextStr := ""
	if len(parts) > 0 {
		contextStr = fmt.Sprintf(" [%s]", strings.Join(parts, " "))
	}

	if e.Err != nil {
		return e.Err.Error() + contextStr
	}
	return "repository error" + contextStr
}

func (e *RepositoryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is implements error matching for errors.Is
func (e *RepositoryError) Is(target error) bool {
	if e == nil {
		return false
	}
	if t, ok := target.(*RepositoryError); ok {
		return e.Code == t.Code
	}
	// Also check if the target matches the underlying/wrapped error
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

// IsRetryable returns whether the error is retryable
func (e *RepositoryError) IsRetryable() bool {
	if e == nil {
		return false
	}
	return e.Retryable
}

// GetCode returns the error code as a string (for logging interface compatibility)
func (e *RepositoryError) GetCode() string {
	if e == nil {
		return ErrCodeUnknown.String()
	}
	return e.Code.String()
}

// GetContext returns the error context (for logging interface compatibility)
func (e *RepositoryError) GetContext() map[string]string {
	if e == nil {
		return make(map[string]string)
	}
	if e.Context == nil {
		return make(map[string]string)
	}
	return e.Context
}

// GetTimestamp returns the error timestamp (for logging interface compatibility)
func (e *RepositoryError) GetTimestamp() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.Timestamp
}

// WithContext adds context information to the error by mutating the receiver.
// This method is not concurrency-safe and should not be used after the error
// has been published to other goroutines without proper synchronization.
// For concurrent usage, create a new error with NewRepositoryErrorWithContext instead.
func (e *RepositoryError) WithContext(key, value string) *RepositoryError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// NewRepositoryError creates a new repository error with the given parameters
func NewRepositoryError(op string, err error, code ErrorCode) *RepositoryError {
	return &RepositoryError{
		Op:        op,
		Err:       err,
		Code:      code,
		Retryable: isRetryableError(code, err),
		Context:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

// NewRepositoryErrorWithContext creates a new repository error with additional context
func NewRepositoryErrorWithContext(op string, err error, code ErrorCode, context map[string]string) *RepositoryError {
	repoErr := NewRepositoryError(op, err, code)
	if context != nil {
		// Clone the context map to avoid external mutation and data races
		repoErr.Context = make(map[string]string, len(context))
		for k, v := range context {
			repoErr.Context[k] = v
		}
	}
	return repoErr
}

// isDiskSpaceRetryable determines if disk space errors should be retryable
// This can be configured based on application needs - by default returns false
// as disk space errors require external intervention (cleanup, more storage)
func isDiskSpaceRetryable() bool {
	// TODO: This could be made configurable via environment variable or config file
	// For now, disk space errors are non-retryable by default
	return false
}

// isRetryableError determines if an error is retryable based on its type
func isRetryableError(code ErrorCode, err error) bool {
	switch code {
	case ErrCodeConnection, ErrCodeTimeout, ErrCodeTransaction, ErrCodeBusy:
		return true
	case ErrCodeRetryable:
		return true
	case ErrCodeNonRetryable:
		return false
	case ErrCodeNotFound, ErrCodeDuplicate, ErrCodeConstraint, ErrCodeValidation, ErrCodePermission, ErrCodeCorruption, ErrCodeInternal, ErrCodeSchema:
		return false
	case ErrCodeDiskSpace:
		// Disk space errors are non-retryable by default as they require external intervention
		// (cleanup, adding storage). Can be made retryable via configuration if needed.
		return isDiskSpaceRetryable()
	default:
		// For unknown errors, check the underlying error message
		if err != nil {
			errStr := strings.ToLower(err.Error())
			return strings.Contains(errStr, "temporary") ||
				strings.Contains(errStr, "retry") ||
				strings.Contains(errStr, "busy") ||
				strings.Contains(errStr, "locked") ||
				strings.Contains(errStr, "deadlock")
		}
		return false
	}
}

// Error classification functions

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeNotFound
	}
	return false
}

// IsDuplicate checks if the error is a "duplicate" error
func IsDuplicate(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeDuplicate
	}
	return false
}

// IsConstraint checks if the error is a "constraint violation" error
func IsConstraint(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeConstraint
	}
	return false
}

// IsConnection checks if the error is a "connection" error
func IsConnection(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeConnection
	}
	return false
}

// IsTransaction checks if the error is a "transaction" error
func IsTransaction(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeTransaction
	}
	return false
}

// IsTimeout checks if the error is a "timeout" error
func IsTimeout(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeTimeout
	}
	return false
}

// IsRetryable checks if the error is retryable
func IsRetryable(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Retryable
	}
	return false
}

// IsValidation checks if the error is a validation error
func IsValidation(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeValidation
	}
	return false
}

// IsPermission checks if the error is a permission error
func IsPermission(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodePermission
	}
	return false
}

// IsDiskSpace checks if the error is a disk space error
func IsDiskSpace(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeDiskSpace
	}
	return false
}

// IsCorruption checks if the error is a corruption error
func IsCorruption(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeCorruption
	}
	return false
}

// IsInternal checks if the error is an internal/API misuse error
func IsInternal(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeInternal
	}
	return false
}

// IsBusy checks if the error is a busy/locked error
func IsBusy(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeBusy
	}
	return false
}

// IsSchema checks if the error is a schema error
func IsSchema(err error) bool {
	var repoErr *RepositoryError
	if errors.As(err, &repoErr) {
		return repoErr.Code == ErrCodeSchema
	}
	return false
}
