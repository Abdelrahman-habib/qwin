package errors

import (
	"testing"

	"github.com/mattn/go-sqlite3"
)

func TestClassifySQLiteError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrCodeUnknown,
		},
		{
			name:     "non-sqlite error",
			err:      &customError{msg: "some other error"},
			expected: ErrCodeUnknown,
		},
		{
			name: "unique constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintUnique,
			},
			expected: ErrCodeDuplicate,
		},
		{
			name: "primary key constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintPrimaryKey,
			},
			expected: ErrCodeDuplicate,
		},
		{
			name: "foreign key constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintForeignKey,
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "not null constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintNotNull,
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "check constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintCheck,
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "trigger constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintTrigger,
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "rowid constraint violation",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: sqlite3.ErrConstraintRowID,
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "generic constraint with unique in message",
			err: sqlite3.Error{
				Code:         sqlite3.ErrConstraint,
				ExtendedCode: 0, // No extended code
			},
			expected: ErrCodeConstraint,
		},
		{
			name: "database corruption",
			err: sqlite3.Error{
				Code: sqlite3.ErrCorrupt,
			},
			expected: ErrCodeCorruption,
		},
		{
			name: "permission denied",
			err: sqlite3.Error{
				Code: sqlite3.ErrPerm,
			},
			expected: ErrCodePermission,
		},
		{
			name: "database busy",
			err: sqlite3.Error{
				Code: sqlite3.ErrBusy,
			},
			expected: ErrCodeBusy,
		},
		{
			name: "database locked",
			err: sqlite3.Error{
				Code: sqlite3.ErrLocked,
			},
			expected: ErrCodeBusy,
		},
		{
			name: "disk full",
			err: sqlite3.Error{
				Code: sqlite3.ErrFull,
			},
			expected: ErrCodeDiskSpace,
		},
		{
			name: "misuse error",
			err: sqlite3.Error{
				Code: sqlite3.ErrMisuse,
			},
			expected: ErrCodeInternal,
		},
		{
			name: "schema error",
			err: sqlite3.Error{
				Code: sqlite3.ErrSchema,
			},
			expected: ErrCodeSchema,
		},
		{
			name: "unknown sqlite error",
			err: sqlite3.Error{
				Code: sqlite3.ErrRange, // Some other error code
			},
			expected: ErrCodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifySQLiteError(tt.err)
			if result != tt.expected {
				t.Errorf("classifySQLiteError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyError_SQLiteIntegration(t *testing.T) {
	// Test that the main ClassifyError function properly uses SQLite-specific classification
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "SQLite unique constraint violation via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique},
			expected: ErrCodeDuplicate,
		},
		{
			name:     "SQLite foreign key constraint violation via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintForeignKey},
			expected: ErrCodeConstraint,
		},
		{
			name:     "SQLite database locked via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrLocked},
			expected: ErrCodeBusy,
		},
		{
			name:     "SQLite database corrupt via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrCorrupt},
			expected: ErrCodeCorruption,
		},
		{
			name:     "SQLite disk full via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrFull},
			expected: ErrCodeDiskSpace,
		},
		{
			name:     "SQLite permission denied via ClassifyError",
			err:      sqlite3.Error{Code: sqlite3.ErrPerm},
			expected: ErrCodePermission,
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

// customError is a helper type for testing non-sqlite errors
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
