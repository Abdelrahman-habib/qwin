package app

import (
	"context"

	"qwin/internal/services"
	"qwin/internal/types"
)

// App struct represents the main application
type App struct {
	ctx     context.Context
	tracker *services.ScreenTimeTracker
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		tracker: services.NewScreenTimeTracker(),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.tracker.Start()
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// BeforeClose is called when the application is about to quit
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	a.tracker.Stop()
}

// GetUsageData returns the current usage data for the frontend
func (a *App) GetUsageData() *types.UsageData {
	return a.tracker.GetUsageData()
}

// ResetUsageData resets the usage data (for daily reset)
func (a *App) ResetUsageData() {
	a.tracker.ResetUsageData()
}
