package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"qwin/internal/types"
)

func TestSQLiteRepository_GetUsageHistory(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Create test data for the last 3 days using a fixed base date that's recent
	now := time.Now()
	baseDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := range 3 {
		date := baseDate.AddDate(0, 0, -i)

		// Save daily usage
		usage := &types.UsageData{
			TotalTime: int64(3600 * (i + 1)),
		}
		err := repo.SaveDailyUsage(ctx, date, usage)
		if err != nil {
			t.Fatalf("Failed to save daily usage for day %d: %v", i, err)
		}

		// Save app usage
		appUsage := &types.AppUsage{
			Name:     "HistoryApp",
			Duration: int64(1800 * (i + 1)),
		}
		err = repo.SaveAppUsage(ctx, date, appUsage)
		if err != nil {
			t.Fatalf("Failed to save app usage for day %d: %v", i, err)
		}
	}

	// Test GetUsageHistory
	history, err := repo.GetUsageHistory(ctx, 3)
	if err != nil {
		t.Fatalf("GetUsageHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 days of history, got %d", len(history))
	}

	// Verify each day has the correct data
	for dateKey, usageData := range history {
		if usageData.TotalTime <= 0 {
			t.Errorf("Day %s should have positive total time", dateKey)
		}

		if len(usageData.Apps) != 1 {
			t.Errorf("Day %s should have 1 app, got %d", dateKey, len(usageData.Apps))
		}

		if usageData.Apps[0].Name != "HistoryApp" {
			t.Errorf("Day %s should have HistoryApp, got %s", dateKey, usageData.Apps[0].Name)
		}
	}
}

func TestSQLiteRepository_GetUsageHistory_InvalidDays(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Test with invalid days parameter
	_, err := repo.GetUsageHistory(ctx, 0)
	if err == nil {
		t.Error("GetUsageHistory should fail with days = 0")
	}

	_, err = repo.GetUsageHistory(ctx, -1)
	if err == nil {
		t.Error("GetUsageHistory should fail with negative days")
	}
}

func TestSQLiteRepository_DeleteOldData(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Create old and new data using fixed dates
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	oldDate := baseDate.AddDate(0, 0, -60) // 60 days ago
	newDate := baseDate.AddDate(0, 0, -1)  // 1 day ago

	// Add old data
	oldUsage := &types.UsageData{TotalTime: 3600}
	err := repo.SaveDailyUsage(ctx, oldDate, oldUsage)
	if err != nil {
		t.Fatalf("Failed to save old daily usage: %v", err)
	}

	oldAppUsage := &types.AppUsage{Name: "OldApp", Duration: 1800}
	err = repo.SaveAppUsage(ctx, oldDate, oldAppUsage)
	if err != nil {
		t.Fatalf("Failed to save old app usage: %v", err)
	}

	// Add new data
	newUsage := &types.UsageData{TotalTime: 7200}
	err = repo.SaveDailyUsage(ctx, newDate, newUsage)
	if err != nil {
		t.Fatalf("Failed to save new daily usage: %v", err)
	}

	newAppUsage := &types.AppUsage{Name: "NewApp", Duration: 3600}
	err = repo.SaveAppUsage(ctx, newDate, newAppUsage)
	if err != nil {
		t.Fatalf("Failed to save new app usage: %v", err)
	}

	// Delete data older than 30 days
	cutoffDate := baseDate.AddDate(0, 0, -30)
	err = repo.DeleteOldData(ctx, cutoffDate)
	if err != nil {
		t.Fatalf("DeleteOldData failed: %v", err)
	}

	// Verify old data is gone
	_, err = repo.GetDailyUsage(ctx, oldDate)
	if err == nil {
		t.Error("Old daily usage should be deleted")
	}

	oldApps, err := repo.GetAppUsageByDate(ctx, oldDate)
	if err != nil {
		t.Fatalf("GetAppUsageByDate failed: %v", err)
	}
	if len(oldApps) != 0 {
		t.Error("Old app usage should be deleted")
	}

	// Verify new data still exists
	_, err = repo.GetDailyUsage(ctx, newDate)
	if err != nil {
		t.Errorf("New daily usage should still exist: %v", err)
	}

	newApps, err := repo.GetAppUsageByDate(ctx, newDate)
	if err != nil {
		t.Fatalf("GetAppUsageByDate failed: %v", err)
	}
	if len(newApps) != 1 {
		t.Error("New app usage should still exist")
	}
}

