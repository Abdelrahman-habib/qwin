package repository

import (
	"context"
	"testing"
	"time"

	"qwin/internal/types"
)

// TestUsageRepositoryInterface tests that the interface is properly defined
func TestUsageRepositoryInterface(t *testing.T) {
	// This test ensures the interface compiles and has the expected methods
	var _ UsageRepository = (*mockRepository)(nil)
}

// mockRepository is a minimal implementation to test interface compliance
type mockRepository struct{}

func (m *mockRepository) SaveDailyUsage(ctx context.Context, date time.Time, usage *types.UsageData) error {
	return nil
}

func (m *mockRepository) GetDailyUsage(ctx context.Context, date time.Time) (*types.UsageData, error) {
	return &types.UsageData{}, nil
}

func (m *mockRepository) SaveAppUsage(ctx context.Context, date time.Time, appUsage *types.AppUsage) error {
	return nil
}

func (m *mockRepository) GetAppUsageByDate(ctx context.Context, date time.Time) ([]types.AppUsage, error) {
	return []types.AppUsage{}, nil
}

func (m *mockRepository) GetAppUsageByDateRange(ctx context.Context, startDate, endDate time.Time) ([]types.AppUsage, error) {
	return []types.AppUsage{}, nil
}

func (m *mockRepository) GetUsageHistory(ctx context.Context, days int) (map[string]*types.UsageData, error) {
	return make(map[string]*types.UsageData), nil
}

func (m *mockRepository) DeleteOldData(ctx context.Context, olderThan time.Time) error {
	return nil
}

func (m *mockRepository) WithTransaction(ctx context.Context, fn func(repo UsageRepository) error) error {
	return fn(m)
}

func (m *mockRepository) BatchProcessAppUsage(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy) error {
	return nil
}

func (m *mockRepository) BatchIncrementAppUsageDurations(ctx context.Context, date time.Time, increments map[string]int64) error {
	return nil
}

func (m *mockRepository) GetAppUsageByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) (*types.PaginatedAppUsageResult, error) {
	return &types.PaginatedAppUsageResult{
		Results: []types.AppUsage{},
		Total:   0,
	}, nil
}

func (m *mockRepository) GetAppUsageByNameAndDateRange(ctx context.Context, appName string, startDate, endDate time.Time) ([]types.AppUsage, error) {
	return []types.AppUsage{}, nil
}
