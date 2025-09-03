package services

import (
	"context"
	"time"

	"qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

// GetHistoricalUsage retrieves usage data for a specified number of days back from today
func (st *ScreenTimeTracker) GetHistoricalUsage(days int) (map[string]*types.UsageData, error) {
	if st.repository == nil {
		return nil, errors.NewRepositoryError("GetHistoricalUsage", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()
	return st.repository.GetUsageHistory(ctx, days)
}

// GetUsageForDate retrieves usage data for a specific date
func (st *ScreenTimeTracker) GetUsageForDate(date time.Time) (*types.UsageData, error) {
	if st.repository == nil {
		return nil, errors.NewRepositoryError("GetUsageForDate", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()

	// Get daily usage summary
	dailyUsage, err := st.repository.GetDailyUsage(ctx, date)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty usage data if no data exists for this date
			return &types.UsageData{
				TotalTime: 0,
				Apps:      []types.AppUsage{},
			}, nil
		}
		return nil, err
	}

	// Get app usage data for the date
	appUsages, err := st.repository.GetAppUsageByDate(ctx, date)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	// If no app usage data found, return daily summary with empty apps
	if len(appUsages) == 0 {
		return &types.UsageData{
			TotalTime: dailyUsage.TotalTime,
			Apps:      []types.AppUsage{},
		}, nil
	}

	// Sort apps by duration (descending)
	st.sortAppsByDuration(appUsages)

	return &types.UsageData{
		TotalTime: dailyUsage.TotalTime,
		Apps:      appUsages,
	}, nil
}

// GetUsageForDateRange retrieves usage data for a date range
func (st *ScreenTimeTracker) GetUsageForDateRange(startDate, endDate time.Time) ([]types.AppUsage, error) {
	if st.repository == nil {
		return nil, errors.NewRepositoryError("GetUsageForDateRange", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()
	return st.repository.GetAppUsageByDateRange(ctx, startDate, endDate)
}

// GetAppUsageHistory retrieves historical usage data for a specific application
func (st *ScreenTimeTracker) GetAppUsageHistory(appName string, days int) ([]types.AppUsage, error) {
	if st.repository == nil {
		return nil, errors.NewRepositoryError("GetAppUsageHistory", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get app usage data filtered by app name at the database level
	return st.repository.GetAppUsageByNameAndDateRange(ctx, appName, startDate, endDate)
}
