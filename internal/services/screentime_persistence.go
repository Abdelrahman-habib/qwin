package services

import (
	"context"
	"log"
	"math"
	"time"

	"qwin/internal/infrastructure/errors"
	"qwin/internal/platform"
	"qwin/internal/repository"
	"qwin/internal/types"
)

// startPersistenceLoop starts the periodic data persistence (every 30 seconds)
func (st *ScreenTimeTracker) startPersistenceLoop() {
	ticker := time.NewTicker(30 * time.Second)

	// Assign ticker to struct field and capture stop channel under mutex protection
	st.mutex.Lock()
	st.persistTicker = ticker
	stopCh := st.stopTracking
	st.mutex.Unlock()

	go func() {
		for {
			select {
			case <-ticker.C:
				st.persistCurrentData()
			case <-stopCh:
				// Stop the ticker to prevent timer leak
				ticker.Stop()
				return
			}
		}
	}()
}

// persistCurrentData saves current usage data to the database
func (st *ScreenTimeTracker) persistCurrentData() {
	if st.repository == nil || !st.persistenceEnabled {
		return
	}

	ctx := context.Background()

	// Snapshot state under lock to minimize lock contention
	st.mutex.Lock()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !today.Equal(st.currentDate) {
		// Date has changed, persist old data and reset for new day
		oldDate := st.currentDate
		oldStartTime := st.startTime
		oldUsageData := make(map[string]int64, len(st.usageData))
		for k, v := range st.usageData {
			oldUsageData[k] = v
		}
		oldAppInfoCache := make(map[string]*platform.AppInfo, len(st.appInfoCache))
		for k, v := range st.appInfoCache {
			oldAppInfoCache[k] = v
		}

		// Update state for new day
		st.currentDate = today
		st.usageData = make(map[string]int64)
		st.startTime = now
		st.mutex.Unlock()

		// Persist old data outside the lock
		st.persistDataForDateWithSnapshot(ctx, oldDate, oldStartTime, oldUsageData, oldAppInfoCache, now)
		return
	}

	// Snapshot current day's data
	currentDate := st.currentDate
	startTime := st.startTime
	usageDataCopy := make(map[string]int64, len(st.usageData))
	for k, v := range st.usageData {
		usageDataCopy[k] = v
	}
	appInfoCacheCopy := make(map[string]*platform.AppInfo, len(st.appInfoCache))
	for k, v := range st.appInfoCache {
		appInfoCacheCopy[k] = v
	}
	st.lastPersist = now
	st.mutex.Unlock()

	// Persist current day's data outside the lock
	st.persistDataForDateWithSnapshot(ctx, currentDate, startTime, usageDataCopy, appInfoCacheCopy, now)
}

