package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

// GetUsageHistory retrieves usage history for the specified number of days
func (r *SQLiteRepository) GetUsageHistory(ctx context.Context, days int) (map[string]*types.UsageData, error) {
	if days <= 0 {
		return nil, repoerrors.NewRepositoryError("GetUsageHistory", errors.New("days must be positive"), repoerrors.ErrCodeConstraint)
	}

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days+1) // Include today

	// Normalize dates
	normalizedStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	normalizedEnd := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	// Get daily usage data
	dailyUsageRows, err := r.queries.GetDailyUsageByDateRange(ctx, queries.GetDailyUsageByDateRangeParams{
		Date:   normalizedStart,
		Date_2: normalizedEnd,
	})
	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetUsageHistory", err, r.classifyError(err))
	}

	// Get app usage data for the same range
	appUsageRows, err := r.queries.GetAppUsageByDateRange(ctx, queries.GetAppUsageByDateRangeParams{
		Date:   normalizedStart,
		Date_2: normalizedEnd,
	})
	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetUsageHistory", err, r.classifyError(err))
	}

	// Build result map
	result := make(map[string]*types.UsageData)

	// Initialize with daily usage data
	for _, dailyRow := range dailyUsageRows {
		dateKey := dailyRow.Date.Format("2006-01-02")
		result[dateKey] = &types.UsageData{
			TotalTime: dailyRow.TotalTime,
			Apps:      []types.AppUsage{},
		}
	}

	// Group app usage by date
	appsByDate := make(map[string][]types.AppUsage)
	for _, appRow := range appUsageRows {
		dateKey := appRow.Date.Format("2006-01-02")
		appsByDate[dateKey] = append(appsByDate[dateKey], r.convertAppUsageFromDB(appRow))
	}

	// Merge app usage into result
	for dateKey, apps := range appsByDate {
		if usageData, exists := result[dateKey]; exists {
			usageData.Apps = apps
		} else {
			// Create entry if daily usage doesn't exist
			totalTime := int64(0)
			for _, app := range apps {
				totalTime += app.Duration
			}
			result[dateKey] = &types.UsageData{
				TotalTime: totalTime,
				Apps:      apps,
			}
		}
	}

	return result, nil
}

// DeleteOldData removes data older than the specified date
func (r *SQLiteRepository) DeleteOldData(ctx context.Context, olderThan time.Time) error {
	// Start transaction for consistency
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repoerrors.NewRepositoryError("DeleteOldData", err, repoerrors.ErrCodeTransaction)
	}

	var committed bool
	defer func(ctx context.Context, olderThan time.Time) {
		if !committed && tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				r.logger.Debug("Failed to rollback transaction in DeleteOldData",
					"rollback_error", rollbackErr,
					"context", "cleanup_after_error",
					"older_than", olderThan.Format("2006-01-02"))
			}
		}
	}(ctx, olderThan)

	txQueries := r.queries.WithTx(tx)

	// Delete old app usage data
	if err := txQueries.DeleteOldAppUsage(ctx, olderThan); err != nil {
		return repoerrors.NewRepositoryError("DeleteOldData", err, r.classifyError(err))
	}

	// Delete old daily usage data
	if err := txQueries.DeleteOldDailyUsage(ctx, olderThan); err != nil {
		return repoerrors.NewRepositoryError("DeleteOldData", err, r.classifyError(err))
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return repoerrors.NewRepositoryError("DeleteOldData", err, repoerrors.ErrCodeTransaction)
	}
	committed = true

	return nil
}

// GetAppUsageByDateRangePaginated retrieves application usage data with pagination metadata for large datasets
func (r *SQLiteRepository) GetAppUsageByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) (*types.PaginatedAppUsageResult, error) {
	// Normalize dates
	normalizedStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	normalizedEnd := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	// Get paginated results
	rows, err := r.queries.GetAppUsageByDateRangePaginated(ctx, queries.GetAppUsageByDateRangePaginatedParams{
		Date:   normalizedStart,
		Date_2: normalizedEnd,
		Limit:  int64(limit),
		Offset: int64(offset),
	})

	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetAppUsageByDateRangePaginated", err, r.classifyError(err))
	}

	// Get total count for pagination metadata
	totalCount, err := r.queries.GetAppUsageCountByDateRange(ctx, queries.GetAppUsageCountByDateRangeParams{
		Date:   normalizedStart,
		Date_2: normalizedEnd,
	})

	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetAppUsageByDateRangePaginated.Count", err, r.classifyError(err))
	}

	// Convert results
	apps := make([]types.AppUsage, len(rows))
	for i, row := range rows {
		apps[i] = r.convertAppUsageFromDB(row)
	}

	return &types.PaginatedAppUsageResult{
		Results: apps,
		Total:   int(totalCount),
	}, nil
}
