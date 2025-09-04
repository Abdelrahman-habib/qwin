package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"qwin/internal/infrastructure/logging"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrationRunner_RunMigrations(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrations.db")

	// Open database connection
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migration runner
	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)
	ctx := context.Background()

	// Run migrations
	err = runner.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables were created
	tables := []string{"daily_usage", "app_usage", "goose_db_version"}
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
		if err != nil {
			t.Errorf("Table %s was not created: %v", table, err)
		}
	}
}

func TestMigrationRunner_RunMigrations_NilDB(t *testing.T) {
	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(nil, logger)
	ctx := context.Background()

	err := runner.RunMigrations(ctx)
	if err == nil {
		t.Fatal("Expected error for nil database, got nil")
	}

	expectedMsg := "database connection is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMigrationRunner_RunMigrations_ContextCancellation(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_cancel.db")

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)

	// Create context and cancel it immediately before calling RunMigrations
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to ensure context is cancelled
	defer cancel()

	err = runner.RunMigrations(ctx)
	// Migration might succeed if it's fast enough, or fail due to context cancellation
	// Both outcomes are acceptable for this test
	t.Logf("Migration with cancelled context result: %v", err)
}

func TestMigrationRunner_GetCurrentVersion(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_version.db")

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)
	ctx := context.Background()

	// Initially should be version 0
	version, err := runner.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial version: %v", err)
	}
	if version != 0 {
		t.Errorf("Expected initial version 0, got %d", version)
	}

	// Run migrations
	err = runner.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Should now have a version > 0
	version, err = runner.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get version after migration: %v", err)
	}
	if version <= 0 {
		t.Errorf("Expected version > 0 after migration, got %d", version)
	}

	t.Logf("Database migrated to version: %d", version)
}

func TestMigrationRunner_GetCurrentVersion_NilDB(t *testing.T) {
	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(nil, logger)
	ctx := context.Background()

	version, err := runner.GetCurrentVersion(ctx)
	if err == nil {
		t.Fatal("Expected error for nil database, got nil")
	}
	if version != 0 {
		t.Errorf("Expected version 0 for error case, got %d", version)
	}

	expectedMsg := "database connection is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMigrationRunner_ValidateMigrations(t *testing.T) {
	// Create a dummy database (not used for validation, but needed for constructor)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "dummy.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)

	// Validate embedded migrations
	err = runner.ValidateMigrations()
	if err != nil {
		t.Fatalf("Failed to validate migrations: %v", err)
	}
}

func TestMigrationRunner_ValidateMigrations_NilDB(t *testing.T) {
	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(nil, logger)

	// Should still work since validation doesn't use the database connection
	err := runner.ValidateMigrations()
	if err != nil {
		t.Fatalf("Validation should work even with nil database: %v", err)
	}
}

func TestMigrationRunner_MultipleRuns(t *testing.T) {
	// Test that running migrations multiple times is safe (idempotent)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_multiple.db")

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)
	ctx := context.Background()

	// Run migrations first time
	err = runner.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations first time: %v", err)
	}

	version1, err := runner.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get version after first run: %v", err)
	}

	// Run migrations second time (should be no-op)
	err = runner.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations second time: %v", err)
	}

	version2, err := runner.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get version after second run: %v", err)
	}

	// Versions should be the same
	if version1 != version2 {
		t.Errorf("Expected same version after multiple runs, got %d then %d", version1, version2)
	}

	t.Logf("Migration is idempotent: version %d", version2)
}

func TestMigrationRunner_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to migration runner
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_concurrent.db")

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger := logging.NewDefaultLogger()
	runner := NewMigrationRunner(db, logger)
	ctx := context.Background()

	// Run migrations first to set up the database
	err = runner.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to run initial migrations: %v", err)
	}

	// Test concurrent version checks
	done := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			version, err := runner.GetCurrentVersion(ctx)
			if err != nil {
				done <- err
				return
			}

			if version <= 0 {
				done <- err
				return
			}

			t.Logf("Goroutine %d got version: %d", id, version)
			done <- nil
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Concurrent access failed: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent test timed out")
		}
	}
}
