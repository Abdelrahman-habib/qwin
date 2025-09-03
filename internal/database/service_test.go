package database

import (
	"context"
	"os"
	"path/filepath"
	dberrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestSQLiteService_Connect(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Test connection
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer service.Close()

	// Test health check
	err = service.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created: %s", dbPath)
	}
}

func TestSQLiteService_Migrate(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrate.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect to database
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer service.Close()

	// Run migrations
	err = service.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables were created by checking if we can query them
	db := service.DB()

	// Check daily_usage table
	var n int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM daily_usage").Scan(&n); err != nil {
		t.Fatalf("daily_usage table was not created: %v", err)
	}

	// Check app_usage table
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_usage").Scan(&n); err != nil {
		t.Fatalf("app_usage table was not created: %v", err)
	}
}

func TestSQLiteService_MigrationStatus(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_status.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect and migrate
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer service.Close()

	err = service.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Get current version
	version, err := service.GetMigrationVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get current migration version: %v", err)
	}

	if version <= 0 {
		t.Fatalf("Expected migration version > 0, got %d", version)
	}
}

func TestSQLiteService_ConnectionPool(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_pool.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect to database
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer service.Close()

	db := service.DB()
	if db == nil {
		t.Fatalf("Database connection is nil")
	}

	// Test concurrent access (SQLite should handle this with WAL mode)
	var wg sync.WaitGroup
	wg.Add(2)

	for i := range 2 {
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Simple query to test connection
			var result int
			err := db.QueryRowContext(ctx, "SELECT ?", id).Scan(&result)
			if err != nil {
				t.Errorf("Concurrent query %d failed: %v", id, err)
				return
			}

			if result != id {
				t.Errorf("Expected %d, got %d", id, result)
			}
		}(i)
	}

	// Wait for both goroutines to complete
	wg.Wait()
}

// Error scenario tests

func TestSQLiteService_Connect_InvalidPath(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Try to connect to invalid path (directory that doesn't exist and can't be created)
	config := DefaultConfig()
	config.Path = "/invalid/path/that/does/not/exist/test.db"

	err := service.Connect(ctx, config)

	// Log the result for additional context
	t.Logf("Connect to invalid path result: %v", err)

	// On Windows, this might succeed or fail depending on permissions and path handling
	// On Unix-like systems, this should fail deterministically
	if runtime.GOOS == "windows" {
		// On Windows, accept either outcome due to path handling differences
		t.Skip("Skipping strict assertion on Windows due to path handling differences")
	} else {
		// On Unix-like systems, this should fail
		if err == nil {
			t.Fatal("Expected error for invalid path on Unix-like system, got nil")
		}

		// Verify it's a connection error
		if !dberrors.IsConnection(err) {
			t.Errorf("Expected connection error for invalid path, got: %v", err)
		}
	}
}

func TestSQLiteService_Health_NotConnected(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Try health check without connecting
	err := service.Health(ctx)
	if err == nil {
		t.Fatal("Expected error for health check without connection, got nil")
	}

	// Verify it's a connection error using the structured error system
	if !dberrors.IsConnection(err) {
		t.Errorf("Expected connection error for health check without connection, got: %v", err)
	}
}

func TestSQLiteService_Migrate_NotConnected(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Try migration without connecting
	err := service.Migrate(ctx)
	if err == nil {
		t.Fatal("Expected error for migration without connection, got nil")
	}

	// Verify it's a connection error using the structured error system
	if !dberrors.IsConnection(err) {
		t.Errorf("Expected connection error for migration without connection, got: %v", err)
	}
}

func TestSQLiteService_GetMigrationVersion_NotConnected(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Try to get version without connecting
	version, err := service.GetMigrationVersion(ctx)
	if err == nil {
		t.Fatal("Expected error for version check without connection, got nil")
	}
	if version != 0 {
		t.Errorf("Expected version 0 for error case, got %d", version)
	}

	// Verify it's a connection error using the structured error system
	if !dberrors.IsConnection(err) {
		t.Errorf("Expected connection error for version check without connection, got: %v", err)
	}
}

func TestSQLiteService_Close_NotConnected(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)

	// Close without connecting should not error
	err := service.Close()
	if err != nil {
		t.Errorf("Close without connection should not error, got: %v", err)
	}
}

func TestSQLiteService_Close_NullsReferences(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_close.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect and migrate to populate all references
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = service.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Verify references are populated before closing
	if service.db == nil {
		t.Error("Expected db to be non-nil before closing")
	}
	if service.queries == nil {
		t.Error("Expected queries to be non-nil before closing")
	}
	if service.migrationRunner == nil {
		t.Error("Expected migrationRunner to be non-nil before closing")
	}

	// Close the service
	err = service.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Verify all references are nulled out after closing
	if service.db != nil {
		t.Error("Expected db to be nil after closing")
	}
	if service.queries != nil {
		t.Error("Expected queries to be nil after closing")
	}
	if service.migrationRunner != nil {
		t.Error("Expected migrationRunner to be nil after closing")
	}
}

