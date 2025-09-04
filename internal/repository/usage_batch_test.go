package repository

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/types"
)

func TestSQLiteRepository_BatchProcessAppUsage(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	appUsages := []types.AppUsage{
		{Name: "BatchApp1", Duration: 1800},
		{Name: "BatchApp2", Duration: 2400},
		{Name: "BatchApp3", Duration: 3600},
	}

	// Test batch upsert
	err := repo.BatchProcessAppUsage(ctx, date, appUsages, types.BatchStrategyUpsert)
	if err != nil {
		t.Fatalf("BatchProcessAppUsage failed: %v", err)
	}

	// Verify all apps were saved
	retrievedApps, err := repo.GetAppUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after batch save: %v", err)
	}

	if len(retrievedApps) != 3 {
		t.Errorf("Expected 3 apps, got %d", len(retrievedApps))
	}

	// Test batch insert-only strategy with duplicate (should fail)
	duplicateAppUsages := []types.AppUsage{
		{Name: "BatchApp1", Duration: 9999}, // Should fail on duplicate
	}

	err = repo.BatchProcessAppUsage(ctx, date, duplicateAppUsages, types.BatchStrategyInsertOnly)
	if err == nil {
		t.Error("BatchProcessAppUsage with insert-only should fail on duplicate")
	}

	// Test batch insert-only strategy with new app (should succeed)
	newAppUsages := []types.AppUsage{
		{Name: "BatchApp4", Duration: 1200}, // Should insert new
	}

	err = repo.BatchProcessAppUsage(ctx, date, newAppUsages, types.BatchStrategyInsertOnly)
	if err != nil {
		t.Fatalf("BatchProcessAppUsage with insert-only failed on new app: %v", err)
	}

	// Verify the new app was added but existing wasn't changed
	retrievedApps2, err := repo.GetAppUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after insert-only: %v", err)
	}

	if len(retrievedApps2) != 4 {
		t.Errorf("Expected 4 apps after insert-only, got %d", len(retrievedApps2))
	}

	// Test with empty slice (should not error)
	err = repo.BatchProcessAppUsage(ctx, date, []types.AppUsage{}, types.BatchStrategyUpsert)
	if err != nil {
		t.Errorf("BatchProcessAppUsage should handle empty slice: %v", err)
	}
}

func TestSQLiteRepository_BatchProcessAppUsageWithBatchSize(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	// Create a larger dataset to test custom batch sizing
	var appUsages []types.AppUsage
	for i := 0; i < 10; i++ {
		appUsages = append(appUsages, types.AppUsage{
			Name:     fmt.Sprintf("CustomBatchApp%d", i),
			Duration: int64(1800 + i*300),
		})
	}

	// Test with custom batch size
	customBatchSize := 3
	err := repo.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, types.BatchStrategyUpsert, customBatchSize)
	if err != nil {
		t.Fatalf("BatchProcessAppUsageWithBatchSize failed: %v", err)
	}

	// Verify all apps were saved
	retrievedApps, err := repo.GetAppUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after batch save: %v", err)
	}

	if len(retrievedApps) != 10 {
		t.Errorf("Expected 10 apps, got %d", len(retrievedApps))
	}

	// Test with batchSize = 0 (should use strategy-based calculation)
	date2 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	err = repo.BatchProcessAppUsageWithBatchSize(ctx, date2, appUsages, types.BatchStrategyInsertOnly, 0)
	if err != nil {
		t.Fatalf("BatchProcessAppUsageWithBatchSize with auto-sizing failed: %v", err)
	}

	// Verify apps were saved with auto batch sizing
	retrievedApps2, err := repo.GetAppUsageByDate(ctx, date2)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after auto batch save: %v", err)
	}

	if len(retrievedApps2) != 10 {
		t.Errorf("Expected 10 apps with auto batch sizing, got %d", len(retrievedApps2))
	}
}

