package repository

import (
	"context"
	"testing"
	"time"

	"qwin/internal/types"
)

func TestSQLiteRepository_WithTransaction(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	usage := &types.UsageData{TotalTime: 3600}
	appUsage := &types.AppUsage{Name: "TransactionApp", Duration: 1800}

	// Test successful transaction
	err := repo.WithTransaction(ctx, func(txRepo UsageRepository) error {
		if err := txRepo.SaveDailyUsage(ctx, date, usage); err != nil {
			return err
		}
		return txRepo.SaveAppUsage(ctx, date, appUsage)
	})

	if err != nil {
		t.Fatalf("Transaction should succeed: %v", err)
	}

	// Verify data was saved
	retrievedUsage, err := repo.GetDailyUsage(ctx, date)
	if err != nil {
		t.Fatalf("Failed to retrieve usage after transaction: %v", err)
	}

	if retrievedUsage.TotalTime != usage.TotalTime {
		t.Error("Transaction data was not saved correctly")
	}

	// Test transaction rollback on error
	date2 := date.AddDate(0, 0, 1)
	err = repo.WithTransaction(ctx, func(txRepo UsageRepository) error {
		if err := txRepo.SaveDailyUsage(ctx, date2, usage); err != nil {
			return err
		}
		// Force an error by passing nil
		return txRepo.SaveAppUsage(ctx, date2, nil)
	})

	if err == nil {
		t.Error("Transaction should fail due to validation error")
	}

	// Verify rollback - data should not exist
	_, err = repo.GetDailyUsage(ctx, date2)
	if err == nil {
		t.Error("Transaction should have been rolled back")
	}
}
