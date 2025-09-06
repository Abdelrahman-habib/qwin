package errors

import (
	"errors"
	"strings"

	"github.com/mattn/go-sqlite3"
)

// classifySQLiteError attempts to classify SQLite-specific errors using type assertions
// Returns the appropriate ErrorCode if the error is a sqlite3.Error, otherwise returns ErrCodeUnknown
func classifySQLiteError(err error) ErrorCode {
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return ErrCodeUnknown
	}

	// First check extended error codes for more specific classification
	switch sqliteErr.ExtendedCode {
	// Constraint violations - extended codes
	case sqlite3.ErrConstraintUnique, sqlite3.ErrConstraintPrimaryKey:
		return ErrCodeDuplicate
	case sqlite3.ErrConstraintForeignKey:
		return ErrCodeConstraint
	case sqlite3.ErrConstraintCheck:
		return ErrCodeConstraint
	case sqlite3.ErrConstraintNotNull:
		return ErrCodeConstraint
	case sqlite3.ErrConstraintTrigger, sqlite3.ErrConstraintRowID:
		return ErrCodeConstraint
	}

	// Then check base error codes for broader categories
	switch sqliteErr.Code {
	case sqlite3.ErrConstraint:
		// Generic constraint error - check the error message for more specifics
		errStr := strings.ToLower(sqliteErr.Error())
		if strings.Contains(errStr, "unique") {
			return ErrCodeDuplicate
		}
		return ErrCodeConstraint

	// Database corruption
	case sqlite3.ErrCorrupt, sqlite3.ErrNotADB:
		return ErrCodeCorruption

	// Permission and access errors
	case sqlite3.ErrPerm, sqlite3.ErrAuth:
		return ErrCodePermission
	case sqlite3.ErrReadonly:
		return ErrCodePermission

	// Connection and I/O errors
	case sqlite3.ErrBusy, sqlite3.ErrLocked:
		return ErrCodeBusy
	case sqlite3.ErrCantOpen:
		return ErrCodeConnection
	case sqlite3.ErrIoErr:
		return ErrCodeConnection

	// Disk space errors
	case sqlite3.ErrFull:
		return ErrCodeDiskSpace

	// API misuse errors
	case sqlite3.ErrMisuse:
		// Indicates incorrect API usage (e.g., calling prepared statement after finalizing)
		// This is a programming error, not a transient transaction failure
		return ErrCodeInternal

	// Schema errors (indicate database schema/migration problems)
	case sqlite3.ErrSchema:
		return ErrCodeSchema

	default:
		return ErrCodeUnknown
	}
}
