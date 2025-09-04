package database

import (
	"context"
	"database/sql"
	"fmt"
	queries "qwin/internal/database/generated"
	dberrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteService implements the Service interface for SQLite
//
// Lifecycle:
// 1. Create service with NewSQLiteService()
// 2. Connect to database with Connect()
// 3. Optionally run migrations with Migrate()
// 4. Use GetQueries() for regular queries or GetPreparedQueries() for prepared statements
// 5. Close service with Close() to clean up all resources including prepared statements
type SQLiteService struct {
	db              *sql.DB
	config          *Config
	migrationRunner MigrationManager
	queries         *queries.Queries
	prepared        *queries.Queries // Centralized prepared statements
	preparedMu      sync.RWMutex     // Protects lazy initialization of prepared statements
	logger          logging.Logger
}

// NewSQLiteService creates a new SQLite database service
func NewSQLiteService(logger logging.Logger) *SQLiteService {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}
	return &SQLiteService{
		logger: logger,
	}
}

// Connect establishes a connection to the SQLite database
func (s *SQLiteService) Connect(ctx context.Context, config *Config) error {
	s.config = config

	// Close any existing connection to prevent resource leaks
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Failed to close existing database connection", "error", err)
			// Continue with new connection even if close fails
		}
		// Clear references to prevent accidental reuse
		s.db = nil
		s.queries = nil
		s.migrationRunner = nil

		// Clear prepared statements
		s.preparedMu.Lock()
		if s.prepared != nil {
			if err := s.prepared.Close(); err != nil {
				s.logger.Error("Failed to close existing prepared statements", "error", err)
			}
			s.prepared = nil
		}
		s.preparedMu.Unlock()
	}

	// Build connection string with configuration options
	connStr := config.GetConnectionString()

	// Open database connection
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return dberrors.HandleConnectionError("Connect", fmt.Sprintf("failed to open database: %v", err))
	}

	// Configure connection pool based on SQLite capabilities
	s.configureConnectionPool(db, config)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return dberrors.HandleConnectionError("Connect", fmt.Sprintf("failed to ping database: %v", err))
	}

	s.db = db
	s.queries = queries.New(db)

	// Initialize migration runner
	s.migrationRunner = NewMigrationRunner(db, s.logger)

	s.logger.Info("Connected to SQLite database", "path", config.Path)
	return nil
}

// Close closes the database connection
func (s *SQLiteService) Close() error {
	if s.db == nil {
		return nil
	}

	// Close prepared statements first to avoid masking errors
	s.preparedMu.Lock()
	if s.prepared != nil {
		if err := s.prepared.Close(); err != nil {
			s.logger.Error("Failed to close prepared statements", "error", err)
			// Continue with cleanup even if prepared statements fail to close
		}
		s.prepared = nil
	}
	s.preparedMu.Unlock()

	// Close database connection
	if err := s.db.Close(); err != nil {
		return dberrors.HandleConnectionError("Close", fmt.Sprintf("failed to close database: %v", err))
	}

	// Null out remaining internal references to prevent accidental reuse
	s.db = nil
	s.queries = nil
	s.migrationRunner = nil

	s.logger.Info("Closed SQLite database connection")
	return nil
}

// Migrate runs database migrations using the migration runner
func (s *SQLiteService) Migrate(ctx context.Context) error {
	if s.db == nil {
		return dberrors.HandleConnectionError("Migrate", "database not connected")
	}

	if s.migrationRunner == nil {
		return dberrors.HandleValidationError("Migrate", "migrationRunner", "nil", "migration runner not initialized")
	}

	// Validate migrations first
	if err := s.migrationRunner.ValidateMigrations(); err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Migrate", err, map[string]string{
			"phase": "validation",
		})
	}

	// Run migrations
	if err := s.migrationRunner.RunMigrations(ctx); err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Migrate", err, map[string]string{
			"phase": "execution",
		})
	}

	return nil
}

// Health checks the database connection health
func (s *SQLiteService) Health(ctx context.Context) error {
	if s.db == nil {
		return dberrors.HandleConnectionError("Health", "database not connected")
	}

	// Simple ping to check connection
	if err := s.db.PingContext(ctx); err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Health", err, map[string]string{
			"phase": "ping",
		})
	}

	// Test with a simple query
	var result int
	err := s.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Health", err, map[string]string{
			"phase": "query",
		})
	}

	if result != 1 {
		return dberrors.HandleValidationError("Health", "query_result", fmt.Sprintf("%d", result), "expected result 1")
	}

	return nil
}

// DB returns the underlying database connection for use by repositories
func (s *SQLiteService) DB() *sql.DB {
	return s.db
}

// GetQueries returns the queries instance for repository use
func (s *SQLiteService) GetQueries() *queries.Queries {
	return s.queries
}

