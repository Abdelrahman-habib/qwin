package database

import (
	"context"
	"database/sql"
	queries "qwin/internal/database/generated"
)

// Service defines the interface for database service operations
// This interface abstracts database connection management, migrations, and maintenance
type Service interface {
	// Connection management
	Connect(ctx context.Context, config *Config) error
	Close() error
	Health(ctx context.Context) error

	// Database access
	DB() *sql.DB
	GetQueries() *queries.Queries
	GetPreparedQueries(ctx context.Context) (*queries.Queries, error)

	// Migration management
	Migrate(ctx context.Context) error
	GetMigrationVersion(ctx context.Context) (int64, error)

	// Maintenance operations
	Optimize(ctx context.Context) error
	GetStats() sql.DBStats
}

// MigrationManager defines the interface for database migration operations
// This interface handles schema evolution and migration management
type MigrationManager interface {
	// Migration execution
	RunMigrations(ctx context.Context) error
	GetCurrentVersion(ctx context.Context) (int64, error)

	// Migration validation
	ValidateMigrations() error
}
