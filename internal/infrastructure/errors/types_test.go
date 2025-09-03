package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrCodeNotFound, "NOT_FOUND"},
		{ErrCodeDuplicate, "DUPLICATE"},
		{ErrCodeConstraint, "CONSTRAINT"},
		{ErrCodeConnection, "CONNECTION"},
		{ErrCodeTransaction, "TRANSACTION"},
		{ErrCodeTimeout, "TIMEOUT"},
		{ErrCodeValidation, "VALIDATION"},
		{ErrCodePermission, "PERMISSION"},
		{ErrCodeDiskSpace, "DISK_SPACE"},
		{ErrCodeCorruption, "CORRUPTION"},
		{ErrCodeUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ErrorCode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepositoryError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RepositoryError
		contains []string
	}{
		{
			name: "basic error",
			err: &RepositoryError{
				Op:   "test_operation",
				Err:  errors.New("test error"),
				Code: ErrCodeNotFound,
			},
			contains: []string{"test error", "op=test_operation", "code=NOT_FOUND"},
		},
		{
			name: "error with context",
			err: &RepositoryError{
				Op:   "test_operation",
				Err:  errors.New("test error"),
				Code: ErrCodeNotFound,
				Context: map[string]string{
					"table": "users",
					"id":    "123",
				},
			},
			contains: []string{"test error", "op=test_operation", "code=NOT_FOUND", "table=users", "id=123"},
		},
		{
			name: "retryable error",
			err: &RepositoryError{
				Op:        "test_operation",
				Err:       errors.New("test error"),
				Code:      ErrCodeConnection,
				Retryable: true,
			},
			contains: []string{"test error", "op=test_operation", "code=CONNECTION", "retryable=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, contain := range tt.contains {
				if !strings.Contains(errStr, contain) {
					t.Errorf("RepositoryError.Error() = %v, should contain %v", errStr, contain)
				}
			}
		})
	}
}

func TestRepositoryError_Is(t *testing.T) {
	err1 := &RepositoryError{Code: ErrCodeNotFound}
	err2 := &RepositoryError{Code: ErrCodeNotFound}
	err3 := &RepositoryError{Code: ErrCodeDuplicate}
	otherErr := errors.New("other error")

	if !errors.Is(err1, err2) {
		t.Error("Expected errors with same code to match")
	}

	if errors.Is(err1, err3) {
		t.Error("Expected errors with different codes not to match")
	}

	if errors.Is(err1, otherErr) {
		t.Error("Expected repository error not to match non-repository error")
	}

	// Test wrapped error matching
	wrappedErr := errors.New("wrapped error")
	repoErrWithWrapped := &RepositoryError{
		Code: ErrCodeConnection,
		Err:  wrappedErr,
	}

	if !errors.Is(repoErrWithWrapped, wrappedErr) {
		t.Error("Expected repository error to match its wrapped error")
	}

	if errors.Is(repoErrWithWrapped, otherErr) {
		t.Error("Expected repository error not to match different wrapped error")
	}
}

func TestRepositoryError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	repoErr := &RepositoryError{Err: originalErr}

	if unwrapped := repoErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", originalErr, unwrapped)
	}
}

func TestRepositoryError_WithContext(t *testing.T) {
	err := &RepositoryError{}
	err = err.WithContext("key1", "value1")
	err = err.WithContext("key2", "value2")

	if err.Context["key1"] != "value1" {
		t.Errorf("Expected context key1 to be 'value1', got %v", err.Context["key1"])
	}

	if err.Context["key2"] != "value2" {
		t.Errorf("Expected context key2 to be 'value2', got %v", err.Context["key2"])
	}
}

func TestNewRepositoryError(t *testing.T) {
	originalErr := errors.New("test error")
	repoErr := NewRepositoryError("test_op", originalErr, ErrCodeNotFound)

	if repoErr.Op != "test_op" {
		t.Errorf("Expected Op to be 'test_op', got %v", repoErr.Op)
	}

	if repoErr.Err != originalErr {
		t.Errorf("Expected Err to be %v, got %v", originalErr, repoErr.Err)
	}

	if repoErr.Code != ErrCodeNotFound {
		t.Errorf("Expected Code to be ErrCodeNotFound, got %v", repoErr.Code)
	}

	if repoErr.Context == nil {
		t.Error("Expected Context to be initialized")
	}

	if repoErr.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
}

