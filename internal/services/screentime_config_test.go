package services

import (
	"testing"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/platform"
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
