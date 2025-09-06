package services

import (
	"math"
	"sort"
	"sync"
	"time"

	"qwin/internal/infrastructure/logging"
	"qwin/internal/platform"
	"qwin/internal/repository"
	"qwin/internal/types"
)

const defaultTopN = 5

// ScreenTimeTracker manages screen time tracking functionality
type ScreenTimeTracker struct {
	usageData          map[string]int64
	appInfoCache       map[string]*platform.AppInfo
	mutex              sync.RWMutex
	lastApp            string
	lastTime           time.Time
	startTime          time.Time
	running            bool // Protected by mutex, indicates if service is active
	stopTracking       chan struct{}
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
	return NewScreenTimeTrackerWithWindowAPI(repo, logger, platform.NewWindowAPI())
}

// NewScreenTimeTrackerWithWindowAPI creates a new screen time tracker with dependency injection for testing
func NewScreenTimeTrackerWithWindowAPI(repo repository.UsageRepository, logger logging.Logger, windowAPI platform.WindowAPI) *ScreenTimeTracker {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	return &ScreenTimeTracker{
		usageData:    make(map[string]int64),
		appInfoCache: make(map[string]*platform.AppInfo),
		// startTime will be set when Start() is called
		// stopTracking channel will be created in Start()
		windowAPI:  windowAPI,
		repository: repo,
		logger:     logger,
		// currentDate is initialized in Start()
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

	// Create stop tracking channel for this session
	st.stopTracking = make(chan struct{})

	// Mark as running
	st.running = true
	st.mutex.Unlock()

	// Initialize current date to today's midnight
	nowMidnight := time.Now()
	st.mutex.Lock()
	st.currentDate = time.Date(nowMidnight.Year(), nowMidnight.Month(), nowMidnight.Day(), 0, 0, 0, 0, nowMidnight.Location())
	st.mutex.Unlock()

	// Load existing data for today
	st.loadTodaysData()

	// Start tracking loop
	go st.trackingLoop()

	// Start persistence loop (every 30 seconds)
	go st.startPersistenceLoop()
}

// Stop stops the tracking process
func (st *ScreenTimeTracker) Stop() {
	// Clear running state and capture channel reference under mutex protection
	st.mutex.Lock()
	if !st.running {
		st.mutex.Unlock()
		return // Already stopped
	}
	st.running = false
	ticker := st.persistTicker
	st.persistTicker = nil
	stopCh := st.stopTracking
	st.stopTracking = nil
	wasStarted := !st.startTime.IsZero()
	st.mutex.Unlock()

	// Stop persistence ticker
	if ticker != nil {
		ticker.Stop()
	}

	// Stop tracking by closing the channel (broadcasts to all listeners)
	if stopCh != nil {
		close(stopCh)
	}

	// Attribute any final elapsed time for the last active app
	st.mutex.Lock()
	if st.lastApp != "" && !st.lastTime.IsZero() {
		elapsed := time.Since(st.lastTime).Seconds()
		if elapsed > 0 {
			st.usageData[st.lastApp] += int64(math.Round(elapsed))
		}
	}
	st.mutex.Unlock()

	// Persist final data once (only if tracking was started)
	if wasStarted {
		st.persistCurrentData()
	}
}

// trackingLoop runs the main tracking loop
func (st *ScreenTimeTracker) trackingLoop() {
	// Capture the stop channel reference at start to avoid data races
	st.mutex.RLock()
	stopCh := st.stopTracking
	st.mutex.RUnlock()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			st.trackCurrentApp()
		case <-stopCh:
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

	// Attribute elapsed time to the previously active app, if any
	if st.lastApp != "" && !st.lastTime.IsZero() {
		elapsed := now.Sub(st.lastTime).Seconds()
		if elapsed > 0 {
			st.usageData[st.lastApp] += int64(math.Round(elapsed))
		}
	}

	// Set current app as the new active app
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
		end := time.Now()
		if !st.running && !st.lastTime.IsZero() && st.lastTime.After(st.startTime) {
			end = st.lastTime
		}
		if end.After(st.startTime) {
			totalTime = int64(end.Sub(st.startTime).Seconds())
		}
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

	// Return top N apps
	if len(apps) > defaultTopN {
		apps = apps[:defaultTopN]
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
