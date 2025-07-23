package services

import (
	"sync"
	"time"

	"qwin/internal/platform"
	"qwin/internal/types"
)

// ScreenTimeTracker manages screen time tracking functionality
type ScreenTimeTracker struct {
	usageData    map[string]int64
	appInfoCache map[string]*platform.AppInfo
	mutex        sync.RWMutex
	lastApp      string
	lastTime     time.Time
	startTime    time.Time
	stopTracking chan bool
	windowAPI    platform.WindowAPI
}

// NewScreenTimeTracker creates a new screen time tracker
func NewScreenTimeTracker() *ScreenTimeTracker {
	return &ScreenTimeTracker{
		usageData:    make(map[string]int64),
		appInfoCache: make(map[string]*platform.AppInfo),
		startTime:    time.Now(),
		stopTracking: make(chan bool),
		windowAPI:    platform.NewWindowsAPI(),
	}
}

// Start begins the background tracking process
func (st *ScreenTimeTracker) Start() {
	go st.trackingLoop()
}

// Stop stops the tracking process
func (st *ScreenTimeTracker) Stop() {
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

// ResetUsageData resets the usage data
func (st *ScreenTimeTracker) ResetUsageData() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.usageData = make(map[string]int64)
	st.appInfoCache = make(map[string]*platform.AppInfo)
	st.startTime = time.Now()
	st.lastTime = time.Time{}
	st.lastApp = ""
}

// sortAppsByDuration sorts apps by duration in descending order
func (st *ScreenTimeTracker) sortAppsByDuration(apps []types.AppUsage) {
	for i := 0; i < len(apps)-1; i++ {
		for j := i + 1; j < len(apps); j++ {
			if apps[i].Duration < apps[j].Duration {
				apps[i], apps[j] = apps[j], apps[i]
			}
		}
	}
}
