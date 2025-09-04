package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
)

// Configuration methods

// SetRetryConfig updates the retry configuration for the repository
func (r *SQLiteRepository) SetRetryConfig(config *repoerrors.RetryConfig) {
	if config != nil {
		r.retryConfig = config
	}
}

// SetLogger updates the logger for the repository
func (r *SQLiteRepository) SetLogger(logger logging.Logger) {
	if logger != nil {
		r.logger = logger
	}
}

// SetBatchConfig updates the batch configuration for the repository
func (r *SQLiteRepository) SetBatchConfig(config *BatchConfig) {
	if config != nil {
		r.batchConfig = config
	}
}

// GetBatchConfig returns the current batch configuration
func (r *SQLiteRepository) GetBatchConfig() *BatchConfig {
	return r.batchConfig
}

// GetRetryConfig returns the current retry configuration
func (r *SQLiteRepository) GetRetryConfig() *repoerrors.RetryConfig {
	return r.retryConfig
}

// SetDynamicBatchSize updates batch size configuration at runtime based on operation type
func (r *SQLiteRepository) SetDynamicBatchSize(operationType string, batchSize int) error {
	if r.batchConfig == nil {
		return repoerrors.NewRepositoryError("SetDynamicBatchSize",
			errors.New("batch configuration is not set"), repoerrors.ErrCodeValidation)
	}
	if batchSize <= 0 {
		return repoerrors.NewRepositoryError("SetDynamicBatchSize",
			errors.New("batch size must be positive"), repoerrors.ErrCodeValidation)
	}

	if batchSize > r.batchConfig.MaxBatchSize {
		return repoerrors.NewRepositoryError("SetDynamicBatchSize",
			fmt.Errorf("batch size %d exceeds maximum allowed %d", batchSize, r.batchConfig.MaxBatchSize),
			repoerrors.ErrCodeValidation)
	}

	// For now, we update the default batch size
	// In a more sophisticated implementation, we could have operation-specific batch sizes
	r.batchConfig.DefaultBatchSize = batchSize

	if r.logger != nil {
		r.logger.Debug("Updated batch size configuration",
			"operation_type", operationType,
			"new_batch_size", batchSize)
	}

	return nil
}

// Health check method with comprehensive error reporting
func (r *SQLiteRepository) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// Test basic connectivity
	err := repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		if err := r.db.PingContext(ctx); err != nil {
			repoErr := repoerrors.NewRepositoryError("HealthCheck.Ping", err, r.classifyError(err))
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in health check ping", "error", err)
			} else {
				logging.LogError(r.logger, repoErr, "HealthCheck.Ping", nil)
			}
			return repoErr
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Test a simple query
	err = repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		var count int
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
		if err != nil {
			repoErr := repoerrors.NewRepositoryError("HealthCheck.Query", err, r.classifyError(err))
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error in health check query", "error", err)
			} else {
				logging.LogError(r.logger, repoErr, "HealthCheck.Query", nil)
			}
			return repoErr
		}
		return nil
	})

	// Log successful health check
	if err == nil {
		logging.LogOperation(r.logger, "HealthCheck", time.Since(start), nil)
	}

	return err
}
