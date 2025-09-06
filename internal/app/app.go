package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"qwin/internal/database"
	"qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"qwin/internal/repository"
	"qwin/internal/services"
	"qwin/internal/types"
)

const (
	// persistenceWaitTime is the duration to wait for pending operations to complete during shutdown
	persistenceWaitTime = 2 * time.Second
)

// App struct represents the main application
type App struct {
	ctx         context.Context
	tracker     *services.ScreenTimeTracker
	environment string
	dbService   database.Service
	repository  repository.UsageRepository
	logger      logging.Logger
}

// NewApp creates a new App application struct with dependency injection
func NewApp(env string) (*App, error) {
	// Initialize logger first (required by all other components)
	logger := logging.NewDefaultLogger()

	// Initialize database configuration based on environment
	config := database.ConfigForEnvironment(env)

	// Initialize database service with logger
	dbService := database.NewSQLiteService(logger)
	if err := dbService.Connect(context.Background(), config); err != nil {
		return nil, err
	}

	// Run database migrations
	if err := dbService.Migrate(context.Background()); err != nil {
		dbService.Close()
		return nil, err
	}

	// Initialize repository with database service and logger
	repo := repository.NewSQLiteRepository(dbService, logger)

	// Initialize services with repository dependency
	tracker := services.NewScreenTimeTracker(repo, logger)

	return &App{
		tracker:     tracker,
		environment: env,
		dbService:   dbService,
		repository:  repo,
		logger:      logger,
	}, nil
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize database and run migrations with proper error handling
	if err := a.initializeDatabase(ctx); err != nil {
		log.Printf("Database initialization failed: %v", err)
		// Implement graceful degradation - continue without persistence
		log.Printf("Continuing without database persistence - data will not be saved")
		// TODO: Notify user through UI that data persistence is unavailable
		a.tracker.SetPersistenceEnabled(false) // Add a flag to tracker to indicate no persistence
	}

	// Start the screen time tracker
	a.tracker.Start()

	log.Printf("Application started successfully in %s mode", a.environment)
}

// initializeDatabase handles database initialization with proper error handling
func (a *App) initializeDatabase(ctx context.Context) error {
	// Check if database service is available
	if a.dbService == nil {
		return errors.NewRepositoryError("startup",
			fmt.Errorf("database service not initialized"),
			errors.ErrCodeConnection)
	}

	// Check database health with timeout
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := a.dbService.Health(healthCtx); err != nil {
		// Try to reconnect if health check fails, but not for schema errors
		if errors.IsRetryable(err) {
			if reconnectErr := a.reconnectDatabase(ctx); reconnectErr != nil {
				return reconnectErr
			}
		} else {
			return errors.NewRepositoryErrorWithContext("startup",
				err,
				errors.ClassifyError(err),
				map[string]string{
					"operation": "health_check",
				})
		}
	}

	log.Printf("Database initialization completed successfully")
	return nil
}

// reconnectDatabase handles database reconnection and migration
func (a *App) reconnectDatabase(ctx context.Context) error {
	log.Printf("Database connection lost, attempting to reconnect...")

	// Attempt reconnection with timeout
	reconnectCtx, reconnectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer reconnectCancel()

	config := database.ConfigForEnvironment(a.environment)
	if err := a.dbService.Connect(reconnectCtx, config); err != nil {
		return errors.NewRepositoryErrorWithContext("startup",
			err,
			errors.ErrCodeConnection,
			map[string]string{
				"operation": "reconnect",
				"db_path":   config.Path,
			})
	}

	// Run migrations after successful reconnection
	migrateCtx, migrateCancel := context.WithTimeout(ctx, 30*time.Second)
	defer migrateCancel()

	if err := a.dbService.Migrate(migrateCtx); err != nil {
		return errors.NewRepositoryErrorWithContext("startup",
			err,
			errors.ErrCodeConnection,
			map[string]string{
				"operation": "migrate",
				"db_path":   config.Path,
			})
	}

	log.Printf("Database reconnected and migrations completed successfully")
	return nil
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
	log.Printf("Starting application shutdown sequence...")

	// Create a timeout context for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Ensure final data persistence before shutdown
	if err := a.ensureFinalDataPersistence(shutdownCtx); err != nil {
		log.Printf("Warning: Failed to persist final data during shutdown: %v", err)
	}

	// Stop the tracker after ensuring data persistence
	a.tracker.Stop()

	// Close database connection with proper error handling
	if err := a.closeDatabaseConnection(shutdownCtx); err != nil {
		log.Printf("Error during database closure: %v", err)
	}

	log.Printf("Application shutdown completed")
}

