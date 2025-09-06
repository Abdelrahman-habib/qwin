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
		{ErrCodeInternal, "INTERNAL"},
		{ErrCodeBusy, "BUSY"},
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
		{"IsInternal with RepositoryError", NewRepositoryError("op", nil, ErrCodeInternal), IsInternal, true},
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
		{"Busy error is retryable", ErrCodeBusy, nil, true},
		{"Disk space error is not retryable", ErrCodeDiskSpace, nil, false},
		{"Not found error is not retryable", ErrCodeNotFound, nil, false},
		{"Duplicate error is not retryable", ErrCodeDuplicate, nil, false},
		{"Constraint error is not retryable", ErrCodeConstraint, nil, false},
		{"Validation error is not retryable", ErrCodeValidation, nil, false},
		{"Permission error is not retryable", ErrCodePermission, nil, false},
		{"Corruption error is not retryable", ErrCodeCorruption, nil, false},
		{"Internal error is not retryable", ErrCodeInternal, nil, false},
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
	busyErr := NewRepositoryError("op", nil, ErrCodeBusy)
	nonRetryableErr := NewRepositoryError("op", nil, ErrCodeNotFound)
	otherErr := errors.New("other error")

	if !IsRetryable(retryableErr) {
		t.Error("Expected retryable error to return true")
	}

	if !IsRetryable(busyErr) {
		t.Error("Expected busy error to return true")
	}

	if IsRetryable(nonRetryableErr) {
		t.Error("Expected non-retryable error to return false")
	}

	if IsRetryable(otherErr) {
		t.Error("Expected non-repository error to return false")
	}
}

func TestRepositoryError_Error_NilSafe(t *testing.T) {
	// Test nil receiver doesn't panic and returns default message
	var nilErr *RepositoryError
	result := nilErr.Error()
	if result != "repository error" {
		t.Errorf("Expected nil receiver to return 'repository error', got %v", result)
	}
}

func TestRepositoryError_Error_DeterministicContext(t *testing.T) {
	// Test that context keys are output in deterministic order
	err := &RepositoryError{
		Op:   "test_op",
		Err:  errors.New("test error"),
		Code: ErrCodeValidation,
		Context: map[string]string{
			"zebra": "last",
			"alpha": "first",
			"beta":  "second",
		},
	}

	// Call Error() multiple times and verify same output
	result1 := err.Error()
	result2 := err.Error()
	result3 := err.Error()

	if result1 != result2 {
		t.Errorf("Error() output not deterministic: %v != %v", result1, result2)
	}

	if result1 != result3 {
		t.Errorf("Error() output not deterministic: %v != %v", result1, result3)
	}

	// Verify context keys appear in alphabetical order
	// Should see: alpha=first beta=second zebra=last
	expectedOrder := []string{"alpha=first", "beta=second", "zebra=last"}
	for _, expected := range expectedOrder {
		if !strings.Contains(result1, expected) {
			t.Errorf("Expected output to contain %v, got %v", expected, result1)
		}
	}

	// Verify alphabetical ordering by checking alpha comes before zebra
	alphaPos := strings.Index(result1, "alpha=first")
	betaPos := strings.Index(result1, "beta=second")
	zebraPos := strings.Index(result1, "zebra=last")

	if alphaPos == -1 || betaPos == -1 || zebraPos == -1 {
		t.Errorf("Context keys not found in output: %v", result1)
	}

	if alphaPos > betaPos || betaPos > zebraPos {
		t.Errorf("Context keys not in alphabetical order in output: %v", result1)
	}
}

func TestRepositoryError_Error_NilContext(t *testing.T) {
	// Test that nil Context is treated as empty
	err := &RepositoryError{
		Op:      "test_op",
		Err:     errors.New("test error"),
		Code:    ErrCodeNotFound,
		Context: nil, // explicitly nil
	}

	result := err.Error()
	expected := "test error [op=test_op code=NOT_FOUND]"
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// Should not panic or include any context fields
	if strings.Contains(result, "<nil>") {
		t.Errorf("Output should not contain nil references: %v", result)
	}
}

func TestRepositoryError_NilReceiverGuards(t *testing.T) {
	// Test all accessor methods with nil receiver to ensure they don't panic
	var nilErr *RepositoryError

	// Test Unwrap
	if unwrapped := nilErr.Unwrap(); unwrapped != nil {
		t.Errorf("Expected nil.Unwrap() to return nil, got %v", unwrapped)
	}

	// Test IsRetryable
	if nilErr.IsRetryable() {
		t.Error("Expected nil.IsRetryable() to return false")
	}

	// Test GetCode
	if code := nilErr.GetCode(); code != "UNKNOWN" {
		t.Errorf("Expected nil.GetCode() to return UNKNOWN string, got %v", code)
	}

	// Test GetContext
	context := nilErr.GetContext()
	if context == nil {
		t.Error("Expected nil.GetContext() to return empty map, got nil")
	}
	if len(context) != 0 {
		t.Errorf("Expected nil.GetContext() to return empty map, got %v", context)
	}

	// Test GetTimestamp
	if timestamp := nilErr.GetTimestamp(); !timestamp.IsZero() {
		t.Errorf("Expected nil.GetTimestamp() to return zero time, got %v", timestamp)
	}

	// Test Error (already tested above, but include for completeness)
	if nilErr.Error() != "repository error" {
		t.Errorf("Expected nil.Error() to return 'repository error', got %v", nilErr.Error())
	}
}

