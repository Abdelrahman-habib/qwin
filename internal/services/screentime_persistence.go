package services

import (
	"context"
	"log"
	"time"

	"qwin/internal/infrastructure/errors"
	"qwin/internal/platform"
	"qwin/internal/types"
)

// startPersistenceLoop starts the periodic data persistence (every 30 seconds)
func (st *ScreenTimeTracker) startPersistenceLoop() {
	st.persistTicker = time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-st.persistTicker.C:
				st.persistCurrentData()
			case <-st.stopTracking:
				// Stop the ticker to prevent timer leak
				if st.persistTicker != nil {
					st.persistTicker.Stop()
				}
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

	st.mutex.Lock()
	defer st.mutex.Unlock()

	// Check if we need to handle date rollover
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !today.Equal(st.currentDate) {
		// Date has changed, persist old data and reset for new day
		st.persistDataForDate(ctx, st.currentDate)
		st.currentDate = today
		st.usageData = make(map[string]int64)
		st.startTime = now
		return
	}

	// Persist current day's data
	st.persistDataForDate(ctx, st.currentDate)
	st.lastPersist = now
}

// persistDataForDate saves usage data for a specific date
func (st *ScreenTimeTracker) persistDataForDate(ctx context.Context, date time.Time) {
	// Calculate total time bounded to the target date to prevent midnight rollover corruption
	var totalTime int64
	if !st.startTime.IsZero() {
		// Calculate start and end boundaries of the target date
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
		
		// Clamp the time interval to the target date boundaries
		start := st.startTime
		if start.Before(startOfDay) {
			start = startOfDay
		}
		
		end := time.Now()
		if end.After(endOfDay) {
			end = endOfDay
		}
		
		// Calculate elapsed time only within the target date boundaries
		if end.After(start) {
			totalTime = int64(end.Sub(start).Seconds())
		} else {
			totalTime = 0 // Clamp negatives to 0
		}
	}
	// If startTime is zero (Start() not called), totalTime remains 0

	// Save daily usage summary
	usageData := &types.UsageData{
		TotalTime: totalTime,
	}

	if err := st.repository.SaveDailyUsage(ctx, date, usageData); err != nil {
		log.Printf("Failed to save daily usage: %v", err)
	}

	// Prepare app usage data for batch save
	appUsages := make([]types.AppUsage, 0, len(st.usageData))
	for name, duration := range st.usageData {
		appUsage := types.AppUsage{
			Name:     name,
			Duration: duration,
			Date:     date,
		}

		// Add cached app info if available
		if cachedInfo, exists := st.appInfoCache[name]; exists {
			appUsage.IconPath = cachedInfo.IconPath
			appUsage.ExePath = cachedInfo.ExePath
		}

		appUsages = append(appUsages, appUsage)
	}

	// Batch save app usage data
	if len(appUsages) > 0 {
		if err := st.repository.BatchProcessAppUsage(ctx, date, appUsages, types.BatchStrategyUpsert); err != nil {
			log.Printf("Failed to save app usage data: %v", err)
		}
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
