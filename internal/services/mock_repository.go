package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"qwin/internal/infrastructure/errors"
	"qwin/internal/repository"
	"qwin/internal/types"
)

// MockRepository implements the UsageRepository interface for testing
type MockRepository struct {
	mu               sync.RWMutex
	dailyUsage       map[string]*types.UsageData // key: date string (YYYY-MM-DD)
	appUsage         map[string][]types.AppUsage // key: date string (YYYY-MM-DD)
	saveCallCount    int
	loadCallCount    int
	batchCallCount   int
	transactionCalls int
	shouldFailSave   bool
	shouldFailLoad   bool
	shouldFailBatch  bool
	shouldFailTx     bool
	deleteCallCount  int
	historyCallCount int
}

// NewMockRepository creates a new mock repository for testing
func NewMockRepository() *MockRepository {
	return &MockRepository{
		dailyUsage: make(map[string]*types.UsageData),
		appUsage:   make(map[string][]types.AppUsage),
	}
}

// SetFailureModes configures the mock to simulate failures
func (m *MockRepository) SetFailureModes(save, load, batch, tx bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailSave = save
	m.shouldFailLoad = load
	m.shouldFailBatch = batch
	m.shouldFailTx = tx
}

// GetCallCounts returns the number of times each method was called
func (m *MockRepository) GetCallCounts() (save, load, batch, tx, delete, history int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveCallCount, m.loadCallCount, m.batchCallCount, m.transactionCalls, m.deleteCallCount, m.historyCallCount
}

