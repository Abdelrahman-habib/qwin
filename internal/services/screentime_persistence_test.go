package services

import (
	"context"
	"testing"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

func TestScreenTimeTracker_DataPersistence(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Simulate some usage data
	tracker.mutex.Lock()
	tracker.usageData["TestApp1"] = 3600               // 1 hour
	tracker.usageData["TestApp2"] = 1800               // 30 minutes
	tracker.startTime = time.Now().Add(-2 * time.Hour) // Started 2 hours ago
	tracker.mutex.Unlock()

	// Test immediate persistence
	err := tracker.SaveCurrentDataNow()
	if err != nil {
		t.Errorf("SaveCurrentDataNow() unexpected error = %v", err)
	}

	// Verify repository was called
	save, _, batch, _, _, _ := mockRepo.GetCallCounts()
	if save == 0 {
		t.Error("SaveCurrentDataNow() did not call repository SaveDailyUsage")
	}
	if batch == 0 {
		t.Error("SaveCurrentDataNow() did not call repository BatchProcessAppUsage")
	}
}

func TestScreenTimeTracker_DataLoading(t *testing.T) {
	mockRepo := NewMockRepository()

	// Create tracker first to establish the current date
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Use tracker's current date to seed mock repository with test data
	testDate := tracker.CurrentDate()
	testUsage := &types.UsageData{
		TotalTime: 7200, // 2 hours
		Apps: []types.AppUsage{
			{Name: "LoadedApp1", Duration: 4800}, // 80 minutes
			{Name: "LoadedApp2", Duration: 2400}, // 40 minutes
		},
	}

	ctx := context.Background()
	mockRepo.SaveDailyUsage(ctx, testDate, testUsage)
	for i := range testUsage.Apps {
		mockRepo.SaveAppUsage(ctx, testDate, &testUsage.Apps[i])
	}

	// Now trigger data loading
	tracker.loadTodaysData()

	// Verify data was loaded
	tracker.mutex.RLock()
	if len(tracker.usageData) != 2 {
		t.Errorf("loadTodaysData() loaded %d apps, want 2", len(tracker.usageData))
	}

	if tracker.usageData["LoadedApp1"] != 4800 {
		t.Errorf("loadTodaysData() LoadedApp1 duration = %d, want 4800", tracker.usageData["LoadedApp1"])
	}

	if tracker.usageData["LoadedApp2"] != 2400 {
		t.Errorf("loadTodaysData() LoadedApp2 duration = %d, want 2400", tracker.usageData["LoadedApp2"])
	}
	tracker.mutex.RUnlock()

	// Verify repository was called
	_, load, _, _, _, _ := mockRepo.GetCallCounts()
	if load == 0 {
		t.Error("loadTodaysData() did not call repository methods")
	}
}

func TestScreenTimeTracker_DataLoadingWithErrors(t *testing.T) {
	mockRepo := NewMockRepository()
	mockRepo.SetFailureModes(false, true, false, false) // Fail on load

	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// This should not panic even with repository errors
	tracker.loadTodaysData()

	// Verify tracker is still functional
	tracker.mutex.RLock()
	usageDataLen := len(tracker.usageData)
	tracker.mutex.RUnlock()

	if usageDataLen != 0 {
		t.Errorf("loadTodaysData() with errors should result in empty usage data, got %d items", usageDataLen)
	}
}

func TestScreenTimeTracker_LoadDataForDate(t *testing.T) {
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

	// Test loading existing data
	loadedData, err := tracker.LoadDataForDate(testDate)
	if err != nil {
		t.Errorf("LoadDataForDate() unexpected error = %v", err)
	}

	if loadedData.TotalTime != 7200 {
		t.Errorf("LoadDataForDate() TotalTime = %d, want 7200", loadedData.TotalTime)
	}

	if len(loadedData.Apps) != 1 {
		t.Errorf("LoadDataForDate() returned %d apps, want 1", len(loadedData.Apps))
	}

	// Test loading non-existent data
	futureDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	emptyData, err := tracker.LoadDataForDate(futureDate)
	if err != nil {
		t.Errorf("LoadDataForDate() for non-existent date unexpected error = %v", err)
	}

	if emptyData.TotalTime != 0 {
		t.Errorf("LoadDataForDate() for non-existent date TotalTime = %d, want 0", emptyData.TotalTime)
	}
}
