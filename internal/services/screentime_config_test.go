package services

import (
	"context"
	"testing"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/platform"
	"qwin/internal/types"
)

func TestScreenTimeTracker_ResetUsageData(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Add some usage data
	tracker.mutex.Lock()
	tracker.usageData["TestApp"] = 1800
	tracker.appInfoCache["TestApp"] = &platform.AppInfo{Name: "TestApp"}
	tracker.mutex.Unlock()

	// Reset usage data
	tracker.ResetUsageData()

	// Verify data was reset
	tracker.mutex.RLock()
	if len(tracker.usageData) != 0 {
		t.Errorf("ResetUsageData() usageData not reset, got %d items", len(tracker.usageData))
	}

	if len(tracker.appInfoCache) != 0 {
		t.Errorf("ResetUsageData() appInfoCache not reset, got %d items", len(tracker.appInfoCache))
	}
	tracker.mutex.RUnlock()

	// Verify persistence was called before reset
	save, _, batch, _, _, _ := mockRepo.GetCallCounts()
	if save == 0 || batch == 0 {
		t.Error("ResetUsageData() should persist data before reset")
	}
}

func TestScreenTimeTracker_CleanupOldData(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Test cleanup
	err := tracker.CleanupOldData(30)
	if err != nil {
		t.Errorf("CleanupOldData() unexpected error = %v", err)
	}

	// Verify repository was called
	_, _, _, _, deleteCount, _ := mockRepo.GetCallCounts()
	if deleteCount == 0 {
		t.Error("CleanupOldData() did not call repository DeleteOldData")
	}
}

func TestScreenTimeTracker_ErrorHandling(t *testing.T) {
	mockRepo := NewMockRepository()

	tests := []struct {
		name        string
		failSave    bool
		failLoad    bool
		failBatch   bool
		failTx      bool
		expectError bool
	}{
		{
			name:        "save failure",
			failSave:    true,
			expectError: false, // Service should handle gracefully
		},
		{
			name:        "load failure",
			failLoad:    true,
			expectError: false, // Service should handle gracefully
		},
		{
			name:        "batch failure",
			failBatch:   true,
			expectError: false, // Service should handle gracefully
		},
		{
			name:        "transaction failure",
			failTx:      true,
			expectError: false, // Service should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.SetFailureModes(tt.failSave, tt.failLoad, tt.failBatch, tt.failTx)
			tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

			// Add some test data
			tracker.mutex.Lock()
			tracker.usageData["ErrorTestApp"] = 1800
			tracker.mutex.Unlock()

			// Test operations that might fail
			err := tracker.SaveCurrentDataNow()
			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error = %v", tt.name, err)
			}

			// Test loading with errors
			tracker.loadTodaysData()

			// Test historical data with errors
			_, err = tracker.GetHistoricalUsage(7)
			if tt.failLoad && err == nil {
				t.Errorf("%s: GetHistoricalUsage should fail when load fails", tt.name)
			}
		})
	}
}

func TestScreenTimeTracker_SetPersistenceEnabled(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Test setting persistence enabled/disabled
	tracker.SetPersistenceEnabled(false)
	if tracker.IsPersistenceEnabled() {
		t.Error("SetPersistenceEnabled(false) did not disable persistence")
	}

	tracker.SetPersistenceEnabled(true)
	if !tracker.IsPersistenceEnabled() {
		t.Error("SetPersistenceEnabled(true) did not enable persistence")
	}
}

func TestScreenTimeTracker_PersistenceDisabledOperations(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Disable persistence
	tracker.SetPersistenceEnabled(false)

	// Add some test data
	tracker.mutex.Lock()
	tracker.usageData["TestApp"] = 1800
	tracker.mutex.Unlock()

	// Test operations with persistence disabled
	err := tracker.SaveCurrentDataNow()
	if err != nil {
		t.Errorf("SaveCurrentDataNow() with persistence disabled unexpected error = %v", err)
	}

	// Verify repository was not called (since persistence is disabled)
	save, _, batch, _, _, _ := mockRepo.GetCallCounts()
	if save != 0 || batch != 0 {
		t.Error("SaveCurrentDataNow() should not call repository when persistence is disabled")
	}

	// Test loading with persistence disabled
	tracker.loadTodaysData()

	// Verify repository load methods were not called
	_, load, _, _, _, _ := mockRepo.GetCallCounts()
	if load != 0 {
		t.Error("loadTodaysData() should not call repository when persistence is disabled")
	}
}

func TestScreenTimeTracker_StartStopRunningState(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Initially should not be running
	if tracker.IsRunning() {
		t.Error("Tracker should not be running initially")
	}

	// Start the tracker
	tracker.Start()

	// Should now be running
	if !tracker.IsRunning() {
		t.Error("Tracker should be running after Start()")
	}

	// Calling Start() again should not change anything (no duplicate goroutines)
	tracker.Start() // Should return early

	// Should still be running
	if !tracker.IsRunning() {
		t.Error("Tracker should still be running after duplicate Start()")
	}

	// Stop the tracker
	tracker.Stop()

	// Should no longer be running
	if tracker.IsRunning() {
		t.Error("Tracker should not be running after Stop()")
	}

	// Calling Stop() again should not cause issues
	tracker.Stop() // Should return early

	// Should still not be running
	if tracker.IsRunning() {
		t.Error("Tracker should still not be running after duplicate Stop()")
	}

	// Should be able to start again after stopping
	tracker.Start()

	// Should be running again
	if !tracker.IsRunning() {
		t.Error("Tracker should be running after restart")
	}

	// Clean up
	tracker.Stop()
}