// SaveDailyUsage implements UsageRepository interface
func (m *MockRepository) SaveDailyUsage(ctx context.Context, date time.Time, usage *types.UsageData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCallCount++

	if m.shouldFailSave {
		return errors.NewRepositoryError("SaveDailyUsage", fmt.Errorf("mock save failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	m.dailyUsage[dateKey] = &types.UsageData{
		TotalTime: usage.TotalTime,
		Apps:      make([]types.AppUsage, len(usage.Apps)),
	}
	copy(m.dailyUsage[dateKey].Apps, usage.Apps)

	return nil
}

// GetDailyUsage implements UsageRepository interface
func (m *MockRepository) GetDailyUsage(ctx context.Context, date time.Time) (*types.UsageData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadCallCount++

	if m.shouldFailLoad {
		return nil, errors.NewRepositoryError("GetDailyUsage", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	usage, exists := m.dailyUsage[dateKey]
	if !exists {
		return nil, errors.NewRepositoryError("GetDailyUsage", fmt.Errorf("not found"), errors.ErrCodeNotFound)
	}

	// Return a copy to avoid race conditions
	result := &types.UsageData{
		TotalTime: usage.TotalTime,
		Apps:      make([]types.AppUsage, len(usage.Apps)),
	}
	copy(result.Apps, usage.Apps)

	return result, nil
}

// SaveAppUsage implements UsageRepository interface
func (m *MockRepository) SaveAppUsage(ctx context.Context, date time.Time, appUsage *types.AppUsage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailSave {
		return errors.NewRepositoryError("SaveAppUsage", fmt.Errorf("mock save failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	if m.appUsage[dateKey] == nil {
		m.appUsage[dateKey] = []types.AppUsage{}
	}

	// Update existing or add new
	found := false
	for i, existing := range m.appUsage[dateKey] {
		if existing.Name == appUsage.Name {
			m.appUsage[dateKey][i] = *appUsage
			found = true
			break
		}
	}

	if !found {
		m.appUsage[dateKey] = append(m.appUsage[dateKey], *appUsage)
	}

	return nil
}

// GetAppUsageByDate implements UsageRepository interface
func (m *MockRepository) GetAppUsageByDate(ctx context.Context, date time.Time) ([]types.AppUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailLoad {
		return nil, errors.NewRepositoryError("GetAppUsageByDate", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	apps, exists := m.appUsage[dateKey]
	if !exists {
		return []types.AppUsage{}, nil
	}

	// Return a copy to avoid race conditions
	result := make([]types.AppUsage, len(apps))
	copy(result, apps)

	return result, nil
}

// GetAppUsageByDateRange implements UsageRepository interface
func (m *MockRepository) GetAppUsageByDateRange(ctx context.Context, startDate, endDate time.Time) ([]types.AppUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailLoad {
		return nil, errors.NewRepositoryError("GetAppUsageByDateRange", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	var result []types.AppUsage

	// Iterate through date range
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		if apps, exists := m.appUsage[dateKey]; exists {
			result = append(result, apps...)
		}
	}

	// Sort results by date DESC then duration DESC to match repository contract
	sort.Slice(result, func(i, j int) bool {
		// Parse dates, treating parse errors as zero time
		dateI, errI := time.Parse("2006-01-02", result[i].Date.Format("2006-01-02"))
		if errI != nil {
			dateI = time.Time{}
		}
		dateJ, errJ := time.Parse("2006-01-02", result[j].Date.Format("2006-01-02"))
		if errJ != nil {
			dateJ = time.Time{}
		}

		// First sort by date DESC (newer dates come first)
		if !dateI.Equal(dateJ) {
			return dateI.After(dateJ)
		}

		// For equal dates, sort by Duration DESC
		return result[i].Duration > result[j].Duration
	})

	return result, nil
}

// GetUsageHistory implements UsageRepository interface
func (m *MockRepository) GetUsageHistory(ctx context.Context, days int) (map[string]*types.UsageData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.historyCallCount++

	if m.shouldFailLoad {
		return nil, errors.NewRepositoryError("GetUsageHistory", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	result := make(map[string]*types.UsageData)

	// Filter by the last N days (inclusive of today) using YYYY-MM-DD string compare to avoid TZ skew
	if days <= 0 {
		return result, nil
	}
	// Compute local start-of-day explicitly to avoid DST drift
	t := time.Now().In(time.Local)
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	earliestKey := startOfDay.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	for dateKey, usage := range m.dailyUsage {
		if dateKey < earliestKey {
			continue
		}
		result[dateKey] = &types.UsageData{
			TotalTime: usage.TotalTime,
			Apps:      make([]types.AppUsage, len(usage.Apps)),
		}
		copy(result[dateKey].Apps, usage.Apps)
	}

	return result, nil
}

// DeleteOldData implements UsageRepository interface
func (m *MockRepository) DeleteOldData(ctx context.Context, olderThan time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCallCount++

	if m.shouldFailSave {
		return errors.NewRepositoryError("DeleteOldData", fmt.Errorf("mock delete failure"), errors.ErrCodeConnection)
	}

	// Remove data older than the specified date
	for dateKey := range m.dailyUsage {
		if date, err := time.Parse("2006-01-02", dateKey); err == nil && date.Before(olderThan) {
			delete(m.dailyUsage, dateKey)
			delete(m.appUsage, dateKey)
		}
	}

	return nil
}

// WithTransaction implements UsageRepository interface
func (m *MockRepository) WithTransaction(ctx context.Context, fn func(repo repository.UsageRepository) error) error {
	m.mu.Lock()
	m.transactionCalls++
	m.mu.Unlock()

	if m.shouldFailTx {
		return errors.NewRepositoryError("WithTransaction", fmt.Errorf("mock transaction failure"), errors.ErrCodeTransaction)
	}

	// Execute the function with this mock repository
	return fn(m)
}

// BatchProcessAppUsage implements UsageRepository interface
func (m *MockRepository) BatchProcessAppUsage(ctx context.Context, date time.Time, appUsages []types.AppUsage, strategy types.BatchStrategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.batchCallCount++

	if m.shouldFailBatch {
		return errors.NewRepositoryError("BatchProcessAppUsage", fmt.Errorf("mock batch failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	if m.appUsage[dateKey] == nil {
		m.appUsage[dateKey] = []types.AppUsage{}
	}

	// Handle different strategies
	for _, appUsage := range appUsages {
		found := false
		for i, existing := range m.appUsage[dateKey] {
			if existing.Name == appUsage.Name {
				switch strategy {
				case types.BatchStrategyUpsert:
					m.appUsage[dateKey][i] = appUsage
				case types.BatchStrategyInsertOnly:
					// Skip if exists for insert-only
				}
				found = true
				break
			}
		}

		if !found {
			m.appUsage[dateKey] = append(m.appUsage[dateKey], appUsage)
		}
	}

	return nil
}

// BatchIncrementAppUsageDurations implements UsageRepository interface
func (m *MockRepository) BatchIncrementAppUsageDurations(ctx context.Context, date time.Time, increments map[string]int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailBatch {
		return errors.NewRepositoryError("BatchIncrementAppUsageDurations", fmt.Errorf("mock batch failure"), errors.ErrCodeConnection)
	}

	dateKey := date.Format("2006-01-02")
	if m.appUsage[dateKey] == nil {
		return nil
	}

	// Increment durations
	for i, app := range m.appUsage[dateKey] {
		if additionalDuration, exists := increments[app.Name]; exists {
			m.appUsage[dateKey][i].Duration += additionalDuration
		}
	}

	return nil
}

// GetAppUsageByDateRangePaginated implements UsageRepository interface
func (m *MockRepository) GetAppUsageByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) (*types.PaginatedAppUsageResult, error) {
	m.mu.RLock()
	shouldFail := m.shouldFailLoad
	m.mu.RUnlock()

	if shouldFail {
		return nil, errors.NewRepositoryError("GetAppUsageByDateRangePaginated", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	// Get all data first
	allApps, err := m.GetAppUsageByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	total := len(allApps)

	// Validate and clamp pagination parameters
	// Ensure offset is not negative
	if offset < 0 {
		offset = 0
	}

	// Return empty result if limit is <= 0
	if limit <= 0 {
		return &types.PaginatedAppUsageResult{
			Results: []types.AppUsage{},
			Total:   total,
		}, nil
	}

	// Apply pagination with bounds checking
	start := offset
	if start >= len(allApps) {
		return &types.PaginatedAppUsageResult{
			Results: []types.AppUsage{},
			Total:   total,
		}, nil
	}

	end := start + limit
	if end > len(allApps) {
		end = len(allApps)
	}

	return &types.PaginatedAppUsageResult{
		Results: allApps[start:end],
		Total:   total,
	}, nil
}

// GetAppUsageByNameAndDateRange implements UsageRepository interface
func (m *MockRepository) GetAppUsageByNameAndDateRange(ctx context.Context, appName string, startDate, endDate time.Time) ([]types.AppUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailLoad {
		return nil, errors.NewRepositoryError("GetAppUsageByNameAndDateRange", fmt.Errorf("mock load failure"), errors.ErrCodeConnection)
	}

	var result []types.AppUsage

	// Iterate through date range and filter by app name
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		if apps, exists := m.appUsage[dateKey]; exists {
			for _, app := range apps {
				if app.Name == appName {
					result = append(result, app)
				}
			}
		}
	}

	return result, nil
}
