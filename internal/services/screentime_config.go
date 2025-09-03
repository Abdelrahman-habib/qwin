package services

import (
	"context"
	"time"

	"qwin/internal/infrastructure/errors"
	"qwin/internal/platform"
)

// ResetUsageData resets the usage data
func (st *ScreenTimeTracker) ResetUsageData() {
	// Persist current data before reset
	st.persistCurrentData()

	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.usageData = make(map[string]int64)
	st.appInfoCache = make(map[string]*platform.AppInfo)
	st.startTime = time.Now()
	st.lastTime = time.Time{}
	st.lastApp = ""

	// Update current date
	now := time.Now()
	st.currentDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// CleanupOldData removes usage data older than the specified number of days
func (st *ScreenTimeTracker) CleanupOldData(retentionDays int) error {
	if st.repository == nil {
		return errors.NewRepositoryError("CleanupOldData", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	return st.repository.DeleteOldData(ctx, cutoffDate)
}

// SetPersistenceEnabled enables or disables data persistence
func (st *ScreenTimeTracker) SetPersistenceEnabled(enabled bool) {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	st.persistenceEnabled = enabled
}

// IsPersistenceEnabled returns whether data persistence is enabled
func (st *ScreenTimeTracker) IsPersistenceEnabled() bool {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	return st.persistenceEnabled
}
