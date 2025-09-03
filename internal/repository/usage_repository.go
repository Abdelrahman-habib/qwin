package repository

import (
	"context"
	"database/sql"
	"fmt"

	"qwin/internal/database"
	queries "qwin/internal/database/generated"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
)

// BatchConfig holds configuration for batch operations
type BatchConfig struct {
	DefaultBatchSize int
	MaxBatchSize     int
}

// DefaultBatchConfig returns sensible defaults for batch operations
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		DefaultBatchSize: 100,
		MaxBatchSize:     1000,
	}
}

// SQLiteRepository implements the UsageRepository interface using SQLite
type SQLiteRepository struct {
	db          *sql.DB
	queries     *queries.Queries
	dbService   database.Service
	retryConfig *repoerrors.RetryConfig
	batchConfig *BatchConfig
	logger      logging.Logger
}

// NewSQLiteRepository creates a new SQLite repository instance
func NewSQLiteRepository(dbService database.Service, logger logging.Logger) *SQLiteRepository {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &SQLiteRepository{
		db:          dbService.DB(),
		queries:     dbService.GetQueries(),
		dbService:   dbService,
		retryConfig: repoerrors.DefaultRetryConfig(),
		batchConfig: DefaultBatchConfig(),
		logger:      logger,
	}
}

// NewSQLiteRepositoryWithConfig creates a new SQLite repository instance with custom configuration
func NewSQLiteRepositoryWithConfig(dbService database.Service, retryConfig *repoerrors.RetryConfig, batchConfig *BatchConfig, logger logging.Logger) *SQLiteRepository {
	if retryConfig == nil {
		retryConfig = repoerrors.DefaultRetryConfig()
	}
	if batchConfig == nil {
		batchConfig = DefaultBatchConfig()
	}
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &SQLiteRepository{
		db:          dbService.DB(),
		queries:     dbService.GetQueries(),
		dbService:   dbService,
		retryConfig: retryConfig,
		batchConfig: batchConfig,
		logger:      logger,
	}
}

// NewSQLiteRepositoryWithPreparedQueries creates a repository with prepared statements for better performance
func NewSQLiteRepositoryWithPreparedQueries(ctx context.Context, dbService database.Service, logger logging.Logger) (*SQLiteRepository, error) {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	preparedQueries, err := dbService.GetPreparedQueries(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewSQLiteRepositoryWithPreparedQueries: failed to get prepared queries from database service: %w", err)
	}

	return &SQLiteRepository{
		db:          dbService.DB(),
		queries:     preparedQueries,
		dbService:   dbService,
		retryConfig: repoerrors.DefaultRetryConfig(),
		batchConfig: DefaultBatchConfig(),
		logger:      logger,
	}, nil
}