func TestNewRepositoryErrorWithContext_ClonesContext(t *testing.T) {
	// Test that NewRepositoryErrorWithContext clones the context map to prevent mutations
	originalContext := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// Create error with context
	err := NewRepositoryErrorWithContext("test_op", nil, ErrCodeValidation, originalContext)

	// Verify the context is copied
	if err.Context["key1"] != "value1" {
		t.Errorf("Expected context key1 to be 'value1', got %v", err.Context["key1"])
	}
	if err.Context["key2"] != "value2" {
		t.Errorf("Expected context key2 to be 'value2', got %v", err.Context["key2"])
	}

	// Mutate the original context
	originalContext["key1"] = "modified_value1"
	originalContext["key3"] = "new_value3"

	// Verify the error's context is not affected by the mutation
	if err.Context["key1"] != "value1" {
		t.Errorf("Expected error context key1 to remain 'value1' after original mutation, got %v", err.Context["key1"])
	}
	if _, exists := err.Context["key3"]; exists {
		t.Errorf("Expected error context to not have key3 after original mutation, but it exists with value %v", err.Context["key3"])
	}

	// Verify mutating the error's context doesn't affect the original
	err.Context["key2"] = "error_modified_value2"
	if originalContext["key2"] != "value2" {
		t.Errorf("Expected original context key2 to remain 'value2' after error mutation, got %v", originalContext["key2"])
	}
}

func TestNewRepositoryErrorWithContext_NilContext(t *testing.T) {
	// Test that nil context is handled safely
	err := NewRepositoryErrorWithContext("test_op", nil, ErrCodeValidation, nil)
	if err.Context == nil {
		t.Error("Expected error to have non-nil context even when nil is passed")
	}
	if len(err.Context) != 0 {
		t.Errorf("Expected error context to be empty when nil is passed, got %v", err.Context)
	}
}

func TestWithContext_MutationSemantics(t *testing.T) {
	// Test that WithContext mutates the receiver (not creating a copy)
	err1 := NewRepositoryError("test_op", nil, ErrCodeValidation)
	err2 := err1.WithContext("key1", "value1")

	// Should return the same instance (mutation, not copy)
	if err1 != err2 {
		t.Error("Expected WithContext to return the same instance (mutation semantics)")
	}

	// Both references should see the change
	if err1.Context["key1"] != "value1" {
		t.Errorf("Expected err1 context to have key1='value1', got %v", err1.Context["key1"])
	}
	if err2.Context["key1"] != "value1" {
		t.Errorf("Expected err2 context to have key1='value1', got %v", err2.Context["key1"])
	}

	// Chaining should work
	err3 := err1.WithContext("key2", "value2").WithContext("key3", "value3")
	if err1 != err3 {
		t.Error("Expected chained WithContext to return the same instance")
	}

	if len(err1.Context) != 3 {
		t.Errorf("Expected err1 context to have 3 keys after chaining, got %d", len(err1.Context))
	}
}

func TestIsRetryableError_EnhancedHeuristics(t *testing.T) {
	// Test the enhanced heuristics for unknown errors
	tests := []struct {
		name        string
		errorMsg    string
		expectRetry bool
	}{
		// Existing heuristics
		{"temporary error", "temporary failure occurred", true},
		{"retry message", "please retry the operation", true},
		{"busy message", "database is busy", true},
		// New heuristics
		{"locked message", "table is locked", true},
		{"database locked", "database is locked by another process", true},
		{"deadlock message", "deadlock detected", true},
		{"transaction deadlock", "transaction deadlock occurred", true},
		// Case variations
		{"uppercase LOCKED", "TABLE IS LOCKED", true},
		{"mixed case Deadlock", "Transaction Deadlock Detected", true},
		// Non-matching messages
		{"permanent error", "permanent failure - data corrupted", false},
		{"unrelated message", "invalid input format", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an error with ErrCodeUnknown so it falls back to heuristics
			testErr := errors.New(tt.errorMsg)
			got := isRetryableError(ErrCodeUnknown, testErr)
			if got != tt.expectRetry {
				t.Errorf("isRetryableError() = %v, expected %v for message '%s'", got, tt.expectRetry, tt.errorMsg)
			}
		})
	}
}