func TestSQLiteRepository_BatchProcessAppUsageWithBatchSize_Validation(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	appUsages := []types.AppUsage{
		{Name: "TestApp", Duration: 1800},
	}

	// Test with negative batch size (should fail)
	err := repo.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, types.BatchStrategyUpsert, -1)
	if err == nil {
		t.Error("BatchProcessAppUsageWithBatchSize should fail with negative batch size")
	}

	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for negative batch size, got: %v", err)
	}

	// Test with zero batch size (should succeed - uses default calculation)
	err = repo.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, types.BatchStrategyUpsert, 0)
	if err != nil {
		t.Errorf("BatchProcessAppUsageWithBatchSize should succeed with zero batch size: %v", err)
	}

	// Test with positive batch size (should succeed)
	err = repo.BatchProcessAppUsageWithBatchSize(ctx, date, appUsages, types.BatchStrategyUpsert, 1)
	if err != nil {
		t.Errorf("BatchProcessAppUsageWithBatchSize should succeed with positive batch size: %v", err)
	}
}

func TestSQLiteRepository_BatchIncrementAppUsageDurations(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// First save some apps
	appUsages := []types.AppUsage{
		{Name: "UpdateApp1", Duration: 1800},
		{Name: "UpdateApp2", Duration: 2400},
	}

	err := repo.BatchProcessAppUsage(ctx, date, appUsages, types.BatchStrategyUpsert)
	if err != nil {
		t.Fatalf("Failed to save initial apps: %v", err)
	}

	// Increment durations
	increments := map[string]int64{
		"UpdateApp1": 600,  // Add 10 minutes
		"UpdateApp2": 1200, // Add 20 minutes
	}

	err = repo.BatchIncrementAppUsageDurations(ctx, date, increments)
	if err != nil {
		t.Fatalf("BatchIncrementAppUsageDurations failed: %v", err)
	}

	// Verify updates
	retrievedApps, err := repo.GetAppUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after update: %v", err)
	}

	for _, app := range retrievedApps {
		switch app.Name {
		case "UpdateApp1":
			if app.Duration != 2400 { // 1800 + 600
				t.Errorf("UpdateApp1 duration should be 2400, got %d", app.Duration)
			}
		case "UpdateApp2":
			if app.Duration != 3600 { // 2400 + 1200
				t.Errorf("UpdateApp2 duration should be 3600, got %d", app.Duration)
			}
		}
	}

	// Test with empty increments (should not error)
	err = repo.BatchIncrementAppUsageDurations(ctx, date, map[string]int64{})
	if err != nil {
		t.Errorf("BatchIncrementAppUsageDurations should handle empty map: %v", err)
	}

	// Test validation: negative increment should fail
	negativeIncrements := map[string]int64{
		"UpdateApp1": -100,
	}
	err = repo.BatchIncrementAppUsageDurations(ctx, date, negativeIncrements)
	if err == nil {
		t.Error("BatchIncrementAppUsageDurations should reject negative increments")
	}
	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for negative increment, got: %v", err)
	}

	// Test validation: overflow protection
	// First create an app with near-max duration
	overflowApp := types.AppUsage{
		Name:     "OverflowApp",
		Duration: math.MaxInt64 - 100, // Very close to max
	}
	err = repo.SaveAppUsage(ctx, date, &overflowApp)
	if err != nil {
		t.Fatalf("Failed to save overflow test app: %v", err)
	}

	// Try to increment beyond max
	overflowIncrements := map[string]int64{
		"OverflowApp": 200, // This would cause overflow
	}
	err = repo.BatchIncrementAppUsageDurations(ctx, date, overflowIncrements)
	if err == nil {
		t.Error("BatchIncrementAppUsageDurations should prevent integer overflow")
	}
	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for overflow, got: %v", err)
	}

	// Test insert-on-missing behavior: increment non-existent app should create it
	date2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	missingAppIncrements := map[string]int64{
		"NewApp": 1800, // This app doesn't exist, should be created
	}
	err = repo.BatchIncrementAppUsageDurations(ctx, date2, missingAppIncrements)
	if err != nil {
		t.Fatalf("BatchIncrementAppUsageDurations should create missing app: %v", err)
	}

	// Verify the new app was created
	retrievedApps2, err := repo.GetAppUsageByDate(ctx, date2)
	if err != nil {
		t.Fatalf("Failed to retrieve apps after insert-on-missing: %v", err)
	}

	if len(retrievedApps2) != 1 {
		t.Errorf("Expected 1 app after insert-on-missing, got %d", len(retrievedApps2))
	}

	if retrievedApps2[0].Name != "NewApp" {
		t.Errorf("Expected app name 'NewApp', got %s", retrievedApps2[0].Name)
	}

	if retrievedApps2[0].Duration != 1800 {
		t.Errorf("Expected app duration 1800, got %d", retrievedApps2[0].Duration)
	}
}
