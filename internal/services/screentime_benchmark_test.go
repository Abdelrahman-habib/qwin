package services

import (
	"fmt"
	"testing"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/types"
)

// Benchmark tests for performance validation
func BenchmarkScreenTimeTracker_GetUsageData(b *testing.B) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Add test data
	tracker.mutex.Lock()
	for i := 0; i < 100; i++ {
		tracker.usageData[fmt.Sprintf("BenchApp%d", i)] = int64(i * 60)
	}
	tracker.mutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.GetUsageData()
	}
}

func BenchmarkScreenTimeTracker_SaveCurrentDataNow(b *testing.B) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Add test data
	tracker.mutex.Lock()
	for i := 0; i < 50; i++ {
		tracker.usageData[fmt.Sprintf("BenchApp%d", i)] = int64(i * 60)
	}
	tracker.mutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.SaveCurrentDataNow()
	}
}

func BenchmarkScreenTimeTracker_ConcurrentGetUsageData(b *testing.B) {
	mockRepo := NewMockRepository()
	tracker := NewScreenTimeTracker(mockRepo, logging.NewDefaultLogger())

	// Add test data
	tracker.mutex.Lock()
	for i := 0; i < 100; i++ {
		tracker.usageData[fmt.Sprintf("BenchApp%d", i)] = int64(i * 60)
	}
	tracker.mutex.Unlock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = tracker.GetUsageData()
		}
	})
}

func BenchmarkMockRepository_SaveDailyUsage(b *testing.B) {
	mockRepo := NewMockRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark the mock repository performance
		_ = mockRepo.SaveDailyUsage(nil, mockRepo.getDummyDate(i), mockRepo.getDummyUsageData(i))
	}
}

// Helper methods for benchmark tests
func (m *MockRepository) getDummyDate(i int) time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i%365)
}

func (m *MockRepository) getDummyUsageData(i int) *types.UsageData {
	return &types.UsageData{
		TotalTime: int64(i * 60),
		Apps:      []types.AppUsage{},
	}
}
