package repository

import (
	"context"
	"testing"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

func TestSQLiteRepository_SaveAndGetAppUsage(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	appUsage := &types.AppUsage{
		Name:     "TestApplication",
		Duration: 3600,
		IconPath: "/path/to/icon.png",
		ExePath:  "/path/to/app.exe",
	}

	// Test SaveAppUsage
	err := repo.SaveAppUsage(ctx, date, appUsage)
	if err != nil {
		t.Fatalf("SaveAppUsage failed: %v", err)
	}

	// Test GetAppUsageByDate
	apps, err := repo.GetAppUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("GetAppUsageByDate failed: %v", err)
	}

	if len(apps) != 1 {
		t.Fatalf("Expected 1 app, got %d", len(apps))
	}

	retrievedApp := apps[0]
	if retrievedApp.Name != appUsage.Name {
		t.Errorf("Expected app name %s, got %s", appUsage.Name, retrievedApp.Name)
	}

	if retrievedApp.Duration != appUsage.Duration {
		t.Errorf("Expected duration %d, got %d", appUsage.Duration, retrievedApp.Duration)
	}

	if retrievedApp.IconPath != appUsage.IconPath {
		t.Errorf("Expected icon path %s, got %s", appUsage.IconPath, retrievedApp.IconPath)
	}

	if retrievedApp.ExePath != appUsage.ExePath {
		t.Errorf("Expected exe path %s, got %s", appUsage.ExePath, retrievedApp.ExePath)
	}
}

func TestSQLiteRepository_SaveAppUsage_Validation(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Test with nil app usage data
	err := repo.SaveAppUsage(ctx, date, nil)
	if err == nil {
		t.Error("SaveAppUsage should fail with nil app usage data")
	}

	if !repoerrors.IsValidation(err) {
		t.Error("Expected validation error for nil app usage data")
	}
}

func TestSQLiteRepository_GetAppUsageByDateRange(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Create test data across multiple days
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	// Add data for each day
	for i := 0; i < 3; i++ {
		date := startDate.AddDate(0, 0, i)
		appUsage := &types.AppUsage{
			Name:     "TestApp",
			Duration: int64(1800 * (i + 1)), // Different durations
		}
		err := repo.SaveAppUsage(ctx, date, appUsage)
		if err != nil {
			t.Fatalf("Failed to save app usage for day %d: %v", i, err)
		}
	}

	// Test GetAppUsageByDateRange
	apps, err := repo.GetAppUsageByDateRange(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GetAppUsageByDateRange failed: %v", err)
	}

	if len(apps) != 3 {
		t.Errorf("Expected 3 apps, got %d", len(apps))
	}

	// Verify data is sorted and correct
	for i, app := range apps {
		expectedDuration := int64(1800 * (3 - i)) // Should be in descending order
		if app.Duration != expectedDuration {
			t.Errorf("App %d: expected duration %d, got %d", i, expectedDuration, app.Duration)
		}
	}
}