func TestNewRepositoryErrorWithContext(t *testing.T) {
	originalErr := errors.New("test error")
	context := map[string]string{"key": "value"}
	repoErr := NewRepositoryErrorWithContext("test_op", originalErr, ErrCodeNotFound, context)

	if repoErr.Context["key"] != "value" {
		t.Errorf("Expected context key to be 'value', got %v", repoErr.Context["key"])
	}
}

func TestErrorClassificationFunctions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		testFunc func(error) bool
		expected bool
	}{
		{"IsNotFound with RepositoryError", NewRepositoryError("op", nil, ErrCodeNotFound), IsNotFound, true},
		{"IsNotFound with other error", errors.New("other"), IsNotFound, false},
		{"IsDuplicate with RepositoryError", NewRepositoryError("op", nil, ErrCodeDuplicate), IsDuplicate, true},
		{"IsConstraint with RepositoryError", NewRepositoryError("op", nil, ErrCodeConstraint), IsConstraint, true},
		{"IsConnection with RepositoryError", NewRepositoryError("op", nil, ErrCodeConnection), IsConnection, true},
		{"IsTransaction with RepositoryError", NewRepositoryError("op", nil, ErrCodeTransaction), IsTransaction, true},
		{"IsTimeout with RepositoryError", NewRepositoryError("op", nil, ErrCodeTimeout), IsTimeout, true},
		{"IsValidation with RepositoryError", NewRepositoryError("op", nil, ErrCodeValidation), IsValidation, true},
		{"IsPermission with RepositoryError", NewRepositoryError("op", nil, ErrCodePermission), IsPermission, true},
		{"IsDiskSpace with RepositoryError", NewRepositoryError("op", nil, ErrCodeDiskSpace), IsDiskSpace, true},
		{"IsCorruption with RepositoryError", NewRepositoryError("op", nil, ErrCodeCorruption), IsCorruption, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.testFunc(tt.err); got != tt.expected {
				t.Errorf("Function returned %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		err      error
		expected bool
	}{
		{"Connection error is retryable", ErrCodeConnection, nil, true},
		{"Timeout error is retryable", ErrCodeTimeout, nil, true},
		{"Transaction error is retryable", ErrCodeTransaction, nil, true},
		{"Disk space error is not retryable", ErrCodeDiskSpace, nil, false},
		{"Not found error is not retryable", ErrCodeNotFound, nil, false},
		{"Duplicate error is not retryable", ErrCodeDuplicate, nil, false},
		{"Constraint error is not retryable", ErrCodeConstraint, nil, false},
		{"Validation error is not retryable", ErrCodeValidation, nil, false},
		{"Permission error is not retryable", ErrCodePermission, nil, false},
		{"Corruption error is not retryable", ErrCodeCorruption, nil, false},
		{"Unknown error with 'temporary' is retryable", ErrCodeUnknown, errors.New("temporary failure"), true},
		{"Unknown error with 'retry' is retryable", ErrCodeUnknown, errors.New("please retry"), true},
		{"Unknown error with 'busy' is retryable", ErrCodeUnknown, errors.New("database busy"), true},
		{"Unknown error without keywords is not retryable", ErrCodeUnknown, errors.New("permanent failure"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.code, tt.err); got != tt.expected {
				t.Errorf("isRetryableError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	retryableErr := NewRepositoryError("op", nil, ErrCodeConnection)
	nonRetryableErr := NewRepositoryError("op", nil, ErrCodeNotFound)
	otherErr := errors.New("other error")

	if !IsRetryable(retryableErr) {
		t.Error("Expected retryable error to return true")
	}

	if IsRetryable(nonRetryableErr) {
		t.Error("Expected non-retryable error to return false")
	}

	if IsRetryable(otherErr) {
		t.Error("Expected non-repository error to return false")
	}
}
