package repository

import (
	"context"
	"testing"
	"time"

	repoerrors "qwin/internal/infrastructure/errors"
	"qwin/internal/infrastructure/logging"
)

func TestSQLiteRepository_HealthCheck(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Test health check
	err := repo.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck should pass: %v", err)
	}
}

func TestSQLiteRepository_ConfigurationMethods(t *testing.T) {
	repo := setupTestRepository(t)

	// Test SetRetryConfig
	newConfig := &repoerrors.RetryConfig{
		MaxAttempts:   10,
		InitialDelay:  100 * time.Millisecond,
		BackoffFactor: 3.0,
	}

	repo.SetRetryConfig(newConfig)
	retrievedConfig := repo.GetRetryConfig()

	if retrievedConfig.MaxAttempts != 10 {
		t.Errorf("Expected MaxAttempts 10, got %d", retrievedConfig.MaxAttempts)
	}

	// Test SetRetryConfig with nil (should not change)
	originalConfig := repo.GetRetryConfig()
	repo.SetRetryConfig(nil)
	if repo.GetRetryConfig() != originalConfig {
		t.Error("SetRetryConfig(nil) should not change config")
	}

	// Test SetLogger
	newLogger := logging.NewDefaultLogger()
	repo.SetLogger(newLogger)

	// Test SetLogger with nil (should not change)
	repo.SetLogger(nil)
	// Can't easily test logger change without exposing it, but method should not panic

	// Test batch configuration methods
	// Test GetBatchConfig
	batchConfig := repo.GetBatchConfig()
	if batchConfig == nil {
		t.Error("GetBatchConfig should not return nil")
	}

	// Test SetBatchConfig
	newBatchConfig := &BatchConfig{
		DefaultBatchSize: 200,
		MaxBatchSize:     2000,
	}
	repo.SetBatchConfig(newBatchConfig)
	retrievedBatchConfig := repo.GetBatchConfig()

	if retrievedBatchConfig.DefaultBatchSize != 200 {
		t.Errorf("Expected DefaultBatchSize 200, got %d", retrievedBatchConfig.DefaultBatchSize)
	}

	if retrievedBatchConfig.MaxBatchSize != 2000 {
		t.Errorf("Expected MaxBatchSize 2000, got %d", retrievedBatchConfig.MaxBatchSize)
	}

	// Test SetBatchConfig with nil (should not change)
	originalBatchConfig := repo.GetBatchConfig()
	repo.SetBatchConfig(nil)
	if repo.GetBatchConfig() != originalBatchConfig {
		t.Error("SetBatchConfig(nil) should not change config")
	}

	// Test SetDynamicBatchSize
	err := repo.SetDynamicBatchSize("test_operation", 150)
	if err != nil {
		t.Errorf("SetDynamicBatchSize should succeed: %v", err)
	}

	updatedConfig := repo.GetBatchConfig()
	if updatedConfig.DefaultBatchSize != 150 {
		t.Errorf("Expected DefaultBatchSize 150 after dynamic update, got %d", updatedConfig.DefaultBatchSize)
	}

	// Test SetDynamicBatchSize validation: zero batch size
	err = repo.SetDynamicBatchSize("test_operation", 0)
	if err == nil {
		t.Error("SetDynamicBatchSize should reject zero batch size")
	}
	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for zero batch size, got: %v", err)
	}

	// Test SetDynamicBatchSize validation: negative batch size
	err = repo.SetDynamicBatchSize("test_operation", -10)
	if err == nil {
		t.Error("SetDynamicBatchSize should reject negative batch size")
	}
	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for negative batch size, got: %v", err)
	}

	// Test SetDynamicBatchSize validation: batch size exceeding maximum
	err = repo.SetDynamicBatchSize("test_operation", 3000) // Exceeds MaxBatchSize of 2000
	if err == nil {
		t.Error("SetDynamicBatchSize should reject batch size exceeding maximum")
	}
	if !repoerrors.IsValidation(err) {
		t.Errorf("Expected validation error for batch size exceeding maximum, got: %v", err)
	}
}
