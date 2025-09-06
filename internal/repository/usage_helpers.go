package repository

import (
	"database/sql"
	"time"

	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

// Helper functions for batch size calculations
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// convertAppUsageFromDB converts database AppUsage to types.AppUsage
func (r *SQLiteRepository) convertAppUsageFromDB(dbApp queries.AppUsage) types.AppUsage {
	return types.AppUsage{
		ID:        dbApp.ID,
		Name:      dbApp.Name,
		Duration:  dbApp.Duration,
		IconPath:  r.stringFromNullString(dbApp.IconPath),
		ExePath:   r.stringFromNullString(dbApp.ExePath),
		Date:      dbApp.Date,
		CreatedAt: r.timeFromNullTime(dbApp.CreatedAt),
		UpdatedAt: r.timeFromNullTime(dbApp.UpdatedAt),
	}
}

// nullStringFromString converts string to sql.NullString
func (r *SQLiteRepository) nullStringFromString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// stringFromNullString converts sql.NullString to string
func (r *SQLiteRepository) stringFromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// timeFromNullTime converts sql.NullTime to time.Time
func (r *SQLiteRepository) timeFromNullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

// classifyError classifies database errors into repository error codes
func (r *SQLiteRepository) classifyError(err error) repoerrors.ErrorCode {
	return repoerrors.ClassifyError(err)
}