// persistDataForDateWithSnapshot saves usage data for a specific date using provided snapshot data
// This function does not access st.mutex and can be called without holding locks
func (st *ScreenTimeTracker) persistDataForDateWithSnapshot(
	ctx context.Context,
	date time.Time,
	startTime time.Time,
	usageData map[string]int64,
	appInfoCache map[string]*platform.AppInfo,
	asOfTime time.Time,
) {
	// Calculate total time bounded to the target date using DST-safe calculation
	var totalTime int64
	if !startTime.IsZero() {
		// Calculate start and end boundaries of the target date
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		nextMidnight := startOfDay.AddDate(0, 0, 1)

		// Clamp the time interval to the target date boundaries
		start := startTime
		if start.Before(startOfDay) {
			start = startOfDay
		}
		if start.After(nextMidnight) {
			start = nextMidnight
		}

		end := asOfTime
		if end.After(nextMidnight) {
			end = nextMidnight
		}
		if end.Before(start) {
			end = start
		}

		// Calculate elapsed time only within the target date boundaries using math.Round
		if end.After(start) {
			totalTime = int64(math.Round(end.Sub(start).Seconds()))
		}
	}

	// Create usage data summary
	usageDataSummary := &types.UsageData{
		TotalTime: totalTime,
	}

	// Prepare app usage data for batch save
	appUsages := make([]types.AppUsage, 0, len(usageData))
	for name, duration := range usageData {
		appUsage := types.AppUsage{
			Name:     name,
			Duration: duration,
			Date:     date,
		}

		// Add cached app info if available
		if cachedInfo, exists := appInfoCache[name]; exists {
			appUsage.IconPath = cachedInfo.IconPath
			appUsage.ExePath = cachedInfo.ExePath
		}

		appUsages = append(appUsages, appUsage)
	}

	// Add timeout to avoid hanging on shutdown
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Wrap both operations in a transaction for atomicity
	if err := st.repository.WithTransaction(ctx, func(txRepo repository.UsageRepository) error {
		// Save daily usage summary
		if err := txRepo.SaveDailyUsage(ctx, date, usageDataSummary); err != nil {
			return err
		}

		// Batch save app usage data
		if len(appUsages) > 0 {
			if err := txRepo.BatchProcessAppUsage(ctx, date, appUsages, types.BatchStrategyUpsert); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		st.logger.Error("Failed to persist usage snapshot", "date", date, "error", err)
	}
}

// loadTodaysData loads existing usage data for today from the database
func (st *ScreenTimeTracker) loadTodaysData() {
	if st.repository == nil || !st.persistenceEnabled {
		return
	}

	ctx := context.Background()

	// Load daily usage data
	dailyUsage, err := st.repository.GetDailyUsage(ctx, st.currentDate)
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to load daily usage data: %v", err)
		return
	}

	// Load app usage data
	appUsages, err := st.repository.GetAppUsageByDate(ctx, st.currentDate)
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to load app usage data: %v", err)
		return
	}

	st.mutex.Lock()
	defer st.mutex.Unlock()

	// Restore usage data if found
	if dailyUsage != nil {
		// Adjust start time to account for previously tracked time
		// If we've already tracked dailyUsage.TotalTime seconds today,
		// set startTime so that time.Since(startTime) equals that amount
		now := time.Now()
		st.startTime = now.Add(-time.Duration(dailyUsage.TotalTime) * time.Second)
	}

	// Restore app usage data
	for _, appUsage := range appUsages {
		st.usageData[appUsage.Name] = appUsage.Duration

		// Cache app info
		if appUsage.IconPath != "" || appUsage.ExePath != "" {
			st.appInfoCache[appUsage.Name] = &platform.AppInfo{
				Name:     appUsage.Name,
				IconPath: appUsage.IconPath,
				ExePath:  appUsage.ExePath,
			}
		}
	}

	st.logger.Info("Loaded usage data for applications", "count", len(appUsages))
}

// SaveCurrentDataNow immediately persists current usage data to the database
func (st *ScreenTimeTracker) SaveCurrentDataNow() error {
	if st.repository == nil {
		return errors.NewRepositoryError("SaveCurrentDataNow", nil, errors.ErrCodeConnection)
	}

	st.persistCurrentData()
	return nil
}

// LoadDataForDate loads usage data for a specific date (useful for data recovery)
func (st *ScreenTimeTracker) LoadDataForDate(date time.Time) (*types.UsageData, error) {
	if st.repository == nil {
		return nil, errors.NewRepositoryError("LoadDataForDate", nil, errors.ErrCodeConnection)
	}

	ctx := context.Background()

	// Load daily usage data
	dailyUsage, err := st.repository.GetDailyUsage(ctx, date)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty data if no data exists for this date
			return &types.UsageData{
				TotalTime: 0,
				Apps:      []types.AppUsage{},
			}, nil
		}
		return nil, err
	}

	// Load app usage data
	appUsages, err := st.repository.GetAppUsageByDate(ctx, date)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	// Sort apps by duration (descending)
	if len(appUsages) > 0 {
		st.sortAppsByDuration(appUsages)
	}

	return &types.UsageData{
		TotalTime: dailyUsage.TotalTime,
		Apps:      appUsages,
	}, nil
}
