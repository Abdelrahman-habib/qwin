package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"qwin/internal/infrastructure/logging"
	"sync"

	"github.com/pressly/goose/v3"
)

// Embed migration files at compile time
//
//go:embed migrations/*.sql
var embedMigrations embed.FS

// Global goose configuration state protection
// goose.SetDialect() and goose.SetBaseFS() modify global package state,
// which can cause race conditions when multiple MigrationRunners are created
// concurrently (e.g., in parallel tests). We use sync.Once to ensure these
// are configured exactly once across all instances.
var (
	gooseConfigOnce sync.Once
	gooseConfigErr  error
)

// MigrationRunner handles database migration operations
// It implements the MigrationManager interface
type MigrationRunner struct {
	db     *sql.DB
	logger logging.Logger
}

// Ensure MigrationRunner implements MigrationManager interface
var _ MigrationManager = (*MigrationRunner)(nil)

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB, logger logging.Logger) *MigrationRunner {
	// Ensure logger is never nil by providing a default
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Configure goose globals once to avoid race conditions
	gooseConfigOnce.Do(func() {
		gooseConfigErr = configureGoose()
	})

	return &MigrationRunner{
		db:     db,
		logger: logger,
	}
}

// configureGoose sets up global goose configuration once
func configureGoose() error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	goose.SetBaseFS(embedMigrations)
	return nil
}

// RunMigrations executes all pending migrations using embedded files
func (mr *MigrationRunner) RunMigrations(ctx context.Context) error {
	if mr.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check if goose configuration failed during initialization
	if gooseConfigErr != nil {
		return fmt.Errorf("goose configuration failed: %w", gooseConfigErr)
	}

	mr.logger.Info("Running database migrations from embedded filesystem")

	if err := goose.UpContext(ctx, mr.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Log current version
	if version, err := goose.GetDBVersionContext(ctx, mr.db); err == nil {
		mr.logger.Info("Database migrated to version", "version", version)
	}

	return nil
}

// GetCurrentVersion returns the current migration version
func (mr *MigrationRunner) GetCurrentVersion(ctx context.Context) (int64, error) {
	if mr.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Check if goose configuration failed during initialization
	if gooseConfigErr != nil {
		return 0, fmt.Errorf("goose configuration failed: %w", gooseConfigErr)
	}

	version, err := goose.GetDBVersionContext(ctx, mr.db)
	if err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}

// ValidateMigrations checks if embedded migration files are valid
func (mr *MigrationRunner) ValidateMigrations() error {
	// Check if goose configuration failed during initialization
	if gooseConfigErr != nil {
		return fmt.Errorf("goose configuration failed: %w", gooseConfigErr)
	}

	migrations, err := goose.CollectMigrations("migrations", 0, goose.MaxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}

	if len(migrations) == 0 {
		return fmt.Errorf("no migrations found in embedded filesystem")
	}

	mr.logger.Info("Found valid migrations in embedded filesystem", "count", len(migrations))
	return nil
}

// defaultLogger provides a fallback logger implementation when none is provided
type defaultLogger struct{}

func (dl *defaultLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.Printf("[INFO] %s %v", msg, args)
	} else {
		log.Printf("[INFO] %s", msg)
	}
}

func (dl *defaultLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.Printf("[ERROR] %s %v", msg, args)
	} else {
		log.Printf("[ERROR] %s", msg)
	}
}

func (dl *defaultLogger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.Printf("[DEBUG] %s %v", msg, args)
	} else {
		log.Printf("[DEBUG] %s", msg)
	}
}

func (dl *defaultLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.Printf("[WARN] %s %v", msg, args)
	} else {
		log.Printf("[WARN] %s", msg)
	}
}
