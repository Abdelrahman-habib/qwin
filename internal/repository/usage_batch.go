package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

// Package-level errors
var (
	ErrInvalidBatchSize = errors.New("invalid batch size")
)

// BatchProcessAppUsage processes multiple application usage records with specified strategy
func (r *SQLiteRepository) BatchProcessAppUsage(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy) error {
	return r.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, strategy, 0)
}

// BatchProcessAppUsageWithBatchSize processes multiple application usage records with specified strategy and custom batch size
// If batchSize is 0, uses the configured default batch size calculation
func (r *SQLiteRepository) BatchProcessAppUsageWithBatchSize(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy, batchSize int) error {
	start := time.Now()

	// Validate batch size to prevent undefined behavior/infinite loops
	if batchSize < 0 {
		err := repoerrors.NewRepositoryError("BatchProcessAppUsageWithBatchSize", ErrInvalidBatchSize, repoerrors.ErrCodeValidation)
		logging.LogError(r.logger, err, "BatchProcessAppUsageWithBatchSize", map[string]interface{}{
			"batch_size": batchSize,
			"date":       date.Format("2006-01-02"),
		})
		return err
	}

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

			// If record is missing, insert instead of dropping the increment
			if errors.Is(err, sql.ErrNoRows) {
				if insertErr := txRepo.queries.InsertAppUsage(ctx, queries.InsertAppUsageParams{
					Name:     appName,
					Duration: additionalDuration,
					IconPath: txRepo.nullStringFromString(""),
					ExePath:  txRepo.nullStringFromString(""),
					Date:     normalizedDate,
				}); insertErr != nil {
					return repoerrors.NewRepositoryErrorWithContext(
						"BatchIncrementAppUsageDurations",
						insertErr,
						r.classifyError(insertErr),
						map[string]string{
							"app_name":            appName,
							"date":                normalizedDate.Format("2006-01-02"),
							"additional_duration": fmt.Sprintf("%d", additionalDuration),
							"operation":           "InsertOnMissing",
						},
					)
				}
				continue
			}

			// Check for overflow if record exists
			if err == nil {
				if currentApp.Duration > math.MaxInt64-additionalDuration {
					return repoerrors.NewRepositoryErrorWithContext(
						"BatchIncrementAppUsageDurations",
						errors.New("duration increment would cause integer overflow"),
						repoerrors.ErrCodeValidation,
						map[string]string{
							"app_name":            appName,
							"current_duration":    fmt.Sprintf("%d", currentApp.Duration),
							"additional_duration": fmt.Sprintf("%d", additionalDuration),
							"max_int64":           fmt.Sprintf("%d", int64(math.MaxInt64)),
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
	// Provide safe fallback values if batchConfig is nil or has invalid values
	defaultBatch := 100
	maxBatch := 1000

	if r.batchConfig != nil {
		if r.batchConfig.DefaultBatchSize > 0 {
			defaultBatch = r.batchConfig.DefaultBatchSize
		}
		if r.batchConfig.MaxBatchSize > 0 {
			maxBatch = r.batchConfig.MaxBatchSize
		}
	}

	// Clamp totalItems to reasonable bounds
	if totalItems <= 0 {
		return 1
	}
	if totalItems <= defaultBatch {
		return totalItems
	}

	// For large datasets, use a smaller batch size to avoid long transactions
	if totalItems > maxBatch*2 {
		// Ensure we don't divide by zero or get a result less than 1
		if defaultBatch > 1 {
			return max(defaultBatch/2, 1)
		}
		return 1
	}

	// For medium datasets, use default batch size
	if totalItems > maxBatch {
		return defaultBatch
	}

	// Clamp the result to ensure it's within bounds
	return min(max(defaultBatch, 1), maxBatch)
}

// calculateOptimalBatchSizeWithStrategy determines batch size based on strategy and context
func (r *SQLiteRepository) calculateOptimalBatchSizeWithStrategy(totalItems int, strategy types.BatchStrategy, ctx context.Context) int {
	baseSize := r.calculateOptimalBatchSize(totalItems)

	// Provide safe fallback values if batchConfig is nil or has invalid values
	maxBatch := 1000
	if r.batchConfig != nil && r.batchConfig.MaxBatchSize > 0 {
		maxBatch = r.batchConfig.MaxBatchSize
	}

	// Adjust batch size based on strategy
	switch strategy {
	case types.BatchStrategyInsertOnly:
		// Insert-only operations can handle larger batches since there's no conflict resolution
		if totalItems > 1000 {
			return min(baseSize*2, maxBatch)
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
