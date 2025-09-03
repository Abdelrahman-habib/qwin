package errors

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// ClassifyError classifies database errors into repository error codes
func ClassifyError(err error) ErrorCode {
	if err == nil {
		return ErrCodeUnknown
	}

	// First, try driver-specific type assertions for more accurate classification
	if code := classifySQLiteError(err); code != ErrCodeUnknown {
		return code
	}

	// Handle standard library errors
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrCodeNotFound
	case errors.Is(err, context.DeadlineExceeded):
		return ErrCodeTimeout
	case errors.Is(err, context.Canceled):
		return ErrCodeTimeout
	}

	// Fall back to string-based classification for non-driver-specific errors
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "unique constraint"):
		return ErrCodeDuplicate
	case strings.Contains(errStr, "foreign key constraint"):
		return ErrCodeConstraint
	case strings.Contains(errStr, "check constraint"):
		return ErrCodeConstraint
	case strings.Contains(errStr, "not null constraint"):
		return ErrCodeConstraint
	case strings.Contains(errStr, "database is locked"):
		return ErrCodeConnection
	case strings.Contains(errStr, "database disk image is malformed"):
		return ErrCodeCorruption
	case strings.Contains(errStr, "no such table"):
		return ErrCodeConnection
	case strings.Contains(errStr, "no such column"):
		return ErrCodeConnection
	case strings.Contains(errStr, "permission denied"):
		return ErrCodePermission
	case strings.Contains(errStr, "access denied"):
		return ErrCodePermission
	case strings.Contains(errStr, "disk full"):
		return ErrCodeDiskSpace
	case strings.Contains(errStr, "no space left"):
		return ErrCodeDiskSpace
	case strings.Contains(errStr, "connection refused"):
		return ErrCodeConnection
	case strings.Contains(errStr, "network unreachable"):
		return ErrCodeConnection
	case strings.Contains(errStr, "timeout"):
		return ErrCodeTimeout
	case strings.Contains(errStr, "deadlock"):
		return ErrCodeTransaction
	case strings.Contains(errStr, "serialization failure"):
		return ErrCodeTransaction
	default:
		return ErrCodeUnknown
	}
}

// WrapDatabaseError wraps a database error with repository error context
func WrapDatabaseError(op string, err error) error {
	if err == nil {
		return nil
	}

	code := ClassifyError(err)
	return NewRepositoryError(op, err, code)
}

// WrapDatabaseErrorWithContext wraps a database error with repository error context and additional context
func WrapDatabaseErrorWithContext(op string, err error, contextMap map[string]string) error {
	if err == nil {
		return nil
	}

	code := ClassifyError(err)
	return NewRepositoryErrorWithContext(op, err, code, contextMap)
}

// HandleNotFound creates a standardized not found error for repository operations
func HandleNotFound(op string, resource string, identifier string) error {
	contextMap := map[string]string{
		"resource":   resource,
		"identifier": identifier,
	}
	return NewRepositoryErrorWithContext(op, sql.ErrNoRows, ErrCodeNotFound, contextMap)
}

// HandleValidationError creates a standardized validation error for repository operations
func HandleValidationError(op string, field string, value string, reason string) error {
	contextMap := map[string]string{
		"field":  field,
		"value":  value,
		"reason": reason,
	}
	return NewRepositoryErrorWithContext(op, errors.New("validation failed"), ErrCodeValidation, contextMap)
}

// HandleConstraintError creates a standardized constraint violation error
func HandleConstraintError(op string, constraint string, details string) error {
	contextMap := map[string]string{
		"constraint": constraint,
		"details":    details,
	}
	return NewRepositoryErrorWithContext(op, errors.New("constraint violation"), ErrCodeConstraint, contextMap)
}

// HandleConnectionError creates a standardized connection error
func HandleConnectionError(op string, details string) error {
	contextMap := map[string]string{
		"details": details,
	}
	return NewRepositoryErrorWithContext(op, errors.New("connection error"), ErrCodeConnection, contextMap)
}

// HandleTransactionError creates a standardized transaction error
func HandleTransactionError(op string, phase string, details string) error {
	contextMap := map[string]string{
		"phase":   phase,
		"details": details,
	}
	return NewRepositoryErrorWithContext(op, errors.New("transaction error"), ErrCodeTransaction, contextMap)
}

// HandleTimeoutError creates a standardized timeout error
func HandleTimeoutError(op string, timeout string) error {
	ctx := map[string]string{
		"timeout": timeout,
	}
	return NewRepositoryErrorWithContext(op, context.DeadlineExceeded, ErrCodeTimeout, ctx)
}

// HandleDuplicateError creates a standardized duplicate error
func HandleDuplicateError(op string, resource string, field string, value string) error {
	contextMap := map[string]string{
		"resource": resource,
		"field":    field,
		"value":    value,
	}
	return NewRepositoryErrorWithContext(op, errors.New("duplicate entry"), ErrCodeDuplicate, contextMap)
}

// HandlePermissionError creates a standardized permission error
func HandlePermissionError(op string, resource string, action string) error {
	contextMap := map[string]string{
		"resource": resource,
		"action":   action,
	}
	return NewRepositoryErrorWithContext(op, errors.New("permission denied"), ErrCodePermission, contextMap)
}

// HandleDiskSpaceError creates a standardized disk space error
func HandleDiskSpaceError(op string, path string, required string) error {
	contextMap := map[string]string{
		"path":     path,
		"required": required,
	}
	return NewRepositoryErrorWithContext(op, errors.New("insufficient disk space"), ErrCodeDiskSpace, contextMap)
}

// HandleCorruptionError creates a standardized corruption error
func HandleCorruptionError(op string, resource string, details string) error {
	contextMap := map[string]string{
		"resource": resource,
		"details":  details,
	}
	return NewRepositoryErrorWithContext(op, errors.New("data corruption detected"), ErrCodeCorruption, contextMap)
}
