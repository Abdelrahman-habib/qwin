package errors

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{"nil error", nil, ErrCodeUnknown},
		{"sql.ErrNoRows", sql.ErrNoRows, ErrCodeNotFound},
		{"context.DeadlineExceeded", context.DeadlineExceeded, ErrCodeTimeout},
		{"context.Canceled", context.Canceled, ErrCodeTimeout},
		{"unique constraint", errors.New("UNIQUE constraint failed"), ErrCodeDuplicate},
		{"foreign key constraint", errors.New("FOREIGN KEY constraint failed"), ErrCodeConstraint},
		{"check constraint", errors.New("CHECK constraint failed"), ErrCodeConstraint},
		{"not null constraint", errors.New("NOT NULL constraint failed"), ErrCodeConstraint},
		{"database locked", errors.New("database is locked"), ErrCodeBusy},
		{"database corruption", errors.New("database disk image is malformed"), ErrCodeCorruption},
		{"no such table", errors.New("no such table: users"), ErrCodeSchema},
		{"no such column", errors.New("no such column: name"), ErrCodeSchema},
		{"permission denied", errors.New("permission denied"), ErrCodePermission},
		{"access denied", errors.New("access denied"), ErrCodePermission},
		{"disk full", errors.New("disk full"), ErrCodeDiskSpace},
		{"no space left", errors.New("no space left on device"), ErrCodeDiskSpace},
		{"connection refused", errors.New("connection refused"), ErrCodeConnection},
		{"network unreachable", errors.New("network unreachable"), ErrCodeConnection},
		{"timeout", errors.New("operation timeout"), ErrCodeTimeout},
		{"deadlock", errors.New("deadlock detected"), ErrCodeTransaction},
		{"serialization failure", errors.New("serialization failure"), ErrCodeTransaction},
		{"unknown error", errors.New("some unknown error"), ErrCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyError(tt.err); got != tt.expected {
				t.Errorf("ClassifyError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestClassifyError_StringFallback(t *testing.T) {
	// Test that non-SQLite errors still use string-based classification
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "Generic error with unique constraint message",
			err:      errors.New("UNIQUE constraint failed: users.email"),
			expected: ErrCodeDuplicate,
		},
		{
			name:     "Generic error with foreign key message",
			err:      errors.New("FOREIGN KEY constraint failed"),
			expected: ErrCodeConstraint,
		},
		{
			name:     "Generic error with database locked message",
			err:      errors.New("database is locked"),
			expected: ErrCodeBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyError(tt.err); got != tt.expected {
				t.Errorf("ClassifyError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestWrapDatabaseError(t *testing.T) {
	originalErr := sql.ErrNoRows
	wrappedErr := WrapDatabaseError("test_operation", originalErr)

	var repoErr *RepositoryError
	if !errors.As(wrappedErr, &repoErr) {
		t.Fatal("Expected wrapped error to be a RepositoryError")
	}

	if repoErr.Op != "test_operation" {
		t.Errorf("Expected Op to be 'test_operation', got %v", repoErr.Op)
	}

	if repoErr.Code != ErrCodeNotFound {
		t.Errorf("Expected Code to be ErrCodeNotFound, got %v", repoErr.Code)
	}

	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected wrapped error to unwrap to original error")
	}
}

func TestWrapDatabaseError_NilError(t *testing.T) {
	wrappedErr := WrapDatabaseError("test_operation", nil)
	if wrappedErr != nil {
		t.Errorf("Expected nil error to remain nil, got %v", wrappedErr)
	}
}

func TestWrapDatabaseErrorWithContext(t *testing.T) {
	originalErr := errors.New("unique constraint failed")
	contextMap := map[string]string{
		"table": "users",
		"field": "email",
	}
	wrappedErr := WrapDatabaseErrorWithContext("insert_user", originalErr, contextMap)

	var repoErr *RepositoryError
	if !errors.As(wrappedErr, &repoErr) {
		t.Fatal("Expected wrapped error to be a RepositoryError")
	}

	if repoErr.Context["table"] != "users" {
		t.Errorf("Expected context table to be 'users', got %v", repoErr.Context["table"])
	}

	if repoErr.Context["field"] != "email" {
		t.Errorf("Expected context field to be 'email', got %v", repoErr.Context["field"])
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name            string
		errorFunc       func() error
		expectedCode    ErrorCode
		expectedContext map[string]string
	}{
		{
			name: "HandleNotFound",
			errorFunc: func() error {
				return HandleNotFound("get_user", "user", "123")
			},
			expectedCode: ErrCodeNotFound,
			expectedContext: map[string]string{
				"resource":   "user",
				"identifier": "123",
			},
		},
		{
			name: "HandleValidationError",
			errorFunc: func() error {
				return HandleValidationError("create_user", "email", "invalid-email", "invalid format")
			},
			expectedCode: ErrCodeValidation,
			expectedContext: map[string]string{
				"field":  "email",
				"value":  "invalid-email",
				"reason": "invalid format",
			},
		},
		{
			name: "HandleConstraintError",
			errorFunc: func() error {
				return HandleConstraintError("insert_user", "unique_email", "email already exists")
			},
			expectedCode: ErrCodeConstraint,
			expectedContext: map[string]string{
				"constraint": "unique_email",
				"details":    "email already exists",
			},
		},
		{
			name: "HandleConnectionError",
			errorFunc: func() error {
				return HandleConnectionError("connect_db", "database is locked")
			},
			expectedCode: ErrCodeConnection,
			expectedContext: map[string]string{
				"details": "database is locked",
			},
		},
		{
			name: "HandleTransactionError",
			errorFunc: func() error {
				return HandleTransactionError("commit_transaction", "commit", "deadlock detected")
			},
			expectedCode: ErrCodeTransaction,
			expectedContext: map[string]string{
				"phase":   "commit",
				"details": "deadlock detected",
			},
		},
		{
			name: "HandleTimeoutError",
			errorFunc: func() error {
				return HandleTimeoutError("query_users", "5s")
			},
			expectedCode: ErrCodeTimeout,
			expectedContext: map[string]string{
				"timeout": "5s",
			},
		},
		{
			name: "HandleDuplicateError",
			errorFunc: func() error {
				return HandleDuplicateError("insert_user", "user", "email", "test@example.com")
			},
			expectedCode: ErrCodeDuplicate,
			expectedContext: map[string]string{
				"resource": "user",
				"field":    "email",
				"value":    "test@example.com",
			},
		},
		{
			name: "HandlePermissionError",
			errorFunc: func() error {
				return HandlePermissionError("delete_user", "user", "delete")
			},
			expectedCode: ErrCodePermission,
			expectedContext: map[string]string{
				"resource": "user",
				"action":   "delete",
			},
		},
		{
			name: "HandleDiskSpaceError",
			errorFunc: func() error {
				return HandleDiskSpaceError("write_data", "/var/lib/db", "100MB")
			},
			expectedCode: ErrCodeDiskSpace,
			expectedContext: map[string]string{
				"path":     "/var/lib/db",
				"required": "100MB",
			},
		},
		{
			name: "HandleCorruptionError",
			errorFunc: func() error {
				return HandleCorruptionError("read_data", "database", "checksum mismatch")
			},
			expectedCode: ErrCodeCorruption,
			expectedContext: map[string]string{
				"resource": "database",
				"details":  "checksum mismatch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFunc()

			var repoErr *RepositoryError
			if !errors.As(err, &repoErr) {
				t.Fatal("Expected error to be a RepositoryError")
			}

			if repoErr.Code != tt.expectedCode {
				t.Errorf("Expected Code to be %v, got %v", tt.expectedCode, repoErr.Code)
			}

			for key, expectedValue := range tt.expectedContext {
				if actualValue, exists := repoErr.Context[key]; !exists {
					t.Errorf("Expected context key '%s' to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected context[%s] to be '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		})
	}
}
