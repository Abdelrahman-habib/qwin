package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

// SaveDailyUsage saves or updates daily usage data with retry logic
func (r *SQLiteRepository) SaveDailyUsage(ctx context.Context, date time.Time, usage *types.UsageData) error {
	start := time.Now()

	if usage == nil {
		err := repoerrors.NewRepositoryError("SaveDailyUsage", errors.New("usage data is nil"), repoerrors.ErrCodeValidation)
		logging.LogError(r.logger, err, "SaveDailyUsage", map[string]any{
			"date": date.Format("2006-01-02"),
		})
		return err
	}

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Execute with retry logic
	err := repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		_, err := r.queries.UpsertDailyUsage(ctx, queries.UpsertDailyUsageParams{
			Date:      normalizedDate,
			TotalTime: usage.TotalTime,
		})

		if err != nil {
			repoErr := repoerrors.NewRepositoryErrorWithContext("SaveDailyUsage", err, r.classifyError(err), map[string]string{
				"date":       normalizedDate.Format("2006-01-02"),
				"total_time": fmt.Sprintf("%d", usage.TotalTime),
			})

			// Log retryable errors at debug level, non-retryable at error level
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in SaveDailyUsage", "error", err, "date", normalizedDate)
			} else {
				logging.LogError(r.logger, repoErr, "SaveDailyUsage", map[string]any{
					"date":       normalizedDate.Format("2006-01-02"),
					"total_time": usage.TotalTime,
				})
			}

			return repoErr
		}

		return nil
	})

	// Log successful operation
	if err == nil {
		logging.LogOperation(r.logger, "SaveDailyUsage", time.Since(start), map[string]any{
			"date":       normalizedDate.Format("2006-01-02"),
			"total_time": usage.TotalTime,
		})
	}

	return err
}

// GetDailyUsage retrieves daily usage data for a specific date with enhanced error handling
func (r *SQLiteRepository) GetDailyUsage(ctx context.Context, date time.Time) (*types.UsageData, error) {
	start := time.Now()

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	var result *types.UsageData

	// Execute with retry logic for transient errors
	err := repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		// Get daily usage summary
		dailyUsage, err := r.queries.GetDailyUsageByDate(ctx, normalizedDate)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Not found is not retryable
				return repoerrors.NewRepositoryErrorWithContext("GetDailyUsage", err, repoerrors.ErrCodeNotFound, map[string]string{
					"date": normalizedDate.Format("2006-01-02"),
				})
			}

			repoErr := repoerrors.NewRepositoryErrorWithContext("GetDailyUsage", err, r.classifyError(err), map[string]string{
				"date":      normalizedDate.Format("2006-01-02"),
				"operation": "GetDailyUsageByDate",
			})

			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in GetDailyUsage", "error", err, "date", normalizedDate)
			} else {
				logging.LogError(r.logger, repoErr, "GetDailyUsage", map[string]any{
					"date":      normalizedDate.Format("2006-01-02"),
					"operation": "GetDailyUsageByDate",
				})
			}

			return repoErr
		}

		// Get app usage for the date
		appUsageRows, err := r.queries.GetAppUsageByDate(ctx, normalizedDate)
		if err != nil {
			repoErr := repoerrors.NewRepositoryErrorWithContext("GetDailyUsage", err, r.classifyError(err), map[string]string{
				"date":      normalizedDate.Format("2006-01-02"),
				"operation": "GetAppUsageByDate",
			})

			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in GetDailyUsage", "error", err, "date", normalizedDate)
			} else {
				logging.LogError(r.logger, repoErr, "GetDailyUsage", map[string]interface{}{
					"date":      normalizedDate.Format("2006-01-02"),
					"operation": "GetAppUsageByDate",
				})
			}

			return repoErr
		}

		// Convert to types.AppUsage
		apps := make([]types.AppUsage, len(appUsageRows))
		for i, row := range appUsageRows {
			apps[i] = r.convertAppUsageFromDB(row)
		}

		result = &types.UsageData{
			TotalTime: dailyUsage.TotalTime,
			Apps:      apps,
		}

		return nil
	})

	// Log successful operation
	if err == nil {
		logging.LogOperation(r.logger, "GetDailyUsage", time.Since(start), map[string]interface{}{
			"date":      normalizedDate.Format("2006-01-02"),
			"app_count": len(result.Apps),
		})
	}

	return result, err
}
