package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

func TestScreenTimeTracker_HistoricalUsage(t *testing.T) {
	mockRepo := NewMockRepository()

	// Pre-populate with historical data using recent dates
	ctx := context.Background()
	baseTime := time.Now().Truncate(24 * time.Hour) // Use current date as base
	for i := 0; i < 7; i++ {
		date := baseTime.AddDate(0, 0, -i)
		usage := &types.UsageData{
			TotalTime: int64(3600 * (i + 1)), // Different total times
		}
		mockRepo.SaveDailyUsage(ctx, date, usage)
	}

	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Get historical usage
	history, err := tracker.GetHistoricalUsage(7)
	if err != nil {
		t.Errorf("GetHistoricalUsage() unexpected error = %v", err)
	}

	if len(history) != 7 {
		t.Errorf("GetHistoricalUsage() returned %d days, want 7", len(history))
	}

	// Verify repository was called
	_, _, _, _, _, historyCount := mockRepo.GetCallCounts()
	if historyCount == 0 {
		t.Error("GetHistoricalUsage() did not call repository")
	}
}

func TestScreenTimeTracker_GetUsageForDate(t *testing.T) {
	mockRepo := NewMockRepository()

	// Pre-populate with test data
	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	testUsage := &types.UsageData{
		TotalTime: 7200,
		Apps: []types.AppUsage{
			{Name: "TestApp", Duration: 3600},
		},
	}

	ctx := context.Background()
	mockRepo.SaveDailyUsage(ctx, testDate, testUsage)
	for i := range testUsage.Apps {
		mockRepo.SaveAppUsage(ctx, testDate, &testUsage.Apps[i])
	}

	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Test getting existing data
	usage, err := tracker.GetUsageForDate(testDate)
	if err != nil {
		t.Errorf("GetUsageForDate() unexpected error = %v", err)
	}

	if usage.TotalTime != 7200 {
		t.Errorf("GetUsageForDate() TotalTime = %d, want 7200", usage.TotalTime)
	}

	if len(usage.Apps) != 1 {
		t.Errorf("GetUsageForDate() returned %d apps, want 1", len(usage.Apps))
	}

	// Test getting non-existent data
	futureDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	emptyUsage, err := tracker.GetUsageForDate(futureDate)
	if err != nil {
		t.Errorf("GetUsageForDate() for non-existent date unexpected error = %v", err)
	}

	if emptyUsage.TotalTime != 0 {
		t.Errorf("GetUsageForDate() for non-existent date TotalTime = %d, want 0", emptyUsage.TotalTime)
	}

	if len(emptyUsage.Apps) != 0 {
		t.Errorf("GetUsageForDate() for non-existent date returned %d apps, want 0", len(emptyUsage.Apps))
	}
}

func TestScreenTimeTracker_GetUsageForDateRange(t *testing.T) {
	mockRepo := NewMockRepository()

	// Pre-populate with test data across multiple days
	ctx := context.Background()
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		date := startDate.AddDate(0, 0, i)
		app := types.AppUsage{
			Name:     fmt.Sprintf("RangeApp%d", i),
			Duration: int64(1800 * (i + 1)),
		}
		mockRepo.SaveAppUsage(ctx, date, &app)
	}

	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Test date range query
	apps, err := tracker.GetUsageForDateRange(startDate, endDate)
	if err != nil {
		t.Errorf("GetUsageForDateRange() unexpected error = %v", err)
	}

	if len(apps) != 3 {
		t.Errorf("GetUsageForDateRange() returned %d apps, want 3", len(apps))
	}
}

func TestScreenTimeTracker_GetAppUsageHistory(t *testing.T) {
	mockRepo := NewMockRepository()

	// Pre-populate with test data for specific app using current time range
	ctx := context.Background()
	targetApp := "HistoryTestApp"
	now := time.Now()

	for i := 0; i < 5; i++ {
		date := now.AddDate(0, 0, -i)
		app := types.AppUsage{
			Name:     targetApp,
			Duration: int64(1800 * (i + 1)),
		}
		mockRepo.SaveAppUsage(ctx, date, &app)

		// Add some other apps to test filtering
		otherApp := types.AppUsage{
			Name:     fmt.Sprintf("OtherApp%d", i),
			Duration: int64(900),
		}
		mockRepo.SaveAppUsage(ctx, date, &otherApp)
	}

	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Test app-specific history
	history, err := tracker.GetAppUsageHistory(targetApp, 7)
	if err != nil {
		t.Errorf("GetAppUsageHistory() unexpected error = %v", err)
	}

	if len(history) != 5 {
		t.Errorf("GetAppUsageHistory() returned %d entries, want 5", len(history))
	}

	// Verify all entries are for the target app
	for _, entry := range history {
		if entry.Name != targetApp {
			t.Errorf("GetAppUsageHistory() returned entry for %s, want %s", entry.Name, targetApp)
		}
	}
}

func TestScreenTimeTracker_GetAppUsageByNameAndDateRange(t *testing.T) {
	mockRepo := NewMockRepository()

	// Pre-populate with test data for multiple apps across multiple days
	ctx := context.Background()
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	targetApp := "FilterTestApp"

	for i := 0; i < 3; i++ {
		date := startDate.AddDate(0, 0, i)

		// Add target app
		targetAppUsage := types.AppUsage{
			Name:     targetApp,
			Duration: int64(1800 * (i + 1)),
		}
		mockRepo.SaveAppUsage(ctx, date, &targetAppUsage)

		// Add other apps that should be filtered out
		otherApp := types.AppUsage{
			Name:     fmt.Sprintf("OtherApp%d", i),
			Duration: int64(900),
		}
		mockRepo.SaveAppUsage(ctx, date, &otherApp)
	}

	// Test the repository method directly
	filteredApps, err := mockRepo.GetAppUsageByNameAndDateRange(ctx, targetApp, startDate, endDate)
	if err != nil {
		t.Errorf("GetAppUsageByNameAndDateRange() unexpected error = %v", err)
	}

	if len(filteredApps) != 3 {
		t.Errorf("GetAppUsageByNameAndDateRange() returned %d apps, want 3", len(filteredApps))
	}

	// Verify all entries are for the target app
	for _, app := range filteredApps {
		if app.Name != targetApp {
			t.Errorf("GetAppUsageByNameAndDateRange() returned entry for %s, want %s", app.Name, targetApp)
		}
	}

	// Verify durations are correct
	expectedDurations := []int64{1800, 3600, 5400}
	for i, app := range filteredApps {
		if app.Duration != expectedDurations[i] {
			t.Errorf("GetAppUsageByNameAndDateRange() app %d duration = %d, want %d", i, app.Duration, expectedDurations[i])
		}
	}
}
