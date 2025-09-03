package repository

import (
	"context"
	"testing"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

func TestSQLiteRepository_SaveAndGetDailyUsage(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Test data
	date := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
	usage := &types.UsageData{
		TotalTime: 7200, // 2 hours
		Apps: []types.AppUsage{
			{Name: "TestApp1", Duration: 3600},
			{Name: "TestApp2", Duration: 3600},
		},
	}

	// Test SaveDailyUsage
	err := repo.SaveDailyUsage(ctx, date, usage)
	if err != nil {
		t.Fatalf("SaveDailyUsage failed: %v", err)
	}

	// Test GetDailyUsage
	retrieved, err := repo.GetDailyUsage(ctx, date)
	if err != nil {
		t.Fatalf("GetDailyUsage failed: %v", err)
	}

	if retrieved.TotalTime != usage.TotalTime {
		t.Errorf("Expected TotalTime %d, got %d", usage.TotalTime, retrieved.TotalTime)
	}

	// Test date normalization (should work with different times on same date)
	dateWithDifferentTime := time.Date(2024, 1, 15, 18, 45, 30, 0, time.UTC)
	retrieved2, err := repo.GetDailyUsage(ctx, dateWithDifferentTime)
	if err != nil {
		t.Fatalf("GetDailyUsage with different time failed: %v", err)
	}

	if retrieved2.TotalTime != usage.TotalTime {
		t.Error("Date normalization failed - should retrieve same data regardless of time")
	}
}

func TestSQLiteRepository_SaveDailyUsage_Validation(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Test with nil usage data
	err := repo.SaveDailyUsage(ctx, date, nil)
	if err == nil {
		t.Error("SaveDailyUsage should fail with nil usage data")
	}

	if !repoerrors.IsValidation(err) {
		t.Error("Expected validation error for nil usage data")
	}
}

func TestSQLiteRepository_GetDailyUsage_NotFound(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Try to get data for a date that doesn't exist
	futureDate := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.GetDailyUsage(ctx, futureDate)
	if err == nil {
		t.Error("GetDailyUsage should return error for non-existent date")
	}

	if !repoerrors.IsNotFound(err) {
		t.Errorf("Expected NotFound error, got: %v", err)
	}
}
