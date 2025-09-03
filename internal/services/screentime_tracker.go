package services

import (
	"sort"
	"sync"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/platform"
	"qwin/internal/repository"
	"qwin/internal/types"
)

// ScreenTimeTracker manages screen time tracking functionality
type ScreenTimeTracker struct {
	usageData          map[string]int64
	appInfoCache       map[string]*platform.AppInfo
	mutex              sync.RWMutex
	lastApp            string
	lastTime           time.Time
	startTime          time.Time
	stopTracking       chan bool
	windowAPI          platform.WindowAPI
	repository         repository.UsageRepository
	logger             logging.Logger
	persistTicker      *time.Ticker
	lastPersist        time.Time
	currentDate        time.Time
	persistenceEnabled bool
}

// NewScreenTimeTracker creates a new screen time tracker with repository dependency
func NewScreenTimeTracker(repo repository.UsageRepository, logger logging.Logger) *ScreenTimeTracker {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	now := time.Now()
	return &ScreenTimeTracker{
		usageData:          make(map[string]int64),
		appInfoCache:       make(map[string]*platform.AppInfo),
		startTime:          now,
		stopTracking:       make(chan bool),
		windowAPI:          platform.NewWindowAPI(),
		repository:         repo,
		logger:             logger,
		currentDate:        time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		persistenceEnabled: true, // Default to enabled
	}
}

// Start begins the background tracking process
func (st *ScreenTimeTracker) Start() {
	// Load existing data for today
	st.loadTodaysData()

	// Start tracking loop
	go st.trackingLoop()

	// Start persistence loop (every 30 seconds)
	st.startPersistenceLoop()
}

// Stop stops the tracking process
func (st *ScreenTimeTracker) Stop() {
	// Stop persistence ticker
	if st.persistTicker != nil {
		st.persistTicker.Stop()
	}

	// Persist final data before stopping
	st.persistCurrentData()

	// Stop tracking
	select {
	case st.stopTracking <- true:
	default:
	}
}

// trackingLoop runs the main tracking loop
func (st *ScreenTimeTracker) trackingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			st.trackCurrentApp()
		case <-st.stopTracking:
			return
		}
	}
}

// trackCurrentApp tracks the currently active application
func (st *ScreenTimeTracker) trackCurrentApp() {
	appInfo := st.windowAPI.GetCurrentAppInfo()
	if appInfo == nil || appInfo.Name == "" {
		return
	}

	now := time.Now()

	st.mutex.Lock()
	defer st.mutex.Unlock()

	// Cache app info if not already cached
	if _, exists := st.appInfoCache[appInfo.Name]; !exists {
		st.appInfoCache[appInfo.Name] = appInfo
	}

	// If this is the same app as before, add the elapsed time
	if st.lastApp == appInfo.Name && !st.lastTime.IsZero() {
		elapsed := now.Sub(st.lastTime).Seconds()
		st.usageData[appInfo.Name] += int64(elapsed)
	}

	st.lastApp = appInfo.Name
	st.lastTime = now
}

// GetUsageData returns the current usage data
func (st *ScreenTimeTracker) GetUsageData() *types.UsageData {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	// Calculate total time since start
	totalTime := int64(time.Since(st.startTime).Seconds())

	// Convert map to sorted slice with cached app info
	apps := make([]types.AppUsage, 0, len(st.usageData))
	for name, duration := range st.usageData {
		appUsage := types.AppUsage{
			Name:     name,
			Duration: duration,
		}

		// Add cached app info if available
		if cachedInfo, exists := st.appInfoCache[name]; exists {
			appUsage.IconPath = cachedInfo.IconPath
			appUsage.ExePath = cachedInfo.ExePath
		}

		apps = append(apps, appUsage)
	}

	// Sort apps by duration (descending)
	st.sortAppsByDuration(apps)

	// Return top 5 apps
	if len(apps) > 5 {
		apps = apps[:5]
	}

	return &types.UsageData{
		TotalTime: totalTime,
		Apps:      apps,
	}
}

// sortAppsByDuration sorts apps by duration in descending order
func (st *ScreenTimeTracker) sortAppsByDuration(apps []types.AppUsage) {
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Duration > apps[j].Duration
	})
}
