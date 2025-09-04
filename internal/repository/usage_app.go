package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

// SaveAppUsage saves or updates application usage data with retry logic
func (r *SQLiteRepository) SaveAppUsage(ctx context.Context, date time.Time, appUsage *types.AppUsage) error {
	start := time.Now()

	if appUsage == nil {
		err := repoerrors.NewRepositoryError("SaveAppUsage", fmt.Errorf("app usage data is nil"), repoerrors.ErrCodeValidation)
		logging.LogError(r.logger, err, "SaveAppUsage", map[string]interface{}{
			"date": date.Format("2006-01-02"),
		})
		return err
	}

	// Validate app usage fields
	// Build shared context for validation errors
	validationContext := map[string]string{
		"date":     date.Format("2006-01-02"),
		"app_name": appUsage.Name,
		"duration": fmt.Sprintf("%d", appUsage.Duration),
	}
	
	if strings.TrimSpace(appUsage.Name) == "" {
		err := repoerrors.NewRepositoryErrorWithContext("SaveAppUsage", fmt.Errorf("app name is empty or whitespace"), repoerrors.ErrCodeValidation, validationContext)
		// Convert context to interface{} map for logging
		logContext := make(map[string]interface{}, len(validationContext))
		for k, v := range validationContext {
			logContext[k] = v
		}
		logging.LogError(r.logger, err, "SaveAppUsage", logContext)
		return err
	}

	if appUsage.Duration < 0 {
		err := repoerrors.NewRepositoryErrorWithContext("SaveAppUsage", fmt.Errorf("app duration is negative: %d", appUsage.Duration), repoerrors.ErrCodeValidation, validationContext)
		// Convert context to interface{} map for logging
		logContext := make(map[string]interface{}, len(validationContext))
		for k, v := range validationContext {
			logContext[k] = v
		}
		logging.LogError(r.logger, err, "SaveAppUsage", logContext)
		return err
	}

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Execute with retry logic
	err := repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		_, err := r.queries.UpsertAppUsage(ctx, queries.UpsertAppUsageParams{
			Name:     appUsage.Name,
			Duration: appUsage.Duration,
			IconPath: r.nullStringFromString(appUsage.IconPath),
			ExePath:  r.nullStringFromString(appUsage.ExePath),
			Date:     normalizedDate,
		})

		if err != nil {
			repoErr := repoerrors.NewRepositoryErrorWithContext("SaveAppUsage", err, r.classifyError(err), map[string]string{
				"app_name": appUsage.Name,
				"date":     normalizedDate.Format("2006-01-02"),
				"duration": fmt.Sprintf("%d", appUsage.Duration),
			})

			// Log retryable errors at debug level, non-retryable at error level
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in SaveAppUsage", "error", err, "app", appUsage.Name)
			} else {
				logging.LogError(r.logger, repoErr, "SaveAppUsage", map[string]interface{}{
					"app_name": appUsage.Name,
					"date":     normalizedDate.Format("2006-01-02"),
					"duration": appUsage.Duration,
				})
			}

			return repoErr
		}

		return nil
	})

	// Log successful operation
	if err == nil {
		logging.LogOperation(r.logger, "SaveAppUsage", time.Since(start), map[string]interface{}{
			"app_name": appUsage.Name,
			"date":     normalizedDate.Format("2006-01-02"),
			"duration": appUsage.Duration,
		})
	}

	return err
}

// GetAppUsageByDate retrieves all application usage data for a specific date
func (r *SQLiteRepository) GetAppUsageByDate(ctx context.Context, date time.Time) ([]types.AppUsage, error) {
	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	rows, err := r.queries.GetAppUsageByDate(ctx, normalizedDate)
	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetAppUsageByDate", err, r.classifyError(err))
	}

	apps := make([]types.AppUsage, len(rows))
	for i, row := range rows {
		apps[i] = r.convertAppUsageFromDB(row)
	}

	return apps, nil
}

// GetAppUsageByDateRange retrieves application usage data for a date range.
// Results are ordered by date descending (newest first) and then by duration descending.
// Both start and end date bounds are inclusive.
func (r *SQLiteRepository) GetAppUsageByDateRange(ctx context.Context, startDate, endDate time.Time) ([]types.AppUsage, error) {
	// Normalize dates
	normalizedStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	normalizedEnd := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	rows, err := r.queries.GetAppUsageByDateRange(ctx, queries.GetAppUsageByDateRangeParams{
		Date:   normalizedStart,
		Date_2: normalizedEnd,
	})

	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetAppUsageByDateRange", err, r.classifyError(err))
	}

	apps := make([]types.AppUsage, len(rows))
	for i, row := range rows {
		apps[i] = r.convertAppUsageFromDB(row)
	}

	return apps, nil
}

// GetAppUsageByNameAndDateRange retrieves application usage data for a specific app within a date range
func (r *SQLiteRepository) GetAppUsageByNameAndDateRange(ctx context.Context, appName string, startDate, endDate time.Time) ([]types.AppUsage, error) {
	// Normalize dates
	normalizedStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	normalizedEnd := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	rows, err := r.queries.GetAppUsageByNameAndDateRange(ctx, queries.GetAppUsageByNameAndDateRangeParams{
		Name:   appName,
		Date:   normalizedStart,
		Date_2: normalizedEnd,
	})

	if err != nil {
		return nil, repoerrors.NewRepositoryError("GetAppUsageByNameAndDateRange", err, r.classifyError(err))
	}

	apps := make([]types.AppUsage, len(rows))
	for i, row := range rows {
		apps[i] = r.convertAppUsageFromDB(row)
	}

	return apps, nil
}
