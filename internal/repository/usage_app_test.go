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

	// Test with empty app name - should embed context in error
	empptyNameApp := &types.AppUsage{
		Name:     " ",  // Empty/whitespace name
		Duration: 1800,
	}
	err = repo.SaveAppUsage(ctx, date, empptyNameApp)
	if err == nil {
		t.Error("SaveAppUsage should fail with empty app name")
	}

	if !repoerrors.IsValidation(err) {
		t.Error("Expected validation error for empty app name")
	}

	// Verify context is embedded in the error
	if repoErr, ok := err.(*repoerrors.RepositoryError); ok {
		if repoErr.Context == nil {
			t.Error("Expected error context to be embedded")
		} else {
			if repoErr.Context["app_name"] != " " {
				t.Errorf("Expected app_name in context to be ' ', got %v", repoErr.Context["app_name"])
			}
			if repoErr.Context["duration"] != "1800" {
				t.Errorf("Expected duration in context to be '1800', got %v", repoErr.Context["duration"])
			}
			if repoErr.Context["date"] != "2024-01-15" {
				t.Errorf("Expected date in context to be '2024-01-15', got %v", repoErr.Context["date"])
			}
		}
	} else {
		t.Error("Expected error to be a RepositoryError")
	}

	// Test with negative duration - should embed context in error
	negativeDurationApp := &types.AppUsage{
		Name:     "TestApp",
		Duration: -500, // Negative duration
	}
	err = repo.SaveAppUsage(ctx, date, negativeDurationApp)
	if err == nil {
		t.Error("SaveAppUsage should fail with negative duration")
	}

	if !repoerrors.IsValidation(err) {
		t.Error("Expected validation error for negative duration")
	}

	// Verify context is embedded in the error
	if repoErr, ok := err.(*repoerrors.RepositoryError); ok {
		if repoErr.Context == nil {
			t.Error("Expected error context to be embedded")
		} else {
			if repoErr.Context["app_name"] != "TestApp" {
				t.Errorf("Expected app_name in context to be 'TestApp', got %v", repoErr.Context["app_name"])
			}
			if repoErr.Context["duration"] != "-500" {
				t.Errorf("Expected duration in context to be '-500', got %v", repoErr.Context["duration"])
			}
			if repoErr.Context["date"] != "2024-01-15" {
				t.Errorf("Expected date in context to be '2024-01-15', got %v", repoErr.Context["date"])
			}
		}
	} else {
		t.Error("Expected error to be a RepositoryError")
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
