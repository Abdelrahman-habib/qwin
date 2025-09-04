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
	running            bool // Protected by mutex, indicates if service is active
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
		// startTime will be set when Start() is called
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
	// Check if already running and return early if so
	st.mutex.Lock()
	if st.running {
		st.mutex.Unlock()
		return // Already running, avoid duplicate tracking/persistence
	}
	
	// Initialize start time now that tracking is actually beginning
	now := time.Now()
	if st.startTime.IsZero() {
		st.startTime = now
	}
	
	// Mark as running
	st.running = true
	st.mutex.Unlock()

	// Load existing data for today
	st.loadTodaysData()

	// Start tracking loop
	go st.trackingLoop()

	// Start persistence loop (every 30 seconds)
	st.startPersistenceLoop()
}

// Stop stops the tracking process
func (st *ScreenTimeTracker) Stop() {
	// Clear running state under mutex protection
	st.mutex.Lock()
	if !st.running {
		st.mutex.Unlock()
		return // Already stopped
	}
	st.running = false
	st.mutex.Unlock()

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

	// Calculate total time since start (only if tracking has been started)
	var totalTime int64
	if !st.startTime.IsZero() {
		totalTime = int64(time.Since(st.startTime).Seconds())
	}

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

// CurrentDate returns the current date being tracked
func (st *ScreenTimeTracker) CurrentDate() time.Time {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	return st.currentDate
}

// IsRunning returns whether the tracker is currently running
func (st *ScreenTimeTracker) IsRunning() bool {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	return st.running
}

// sortAppsByDuration sorts apps by duration in descending order
func (st *ScreenTimeTracker) sortAppsByDuration(apps []types.AppUsage) {
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Duration > apps[j].Duration
	})
}
