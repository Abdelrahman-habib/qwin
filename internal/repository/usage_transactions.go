package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
)

// WithTransaction executes a function within a database transaction with retry logic
func (r *SQLiteRepository) WithTransaction(ctx context.Context, fn func(repo UsageRepository) error) error {
	start := time.Now()

	// Execute transaction with retry logic
	err := repoerrors.WithRetry(ctx, r.retryConfig, func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			repoErr := repoerrors.NewRepositoryError("WithTransaction.Begin", err, r.classifyError(err))
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error beginning transaction", "error", err)
			} else {
				logging.LogError(r.logger, repoErr, "WithTransaction.Begin", nil)
			}
			return repoErr
		}

		var originalErr error
		var committed bool
		defer func(ctx context.Context) {
			if !committed && tx != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
					r.logger.Debug("Failed to rollback transaction in WithTransaction",
						"rollback_error", rollbackErr,
						"original_error", originalErr,
						"context", "transaction_cleanup")
				}
			}
		}(ctx)

		// Create a new repository instance with the transaction
		txRepo := &SQLiteRepository{
			db:          r.db, // Keep original db for other operations
			queries:     r.queries.WithTx(tx),
			dbService:   r.dbService,
			retryConfig: r.retryConfig,
			batchConfig: r.batchConfig,
			logger:      r.logger,
		}

		// Execute the function with the transaction repository
		if err := fn(txRepo); err != nil {
			// Log transaction function error but don't wrap it
			// The function should return proper repository errors
			originalErr = err
			r.logger.Debug("Transaction function failed", "error", err)
			return err
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			originalErr = err
			repoErr := repoerrors.NewRepositoryError("WithTransaction.Commit", err, r.classifyError(err))
			if repoErr.IsRetryable() {
				r.logger.Debug("Retryable error committing transaction", "error", err)
			} else {
				logging.LogError(r.logger, repoErr, "WithTransaction.Commit", nil)
			}
			return repoErr
		}
		committed = true

		return nil
	})

	// Log successful transaction
	if err == nil {
		logging.LogOperation(r.logger, "WithTransaction", time.Since(start), nil)
	}

	return err
}