// ensureFinalDataPersistence saves any pending data before shutdown
func (a *App) ensureFinalDataPersistence(ctx context.Context) error {
	if a.tracker == nil {
		return nil // No tracker to persist data from
	}

	log.Printf("Ensuring final data persistence...")

	// Save current data immediately
	if err := a.tracker.SaveCurrentDataNow(); err != nil {
		return errors.NewRepositoryErrorWithContext("shutdown",
			err,
			errors.ClassifyError(err),
			map[string]string{
				"operation": "final_persist",
			})
	}

	// Wait a moment to ensure all pending operations complete
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(persistenceWaitTime):
		// Continue
	}

	log.Printf("Final data persistence completed")
	return nil
}

// closeDatabaseConnection properly closes the database connection with timeout handling
func (a *App) closeDatabaseConnection(ctx context.Context) error {
	if a.dbService == nil {
		return nil // No database service to close
	}

	log.Printf("Closing database connection...")

	// Create a channel to handle the close operation
	done := make(chan error, 1)

	go func() {
		closeErrCh := make(chan error, 1)

		// Run the actual close operation in a separate goroutine
		go func() {
			closeErrCh <- a.dbService.Close()
		}()

		// Wait for either close completion or context cancellation
		select {
		case closeErr := <-closeErrCh:
			// Close operation completed, send result to main goroutine
			select {
			case done <- closeErr:
				// Successfully sent
			case <-ctx.Done():
				// Main goroutine gave up waiting, exit without blocking
			}
		case <-ctx.Done():
			// Context cancelled before close completed, send timeout error
			select {
			case done <- ctx.Err():
				// Successfully sent timeout error
			default:
				// Main goroutine already gave up, exit without blocking
			}
		}
	}()

	// Wait for close operation to complete or timeout
	select {
	case err := <-done:
		if err != nil {
			return errors.NewRepositoryErrorWithContext("shutdown",
				err,
				errors.ClassifyError(err),
				map[string]string{
					"operation": "close_connection",
				})
		}
		log.Printf("Database connection closed successfully")
		return nil
	case <-ctx.Done():
		log.Printf("Database close operation timed out")
		return errors.NewRepositoryError("shutdown",
			ctx.Err(),
			errors.ErrCodeTimeout)
	}
}

// GetUsageData returns the current usage data for the frontend
func (a *App) GetUsageData() *types.UsageData {
	return a.tracker.GetUsageData()
}

// ResetUsageData resets the usage data (for daily reset)
func (a *App) ResetUsageData() {
	a.tracker.ResetUsageData()
}

// GetHistoricalUsage returns usage data for the specified number of days back
func (a *App) GetHistoricalUsage(days int) (map[string]*types.UsageData, error) {
	return a.tracker.GetHistoricalUsage(days)
}

// GetUsageForDate returns usage data for a specific date
func (a *App) GetUsageForDate(year, month, day int) (*types.UsageData, error) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return a.tracker.GetUsageForDate(date)
}

// GetUsageForDateRange returns app usage data for a date range
func (a *App) GetUsageForDateRange(startYear, startMonth, startDay, endYear, endMonth, endDay int) ([]types.AppUsage, error) {
	startDate := time.Date(startYear, time.Month(startMonth), startDay, 0, 0, 0, 0, time.Local)
	// Use start of next day for inclusive end date
	endDate := time.Date(endYear, time.Month(endMonth), endDay+1, 0, 0, 0, 0, time.Local)
	return a.tracker.GetUsageForDateRange(startDate, endDate)
}

// GetAppUsageHistory returns historical usage data for a specific application
func (a *App) GetAppUsageHistory(appName string, days int) ([]types.AppUsage, error) {
	return a.tracker.GetAppUsageHistory(appName, days)
}

// SaveCurrentDataNow immediately saves current usage data to the database
func (a *App) SaveCurrentDataNow() error {
	return a.tracker.SaveCurrentDataNow()
}

// CleanupOldData removes usage data older than the specified number of days
func (a *App) CleanupOldData(retentionDays int) error {
	return a.tracker.CleanupOldData(retentionDays)
}

// GetLogger returns the application's structured logger
func (a *App) GetLogger() logging.Logger {
	return a.logger
}
