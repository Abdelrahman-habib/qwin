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

// BatchProcessAppUsage processes multiple application usage records with specified strategy
func (r *SQLiteRepository) BatchProcessAppUsage(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy) error {
	return r.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, strategy, 0)
}

// BatchProcessAppUsageWithBatchSize processes multiple application usage records with specified strategy and custom batch size
// If batchSize is 0, uses the configured default batch size calculation
func (r *SQLiteRepository) BatchProcessAppUsageWithBatchSize(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy, batchSize int) error {
	start := time.Now()

	if len(appUsages) == 0 {
		return nil
	}

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	strategyName := "upsert"
	if strategy == types.BatchStrategyInsertOnly {
		strategyName = "insert"
	}

	// Process in configurable batches to avoid memory issues and long transactions
	effectiveBatchSize := batchSize
	if batchSize == 0 {
		effectiveBatchSize = r.calculateOptimalBatchSizeWithStrategy(len(appUsages), strategy, ctx)
	}

	for i := 0; i < len(appUsages); i += effectiveBatchSize {
		end := i + effectiveBatchSize
		if end > len(appUsages) {
			end = len(appUsages)
		}

		batch := appUsages[i:end]

		err := r.WithTransaction(ctx, func(repo UsageRepository) error {
			txRepo := repo.(*SQLiteRepository)

			for j, appUsage := range batch {
				var err error

				switch strategy {
				case types.BatchStrategyUpsert:
					_, err = txRepo.queries.UpsertAppUsage(ctx, queries.UpsertAppUsageParams{
						Name:     appUsage.Name,
						Duration: appUsage.Duration,
						IconPath: r.nullStringFromString(appUsage.IconPath),
						ExePath:  r.nullStringFromString(appUsage.ExePath),
						Date:     normalizedDate,
					})
				case types.BatchStrategyInsertOnly:
					err = txRepo.queries.InsertAppUsage(ctx, queries.InsertAppUsageParams{
						Name:     appUsage.Name,
						Duration: appUsage.Duration,
						IconPath: r.nullStringFromString(appUsage.IconPath),
						ExePath:  r.nullStringFromString(appUsage.ExePath),
						Date:     normalizedDate,
					})
				default:
					return repoerrors.NewRepositoryErrorWithContext("BatchProcessAppUsage",
						fmt.Errorf("unsupported batch strategy: %d", strategy),
						repoerrors.ErrCodeValidation, map[string]string{
							"strategy": fmt.Sprintf("%d", strategy),
						})
				}

				if err != nil {
					repoErr := repoerrors.NewRepositoryErrorWithContext("BatchProcessAppUsage", err, r.classifyError(err), map[string]string{
						"app_name":    appUsage.Name,
						"date":        normalizedDate.Format("2006-01-02"),
						"batch_index": fmt.Sprintf("%d", i+j),
						"batch_size":  fmt.Sprintf("%d", len(batch)),
						"total_size":  fmt.Sprintf("%d", len(appUsages)),
						"strategy":    strategyName,
					})

					logging.LogError(r.logger, repoErr, "BatchProcessAppUsage", map[string]any{
						"app_name":    appUsage.Name,
						"date":        normalizedDate.Format("2006-01-02"),
						"batch_index": i + j,
						"batch_size":  len(batch),
						"total_size":  len(appUsages),
						"strategy":    strategyName,
					})

					return repoErr
				}
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	// Log successful batch operation
	logging.LogOperation(r.logger, "BatchProcessAppUsage", time.Since(start), map[string]any{
		"date":       normalizedDate.Format("2006-01-02"),
		"total_size": len(appUsages),
		"batch_size": effectiveBatchSize,
		"strategy":   strategyName,
	})

	return nil
}

// BatchIncrementAppUsageDurations increments multiple app usage durations efficiently
// additionalDuration values must be non-negative to prevent data corruption
func (r *SQLiteRepository) BatchIncrementAppUsageDurations(ctx context.Context, date time.Time, increments map[string]int64) error {
	if len(increments) == 0 {
		return nil
	}

	// Validate all increments before performing any updates
	for appName, additionalDuration := range increments {
		if additionalDuration < 0 {
			return repoerrors.NewRepositoryErrorWithContext(
				"BatchIncrementAppUsageDurations",
				errors.New("negative increment not allowed"),
				repoerrors.ErrCodeValidation,
				map[string]string{
					"app_name":            appName,
					"additional_duration": fmt.Sprintf("%d", additionalDuration),
				},
			)
		}
	}

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	return r.WithTransaction(ctx, func(repo UsageRepository) error {
		txRepo := repo.(*SQLiteRepository)

		for appName, additionalDuration := range increments {
			// Get current duration to check for overflow
			currentApp, err := txRepo.queries.GetAppUsageByNameAndDate(ctx, queries.GetAppUsageByNameAndDateParams{
				Name: appName,
				Date: normalizedDate,
			})

			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return repoerrors.NewRepositoryErrorWithContext(
					"BatchIncrementAppUsageDurations",
					err,
					r.classifyError(err),
					map[string]string{
						"app_name":  appName,
						"date":      normalizedDate.Format("2006-01-02"),
						"operation": "GetCurrentDuration",
					},
				)
			}

			// Check for overflow if record exists
			if err == nil {
				const maxInt64 = 9223372036854775807
				if currentApp.Duration > maxInt64-additionalDuration {
					return repoerrors.NewRepositoryErrorWithContext(
						"BatchIncrementAppUsageDurations",
						errors.New("duration increment would cause integer overflow"),
						repoerrors.ErrCodeValidation,
						map[string]string{
							"app_name":            appName,
							"current_duration":    fmt.Sprintf("%d", currentApp.Duration),
							"additional_duration": fmt.Sprintf("%d", additionalDuration),
							"max_int64":           fmt.Sprintf("%d", maxInt64),
						},
					)
				}
			}

			// Perform the increment
			err = txRepo.queries.BatchUpdateAppUsage(ctx, queries.BatchUpdateAppUsageParams{
				Duration: additionalDuration,
				Name:     appName,
				Date:     normalizedDate,
			})
			if err != nil {
				return repoerrors.NewRepositoryErrorWithContext(
					"BatchIncrementAppUsageDurations",
					err,
					r.classifyError(err),
					map[string]string{
						"app_name":            appName,
						"date":                normalizedDate.Format("2006-01-02"),
						"additional_duration": fmt.Sprintf("%d", additionalDuration),
					},
				)
			}
		}
		return nil
	})
}

// calculateOptimalBatchSize determines the best batch size based on total items
func (r *SQLiteRepository) calculateOptimalBatchSize(totalItems int) int {
	if totalItems <= r.batchConfig.DefaultBatchSize {
		return totalItems
	}

	// For large datasets, use a smaller batch size to avoid long transactions
	if totalItems > r.batchConfig.MaxBatchSize*2 {
		return r.batchConfig.DefaultBatchSize / 2
	}

	// For medium datasets, use default batch size
	if totalItems > r.batchConfig.MaxBatchSize {
		return r.batchConfig.DefaultBatchSize
	}

	return r.batchConfig.DefaultBatchSize
}

// calculateOptimalBatchSizeWithStrategy determines batch size based on strategy and context
func (r *SQLiteRepository) calculateOptimalBatchSizeWithStrategy(totalItems int, strategy types.BatchStrategy, ctx context.Context) int {
	baseSize := r.calculateOptimalBatchSize(totalItems)

	// Adjust batch size based on strategy
	switch strategy {
	case types.BatchStrategyInsertOnly:
		// Insert-only operations can handle larger batches since there's no conflict resolution
		if totalItems > 1000 {
			return min(baseSize*2, r.batchConfig.MaxBatchSize)
		}
		return baseSize

	case types.BatchStrategyUpsert:
		// Upsert operations require more processing per item, so use smaller batches
		if totalItems > 500 {
			return max(baseSize/2, 50) // Minimum batch size of 50
		}
		return baseSize

	default:
		return baseSize
	}
}
