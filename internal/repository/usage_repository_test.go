package repository

import (
	"context"
	"testing"
	"time"

	"qwin/internal/database"
	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
)

func TestNewSQLiteRepository(t *testing.T) {
	repo := setupTestRepository(t)

	if repo == nil {
		t.Fatal("NewSQLiteRepository returned nil")
	}

	if repo.db == nil {
		t.Error("Repository db is nil")
	}

	if repo.queries == nil {
		t.Error("Repository queries is nil")
	}

	if repo.logger == nil {
		t.Error("Repository logger is nil")
	}

	if repo.retryConfig == nil {
		t.Error("Repository retryConfig is nil")
	}
}

func TestNewSQLiteRepositoryWithConfig(t *testing.T) {
	// Create test database service
	config := database.TestConfig()
	logger := logging.NewDefaultLogger()
	dbService := database.NewSQLiteService(logger)

	ctx := context.Background()
	err := dbService.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer dbService.Close()

	// Run migrations
	err = dbService.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test with custom retry config
	customRetryConfig := &repoerrors.RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  50 * time.Millisecond,
		BackoffFactor: 1.5,
	}

	repo := NewSQLiteRepositoryWithConfig(dbService, customRetryConfig, nil, logger)
	if repo == nil {
		t.Fatal("NewSQLiteRepositoryWithConfig returned nil")
	}

	if repo.retryConfig.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", repo.retryConfig.MaxAttempts)
	}

	// Test with nil config (should use default)
	repo2 := NewSQLiteRepositoryWithConfig(dbService, nil, nil, logger)
	if repo2.retryConfig == nil {
		t.Error("Repository should have default retry config when nil is passed")
	}

	// Test with nil logger (should use default)
	repo3 := NewSQLiteRepositoryWithConfig(dbService, customRetryConfig, nil, nil)
	if repo3.logger == nil {
		t.Error("Repository should have default logger when nil is passed")
	}
}

// Helper function to set up a test repository
func setupTestRepository(t *testing.T) *SQLiteRepository {
	t.Helper()

	// Create test database service
	config := database.TestConfig()
	logger := logging.NewDefaultLogger()
	dbService := database.NewSQLiteService(logger)

	ctx := context.Background()
	err := dbService.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = dbService.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create repository
	repo := NewSQLiteRepository(dbService, logger)

	// Clean up function to close database when test completes
	t.Cleanup(func() {
		dbService.Close()
	})

	return repo
}