func TestSQLiteRepository_GetAppUsageByDateRangePaginated(t *testing.T) {
	t.Parallel()
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Create test data across multiple days
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Add 10 apps across 2 days (5 each day)
	for day := range 2 {
		date := startDate.AddDate(0, 0, day)
		for i := range 5 {
			appUsage := &types.AppUsage{
				Name:     fmt.Sprintf("PaginatedApp%d_%d", day, i),
				Duration: int64(1800 + i*300), // Different durations
			}
			err := repo.SaveAppUsage(ctx, date, appUsage)
			if err != nil {
				t.Fatalf("Failed to save app %d on day %d: %v", i, day, err)
			}
		}
	}

	endDate := startDate.AddDate(0, 0, 1)

	// Test pagination - first page
	result1, err := repo.GetAppUsageByDateRangePaginated(ctx, startDate, endDate, 5, 0)
	if err != nil {
		t.Fatalf("GetAppUsageByDateRangePaginated failed: %v", err)
	}

	if len(result1.Results) != 5 {
		t.Errorf("Expected 5 apps in first page, got %d", len(result1.Results))
	}

	if result1.Total != 10 {
		t.Errorf("Expected total count of 10, got %d", result1.Total)
	}

	// Verify first page is sorted correctly (date descending, then duration descending, then name ascending)
	apps1 := result1.Results
	for i := 1; i < len(apps1); i++ {
		prev := apps1[i-1]
		curr := apps1[i]

		// Compare by date first (descending)
		if prev.Date.After(curr.Date) {
			continue // correct order
		} else if prev.Date.Before(curr.Date) {
			t.Errorf("Page 1: Apps not sorted by date descending at index %d: %s (%s) should come before %s (%s)",
				i, prev.Name, prev.Date.Format("2006-01-02"), curr.Name, curr.Date.Format("2006-01-02"))
		} else {
			// Same date, compare by duration (descending)
			if prev.Duration > curr.Duration {
				continue // correct order
			} else if prev.Duration < curr.Duration {
				t.Errorf("Page 1: Apps not sorted by duration descending at index %d: %s (%d) should come before %s (%d)",
					i, prev.Name, prev.Duration, curr.Name, curr.Duration)
			} else {
				// Same duration, compare by name (ascending)
				if prev.Name <= curr.Name {
					continue // correct order
				} else {
					t.Errorf("Page 1: Apps not sorted by name ascending at index %d: %s should come before %s",
						i, prev.Name, curr.Name)
				}
			}
		}
	}

	// Test pagination - second page
	result2, err := repo.GetAppUsageByDateRangePaginated(ctx, startDate, endDate, 5, 5)
	if err != nil {
		t.Fatalf("GetAppUsageByDateRangePaginated failed: %v", err)
	}

	if len(result2.Results) != 5 {
		t.Errorf("Expected 5 apps in second page, got %d", len(result2.Results))
	}

	if result2.Total != 10 {
		t.Errorf("Expected total count of 10, got %d", result2.Total)
	}

	// Verify second page is sorted correctly
	apps2 := result2.Results
	for i := 1; i < len(apps2); i++ {
		prev := apps2[i-1]
		curr := apps2[i]

		// Compare by date first (descending)
		if prev.Date.After(curr.Date) {
			continue // correct order
		} else if prev.Date.Before(curr.Date) {
			t.Errorf("Page 2: Apps not sorted by date descending at index %d: %s (%s) should come before %s (%s)",
				i, prev.Name, prev.Date.Format("2006-01-02"), curr.Name, curr.Date.Format("2006-01-02"))
		} else {
			// Same date, compare by duration (descending)
			if prev.Duration > curr.Duration {
				continue // correct order
			} else if prev.Duration < curr.Duration {
				t.Errorf("Page 2: Apps not sorted by duration descending at index %d: %s (%d) should come before %s (%d)",
					i, prev.Name, prev.Duration, curr.Name, curr.Duration)
			} else {
				// Same duration, compare by name (ascending)
				if prev.Name <= curr.Name {
					continue // correct order
				} else {
					t.Errorf("Page 2: Apps not sorted by name ascending at index %d: %s should come before %s",
						i, prev.Name, curr.Name)
				}
			}
		}
	}

	// Verify cross-page ordering stability (last item of page 1 should not be "less" than first item of page 2)
	if len(apps1) > 0 && len(apps2) > 0 {
		lastPage1 := apps1[len(apps1)-1]
		firstPage2 := apps2[0]

		// Check if lastPage1 should come before firstPage2 (which would indicate wrong ordering)
		if lastPage1.Date.After(firstPage2.Date) ||
			(lastPage1.Date.Equal(firstPage2.Date) && lastPage1.Duration > firstPage2.Duration) ||
			(lastPage1.Date.Equal(firstPage2.Date) && lastPage1.Duration == firstPage2.Duration && lastPage1.Name < firstPage2.Name) {
			// This is correct - page 1 last item should come before page 2 first item
		} else if lastPage1.Date.Before(firstPage2.Date) ||
			(lastPage1.Date.Equal(firstPage2.Date) && lastPage1.Duration < firstPage2.Duration) ||
			(lastPage1.Date.Equal(firstPage2.Date) && lastPage1.Duration == firstPage2.Duration && lastPage1.Name > firstPage2.Name) {
			t.Errorf("Cross-page ordering violation: last item of page 1 (%s, %s, %d) should come before first item of page 2 (%s, %s, %d)",
				lastPage1.Name, lastPage1.Date.Format("2006-01-02"), lastPage1.Duration,
				firstPage2.Name, firstPage2.Date.Format("2006-01-02"), firstPage2.Duration)
		}
	}

	// Verify no overlap between pages
	for _, app1 := range result1.Results {
		for _, app2 := range result2.Results {
			if app1.Name == app2.Name {
				t.Errorf("Found duplicate app %s between pages", app1.Name)
			}
		}
	}

	// Test pagination beyond available data
	result3, err := repo.GetAppUsageByDateRangePaginated(ctx, startDate, endDate, 5, 15)
	if err != nil {
		t.Fatalf("GetAppUsageByDateRangePaginated failed: %v", err)
	}

	if len(result3.Results) != 0 {
		t.Errorf("Expected 0 apps beyond available data, got %d", len(result3.Results))
	}

	if result3.Total != 10 {
		t.Errorf("Expected total count of 10 even beyond data, got %d", result3.Total)
	}

	// Verify third page is empty but still properly ordered (no items to check)
	apps3 := result3.Results
	if len(apps3) != 0 {
		t.Errorf("Expected empty third page, got %d items", len(apps3))
	}
}