// GetMigrationVersion returns the current migration version
func (s *SQLiteService) GetMigrationVersion(ctx context.Context) (int64, error) {
	if s.db == nil {
		return 0, dberrors.HandleConnectionError("GetMigrationVersion", "database not connected")
	}
	if s.migrationRunner == nil {
		return 0, dberrors.HandleValidationError("GetMigrationVersion", "migrationRunner", "nil", "migration runner not initialized")
	}

	version, err := s.migrationRunner.GetCurrentVersion(ctx)
	if err != nil {
		return 0, dberrors.WrapDatabaseError("GetMigrationVersion", err)
	}
	return version, nil
}

// GetPreparedQueries returns a centralized prepared queries instance for better performance
// The prepared statements are managed by the service and closed automatically when Close() is called
func (s *SQLiteService) GetPreparedQueries(ctx context.Context) (*queries.Queries, error) {
	if s.db == nil {
		return nil, dberrors.HandleConnectionError("GetPreparedQueries", "database not connected")
	}

	// Fast path: check if prepared queries already exist (read lock)
	s.preparedMu.RLock()
	if s.prepared != nil {
		prepared := s.prepared
		s.preparedMu.RUnlock()
		return prepared, nil
	}
	s.preparedMu.RUnlock()

	// Slow path: need to create prepared queries (write lock)
	s.preparedMu.Lock()
	defer s.preparedMu.Unlock()

	// Double-check pattern: another goroutine might have created it while we waited
	if s.prepared != nil {
		return s.prepared, nil
	}

	// Create prepared statements for better performance
	preparedQueries, err := queries.Prepare(ctx, s.db)
	if err != nil {
		return nil, dberrors.WrapDatabaseError("GetPreparedQueries", err)
	}

	// Store prepared queries for centralized management
	s.prepared = preparedQueries
	return s.prepared, nil
}

// GetStats returns database connection pool statistics for monitoring
func (s *SQLiteService) GetStats() sql.DBStats {
	if s.db == nil {
		return sql.DBStats{}
	}
	return s.db.Stats()
}

// Optimize runs VACUUM and ANALYZE to optimize database performance
func (s *SQLiteService) Optimize(ctx context.Context) error {
	if s.db == nil {
		return dberrors.HandleConnectionError("Optimize", "database not connected")
	}

	// Run ANALYZE to update query planner statistics
	if _, err := s.db.ExecContext(ctx, "ANALYZE"); err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Optimize", err, map[string]string{
			"phase": "analyze",
		})
	}

	// Best-effort WAL checkpoint to trim .wal (ignored on non-WAL)
	if _, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil && s.logger != nil {
		s.logger.Warn("wal_checkpoint failed", "error", err)
	}

	// Run VACUUM to reclaim space and defragment
	if _, err := s.db.ExecContext(ctx, "VACUUM"); err != nil {
		return dberrors.WrapDatabaseErrorWithContext("Optimize", err, map[string]string{
			"phase": "vacuum",
		})
	}

	// Let SQLite apply additional internal optimizations (no-op if unsupported)
	if _, err := s.db.ExecContext(ctx, "PRAGMA optimize"); err != nil && s.logger != nil {
		s.logger.Warn("PRAGMA optimize failed", "error", err)
	}

	s.logger.Info("Database optimization completed")
	return nil
}

// configureConnectionPool sets up connection pool settings optimized for SQLite
func (s *SQLiteService) configureConnectionPool(db *sql.DB, config *Config) {
	// Check if we should force single connection mode
	if config.ForceSingleConnection {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		s.logger.Info("Configured SQLite for single connection mode (forced by config)")
		return
	}

	// Detect SQLite-specific constraints
	isSQLite := true // We know this is SQLite service
	isWALEnabled := strings.EqualFold(config.JournalMode, "WAL")

	if isSQLite && !isWALEnabled {
		// SQLite without WAL mode should use single connection to avoid locking issues
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		s.logger.Info("Configured SQLite for single connection mode (non-WAL journal mode)",
			"journalMode", config.JournalMode)
	} else if isSQLite && isWALEnabled {
		// SQLite with WAL can handle multiple readers, but keep it conservative
		// Compute maxConns from config but ensure it's > 0 and cap at 4
		maxConns := config.MaxConnections
		if maxConns <= 0 {
			maxConns = 4 // Set sane default if <= 0
		}
		if maxConns > 4 {
			maxConns = 4 // Cap at 4 for SQLite even with WAL
		}

		// Compute idleConns as min of config and maxConns, ensure > 0
		idleConns := min(config.MaxIdleConns, maxConns)
		if idleConns <= 0 {
			idleConns = 1 // Set minimum idle connections
		}

		db.SetMaxOpenConns(maxConns)
		db.SetMaxIdleConns(idleConns)
		s.logger.Info("Configured SQLite for limited connection pool (WAL mode)",
			"maxOpenConns", maxConns, "maxIdleConns", idleConns)
	} else {
		// Use configured values for other databases (future-proofing)
		db.SetMaxOpenConns(config.MaxConnections)
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	// Set connection lifetime settings
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