func TestSQLiteService_PreparedQueries_Management(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_prepared.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect and migrate
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = service.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Get prepared queries - should create them
	prepared1, err := service.GetPreparedQueries(ctx)
	if err != nil {
		t.Fatalf("Failed to get prepared queries: %v", err)
	}
	if prepared1 == nil {
		t.Error("Expected prepared queries to be non-nil")
	}

	// Get prepared queries again - should return the same instance
	prepared2, err := service.GetPreparedQueries(ctx)
	if err != nil {
		t.Fatalf("Failed to get prepared queries second time: %v", err)
	}
	if prepared2 != prepared1 {
		t.Error("Expected same prepared queries instance to be returned")
	}

	// Verify prepared field is set
	if service.prepared == nil {
		t.Error("Expected service.prepared to be non-nil")
	}
	if service.prepared != prepared1 {
		t.Error("Expected service.prepared to match returned prepared queries")
	}

	// Close the service - should clean up prepared statements
	err = service.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Verify prepared field is nulled out
	if service.prepared != nil {
		t.Error("Expected service.prepared to be nil after closing")
	}
}

func TestSQLiteService_ConnectionPool_Configuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		journalMode           string
		forceSingleConnection bool
		maxConnections        int
		expectedMaxOpen       int
		expectedMaxIdle       int
	}{
		{
			name:            "WAL mode with default settings",
			journalMode:     "WAL",
			maxConnections:  10,
			expectedMaxOpen: 4, // Capped at 4 for SQLite with WAL
			expectedMaxIdle: 4,
		},
		{
			name:            "DELETE mode should use single connection",
			journalMode:     "DELETE",
			maxConnections:  10,
			expectedMaxOpen: 1, // Single connection for non-WAL
			expectedMaxIdle: 1,
		},
		{
			name:                  "Forced single connection",
			journalMode:           "WAL",
			forceSingleConnection: true,
			maxConnections:        10,
			expectedMaxOpen:       1, // Forced single connection
			expectedMaxIdle:       1,
		},
		{
			name:            "WAL mode with small max connections",
			journalMode:     "WAL",
			maxConnections:  2,
			expectedMaxOpen: 2, // Don't cap when already small
			expectedMaxIdle: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test_pool.db")

			config := DefaultConfig()
			config.Path = dbPath
			config.JournalMode = tt.journalMode
			config.ForceSingleConnection = tt.forceSingleConnection
			config.MaxConnections = tt.maxConnections
			config.MaxIdleConns = tt.maxConnections

			logger := logging.NewDefaultLogger()
			service := NewSQLiteService(logger)
			ctx := context.Background()

			err := service.Connect(ctx, config)
			if err != nil {
				t.Fatalf("Failed to connect to database: %v", err)
			}
			defer service.Close()

			// Check connection pool stats
			stats := service.GetStats()
			if stats.MaxOpenConnections != tt.expectedMaxOpen {
				t.Errorf("Expected MaxOpenConnections to be %d, got %d", tt.expectedMaxOpen, stats.MaxOpenConnections)
			}
		})
	}
}

func TestSQLiteService_DB_NotConnected(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)

	// DB() should return nil when not connected
	db := service.DB()
	if db != nil {
		t.Error("Expected nil database when not connected")
	}
}

func TestSQLiteService_ContextCancellation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_cancel.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)

	// Create context that is immediately cancelled for deterministic behavior
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to ensure deterministic cancellation

	// Try operations with cancelled context
	err := service.Connect(ctx, config)
	// Connection might succeed if it's fast enough
	if err == nil {
		defer service.Close()

		// Try health check with cancelled context
		err = service.Health(ctx)
		t.Logf("Health check with cancelled context: %v", err)
	} else {
		t.Logf("Connect with cancelled context failed as expected: %v", err)
	}
}

func TestSQLiteService_MultipleConnections(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_multiple.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect first time
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed first connection: %v", err)
	}

	// Close first connection before reconnecting to avoid file locking issues
	err = service.Close()
	if err != nil {
		t.Fatalf("Failed to close first connection: %v", err)
	}

	// Try to connect again
	err = service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed second connection: %v", err)
	}
	defer service.Close()

	// Health check should still work
	err = service.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed after reconnection: %v", err)
	}
}

func TestSQLiteService_HealthCheck_DatabaseCorruption(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_corrupt.db")

	config := DefaultConfig()
	config.Path = dbPath

	logger := logging.NewDefaultLogger()
	service := NewSQLiteService(logger)
	ctx := context.Background()

	// Connect and migrate normally
	err := service.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	err = service.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Health check should pass
	err = service.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed on good database: %v", err)
	}

	service.Close()

	// Write garbage to the database file to simulate corruption
	err = os.WriteFile(dbPath, []byte("this is not a valid sqlite database"), 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt database file: %v", err)
	}

	// Try to connect to corrupted database
	err = service.Connect(ctx, config)
	if err != nil {
		t.Logf("Connect to corrupted database failed as expected: %v", err)
		return
	}
	defer service.Close()

	// Health check should fail
	err = service.Health(ctx)
	if err == nil {
		t.Error("Expected health check to fail on corrupted database")
	} else {
		t.Logf("Health check correctly detected corruption: %v", err)
	}
}