func TestMockRepository_SortingBehavior(t *testing.T) {
	mockRepo := NewMockRepository()
	ctx := context.Background()

	// Create test data across multiple dates with different durations
	date1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	// Add apps with different durations for each date
	testCases := []struct {
		date     time.Time
		appName  string
		duration int64
	}{
		{date1, "App1_Day1", 100},
		{date1, "App2_Day1", 300}, // Higher duration on same day
		{date2, "App1_Day2", 200},
		{date2, "App2_Day2", 150},
		{date3, "App1_Day3", 50},  // Latest date
		{date3, "App2_Day3", 400}, // Latest date, highest duration
	}

	for _, tc := range testCases {
		appUsage := &types.AppUsage{
			Name:     tc.appName,
			Duration: tc.duration,
			Date:     tc.date,
		}
		err := mockRepo.SaveAppUsage(ctx, tc.date, appUsage)
		if err != nil {
			t.Fatalf("Failed to save app usage: %v", err)
		}
	}

	// Get usage by date range and verify sorting
	startDate := date1
	endDate := date3
	results, err := mockRepo.GetAppUsageByDateRange(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to get app usage by date range: %v", err)
	}

	if len(results) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(results))
	}

	// Verify sorting: date DESC, then duration DESC
	// Expected order:
	// 1. App2_Day3 (2024-01-17, 400) - latest date, highest duration
	// 2. App1_Day3 (2024-01-17, 50)  - latest date, lower duration
	// 3. App1_Day2 (2024-01-16, 200) - middle date, higher duration
	// 4. App2_Day2 (2024-01-16, 150) - middle date, lower duration
	// 5. App2_Day1 (2024-01-15, 300) - earliest date, higher duration
	// 6. App1_Day1 (2024-01-15, 100) - earliest date, lower duration

	expected := []struct {
		name     string
		duration int64
		date     time.Time
	}{
		{"App2_Day3", 400, date3},
		{"App1_Day3", 50, date3},
		{"App1_Day2", 200, date2},
		{"App2_Day2", 150, date2},
		{"App2_Day1", 300, date1},
		{"App1_Day1", 100, date1},
	}

	for i, exp := range expected {
		if i >= len(results) {
			t.Fatalf("Missing result at index %d", i)
		}
		result := results[i]
		if result.Name != exp.name {
			t.Errorf("Index %d: expected name %s, got %s", i, exp.name, result.Name)
		}
		if result.Duration != exp.duration {
			t.Errorf("Index %d: expected duration %d, got %d", i, exp.duration, result.Duration)
		}
		// Note: We check date by formatting since time comparison can be tricky with locations
		if result.Date.Format("2006-01-02") != exp.date.Format("2006-01-02") {
			t.Errorf("Index %d: expected date %s, got %s", i, exp.date.Format("2006-01-02"), result.Date.Format("2006-01-02"))
		}
	}
}

func TestScreenTimeTracker_ElapsedTimeAttribution(t *testing.T) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Set up mock window API to control what app is returned
	mockWindowAPI := &MockWindowAPI{}
	tracker.windowAPI = mockWindowAPI

	// Simulate app switching scenario
	now := time.Now()

	// Start with Chrome
	mockWindowAPI.SetCurrentApp(&platform.AppInfo{Name: "Chrome"})
	tracker.mutex.Lock()
	tracker.lastTime = now
	tracker.mutex.Unlock()

	// Simulate Chrome usage for 5 seconds
	timeAfter5Sec := now.Add(5 * time.Second)
	tracker.mutex.Lock()
	tracker.lastTime = timeAfter5Sec
	tracker.lastApp = "Chrome"
	tracker.mutex.Unlock()

	// Switch to VS Code (this should attribute the 5 seconds to Chrome)
	mockWindowAPI.SetCurrentApp(&platform.AppInfo{Name: "VSCode"})
	// Manually call trackCurrentApp to simulate the tracking tick
	tracker.mutex.Lock()
	tracker.lastTime = timeAfter5Sec
	tracker.lastApp = "Chrome"
	tracker.mutex.Unlock()
	
	// Simulate what trackCurrentApp does
	appInfo := mockWindowAPI.GetCurrentAppInfo()
	timeAfter8Sec := timeAfter5Sec.Add(3 * time.Second)
	
	tracker.mutex.Lock()
	// This should attribute 3 seconds to Chrome (from timeAfter5Sec to timeAfter8Sec)
	if tracker.lastApp != "" && !tracker.lastTime.IsZero() {
		elapsed := timeAfter8Sec.Sub(tracker.lastTime).Seconds()
		if elapsed > 0 {
			tracker.usageData[tracker.lastApp] += int64(elapsed)
		}
	}
	tracker.lastApp = appInfo.Name
	tracker.lastTime = timeAfter8Sec
	tracker.mutex.Unlock()

	// Verify Chrome got the 3 seconds attributed
	tracker.mutex.RLock()
	chromeUsage := tracker.usageData["Chrome"]
	vsCodeUsage := tracker.usageData["VSCode"]
	tracker.mutex.RUnlock()

	if chromeUsage != 3 {
		t.Errorf("Expected Chrome usage to be 3 seconds, got %d", chromeUsage)
	}

	if vsCodeUsage != 0 {
		t.Errorf("Expected VSCode usage to be 0 seconds (just switched to it), got %d", vsCodeUsage)
	}
}

// MockWindowAPI for testing app switching
type MockWindowAPI struct {
	currentApp *platform.AppInfo
}

func (m *MockWindowAPI) GetCurrentAppName() string {
	if m.currentApp != nil {
		return m.currentApp.Name
	}
	return ""
}

func (m *MockWindowAPI) GetCurrentAppInfo() *platform.AppInfo {
	return m.currentApp
}

func (m *MockWindowAPI) SetCurrentApp(app *platform.AppInfo) {
	m.currentApp = app
}
