package repository

import (
	"context"
	"time"

	"qwin/internal/types"
)

// UsageRepository defines the interface for usage data persistence operations
type UsageRepository interface {
	// Daily usage operations
	SaveDailyUsage(ctx context.Context, date time.Time, usage *types.UsageData) error
	GetDailyUsage(ctx context.Context, date time.Time) (*types.UsageData, error)

	// Application usage operations
	SaveAppUsage(ctx context.Context, date time.Time, appUsage *types.AppUsage) error
	GetAppUsageByDate(ctx context.Context, date time.Time) ([]types.AppUsage, error)
	GetAppUsageByDateRange(ctx context.Context, startDate, endDate time.Time) ([]types.AppUsage, error)

	// Historical data operations
	GetUsageHistory(ctx context.Context, days int) (map[string]*types.UsageData, error)
	DeleteOldData(ctx context.Context, olderThan time.Time) error

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo UsageRepository) error) error

	// Batch operations for performance
	// BatchProcessAppUsage performs batch operations with specified strategy:
	// - BatchStrategyInsertOnly: insert-only operations, failing on conflicts
	// - BatchStrategyUpsert: upsert operations, updating existing records on conflicts
	BatchProcessAppUsage(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy) error
	BatchIncrementAppUsageDurations(ctx context.Context, date time.Time, increments map[string]int64) error

	// Pagination for large datasets with metadata
	// Returns paginated results along with total count for UI rendering without additional queries
	GetAppUsageByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) (*types.PaginatedAppUsageResult, error)

	// Filtered queries for efficiency
	GetAppUsageByNameAndDateRange(ctx context.Context, appName string, startDate, endDate time.Time) ([]types.AppUsage, error)
}
