package database

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfig_Validate_DatabasePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path should fail",
			path:        "",
			expectError: true,
			errorMsg:    "database path cannot be empty",
		},
		{
			name:        "memory database should pass",
			path:        ":memory:",
			expectError: false,
		},
		{
			name:        "valid file path should pass",
			path:        "test.db",
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate for path testing
			if tt.path == ":memory:" {
				config.JournalMode = "MEMORY" // Use compatible journal mode for in-memory
			}
			config.Path = tt.path

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_ConnectionSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "negative maxConnections should fail",
			modifier: func(c *Config) {
				c.MaxConnections = -1
			},
			expectError: true,
			errorMsg:    "maxConnections must be positive",
		},
		{
			name: "zero maxConnections should fail",
			modifier: func(c *Config) {
				c.MaxConnections = 0
			},
			expectError: true,
			errorMsg:    "maxConnections must be positive",
		},
		{
			name: "negative maxIdleConns should fail",
			modifier: func(c *Config) {
				c.MaxIdleConns = -1
			},
			expectError: true,
			errorMsg:    "maxIdleConns cannot be negative",
		},
		{
			name: "maxIdleConns > maxConnections should fail",
			modifier: func(c *Config) {
				c.MaxConnections = 5
				c.MaxIdleConns = 10
			},
			expectError: true,
			errorMsg:    "maxIdleConns (10) cannot be greater than maxConnections (5)",
		},
		{
			name: "negative connMaxLifetime should fail",
			modifier: func(c *Config) {
				c.ConnMaxLifetime = -time.Hour
			},
			expectError: true,
			errorMsg:    "connMaxLifetime cannot be negative",
		},
		{
			name: "negative connMaxIdleTime should fail",
			modifier: func(c *Config) {
				c.ConnMaxIdleTime = -time.Minute
			},
			expectError: true,
			errorMsg:    "connMaxIdleTime cannot be negative",
		},
		{
			name: "valid connection settings should pass",
			modifier: func(c *Config) {
				c.MaxConnections = 10
				c.MaxIdleConns = 5
				c.ConnMaxLifetime = time.Hour
				c.ConnMaxIdleTime = time.Minute
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on connection settings
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_MigrationsPath(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		modifier    func(*Config)
		setup       func() string
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty migrationsPath should fail",
			modifier: func(c *Config) {
				c.MigrationsPath = ""
			},
			expectError: true,
			errorMsg:    "migrationsPath cannot be empty",
		},
		{
			name: "AutoMigrate=false with non-existent path should pass",
			modifier: func(c *Config) {
				c.AutoMigrate = false
				c.MigrationsPath = "/non/existent/path"
			},
			expectError: false,
		},
		{
			name: "AutoMigrate=true with non-existent path should fail",
			modifier: func(c *Config) {
				c.AutoMigrate = true
				c.MigrationsPath = "/non/existent/path"
			},
			expectError: true,
			errorMsg:    "does not exist when AutoMigrate is enabled",
		},
		{
			name: "AutoMigrate=true with existing directory should pass",
			setup: func() string {
				migrationDir := filepath.Join(tempDir, "migrations")
				os.MkdirAll(migrationDir, 0755)
				return migrationDir
			},
			modifier: func(c *Config) {
				c.AutoMigrate = true
			},
			expectError: false,
		},
		{
			name: "AutoMigrate=true with existing file should pass",
			setup: func() string {
				migrationFile := filepath.Join(tempDir, "migrations.sql")
				os.WriteFile(migrationFile, []byte("-- test migration"), 0644)
				return migrationFile
			},
			modifier: func(c *Config) {
				c.AutoMigrate = true
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()

			if tt.setup != nil {
				path := tt.setup()
				config.MigrationsPath = path
			}

			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_JournalMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid journal mode should fail",
			modifier: func(c *Config) {
				c.JournalMode = "INVALID"
			},
			expectError: true,
			errorMsg:    "invalid journalMode: INVALID",
		},
		{
			name: "valid journal modes should pass",
			modifier: func(c *Config) {
				c.JournalMode = "WAL"
			},
			expectError: false,
		},
		{
			name: "WAL with file database should pass",
			modifier: func(c *Config) {
				c.Path = "test.db"
				c.JournalMode = "WAL"
			},
			expectError: false,
		},
		{
			name: "WAL with in-memory database should fail",
			modifier: func(c *Config) {
				c.Path = ":memory:"
				c.JournalMode = "WAL"
			},
			expectError: true,
			errorMsg:    "journalMode cannot be WAL when using in-memory database",
		},
		{
			name: "WAL case insensitive with in-memory database should fail",
			modifier: func(c *Config) {
				c.Path = ":memory:"
				c.JournalMode = "wal"
			},
			expectError: true,
			errorMsg:    "journalMode cannot be WAL when using in-memory database",
		},
		{
			name: "MEMORY with in-memory database should pass",
			modifier: func(c *Config) {
				c.Path = ":memory:"
				c.JournalMode = "MEMORY"
			},
			expectError: false,
		},
		{
			name: "DELETE with in-memory database should pass",
			modifier: func(c *Config) {
				c.Path = ":memory:"
				c.JournalMode = "DELETE"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on journal mode testing
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_PerformanceSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid synchronous mode should fail",
			modifier: func(c *Config) {
				c.SynchronousMode = "INVALID"
			},
			expectError: true,
			errorMsg:    "invalid synchronousMode: INVALID",
		},
		{
			name: "negative cache size should fail",
			modifier: func(c *Config) {
				c.CacheSize = -100
			},
			expectError: true,
			errorMsg:    "cacheSize must be positive",
		},
		{
			name: "zero cache size should fail",
			modifier: func(c *Config) {
				c.CacheSize = 0
			},
			expectError: true,
			errorMsg:    "cacheSize must be positive",
		},
		{
			name: "negative busy timeout should fail",
			modifier: func(c *Config) {
				c.BusyTimeout = -1000
			},
			expectError: true,
			errorMsg:    "busyTimeout cannot be negative",
		},
		{
			name: "zero busy timeout should pass",
			modifier: func(c *Config) {
				c.BusyTimeout = 0
			},
			expectError: false,
		},
		{
			name: "valid performance settings should pass",
			modifier: func(c *Config) {
				c.SynchronousMode = "NORMAL"
				c.CacheSize = 1000
				c.BusyTimeout = 5000
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on specific validation
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_MaintenanceSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "negative vacuum interval should fail",
			modifier: func(c *Config) {
				c.VacuumInterval = -time.Hour
			},
			expectError: true,
			errorMsg:    "vacuumInterval cannot be negative",
		},
		{
			name: "negative analyze interval should fail",
			modifier: func(c *Config) {
				c.AnalyzeInterval = -time.Hour
			},
			expectError: true,
			errorMsg:    "analyzeInterval cannot be negative",
		},
		{
			name: "zero intervals should pass",
			modifier: func(c *Config) {
				c.VacuumInterval = 0
				c.AnalyzeInterval = 0
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on maintenance settings
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_DataRetentionSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "negative retention days should fail",
			modifier: func(c *Config) {
				c.RetentionDays = -30
			},
			expectError: true,
			errorMsg:    "retentionDays cannot be negative",
		},
		{
			name: "zero retention days should pass",
			modifier: func(c *Config) {
				c.RetentionDays = 0
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on data retention settings
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_BackupSettings(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "backup enabled with empty path should fail",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = ""
			},
			expectError: true,
			errorMsg:    "backupPath cannot be empty when backups are enabled",
		},
		{
			name: "backup enabled with zero interval should fail",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = tempDir
				c.BackupInterval = 0
			},
			expectError: true,
			errorMsg:    "backupInterval must be positive when backups are enabled",
		},
		{
			name: "backup enabled with negative interval should fail",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = tempDir
				c.BackupInterval = -time.Hour
			},
			expectError: true,
			errorMsg:    "backupInterval must be positive when backups are enabled",
		},
		{
			name: "backup enabled with zero retention should fail",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = tempDir
				c.BackupRetention = 0
			},
			expectError: true,
			errorMsg:    "backupRetention must be positive when backups are enabled",
		},
		{
			name: "backup enabled with negative retention should fail",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = tempDir
				c.BackupRetention = -5
			},
			expectError: true,
			errorMsg:    "backupRetention must be positive when backups are enabled",
		},
		{
			name: "backup disabled with invalid settings should pass",
			modifier: func(c *Config) {
				c.BackupEnabled = false
				c.BackupPath = ""
				c.BackupInterval = 0
				c.BackupRetention = 0
			},
			expectError: false,
		},
		{
			name: "backup enabled with valid settings should pass",
			modifier: func(c *Config) {
				c.BackupEnabled = true
				c.BackupPath = tempDir
				c.BackupInterval = time.Hour
				c.BackupRetention = 7
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on backup settings
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_EnvironmentAndLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid environment should fail",
			modifier: func(c *Config) {
				c.Environment = "invalid"
			},
			expectError: true,
			errorMsg:    "invalid environment: invalid",
		},
		{
			name: "valid environments should pass",
			modifier: func(c *Config) {
				c.Environment = "development"
			},
			expectError: false,
		},
		{
			name: "invalid log level should fail",
			modifier: func(c *Config) {
				c.LogLevel = "invalid"
			},
			expectError: true,
			errorMsg:    "invalid logLevel: invalid",
		},
		{
			name: "valid log levels should pass",
			modifier: func(c *Config) {
				c.LogLevel = "debug"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate to focus on environment and log level validation
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_DefaultConfigurations(t *testing.T) {
	t.Parallel()

	// Create a temporary migrations directory for testing
	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	tests := []struct {
		name     string
		configFn func() *Config
		setup    func(*Config)
	}{
		{
			name:     "default config should be valid",
			configFn: DefaultConfig,
			setup: func(c *Config) {
				c.MigrationsPath = migrationsDir // Use test migrations path
			},
		},
		{
			name:     "development config should be valid",
			configFn: DevelopmentConfig,
			setup: func(c *Config) {
				c.MigrationsPath = migrationsDir // Use test migrations path
			},
		},
		{
			name:     "test config should be valid",
			configFn: TestConfig,
			setup: func(c *Config) {
				c.MigrationsPath = migrationsDir // Use test migrations path
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := tt.configFn()
			tt.setup(config)
			err := config.Validate()
			if err != nil {
				t.Errorf("Configuration %s should be valid but got error: %v", tt.name, err)
			}
		})
	}
}

func TestConfig_Validate_ComplexScenarios(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	tests := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "multiple validation errors should return first error",
			modifier: func(c *Config) {
				c.Path = ""
				c.MaxConnections = -1
			},
			expectError: true,
			errorMsg:    "database path cannot be empty",
		},
		{
			name: "complex valid configuration",
			modifier: func(c *Config) {
				c.Path = "complex.db"
				c.MaxConnections = 20
				c.MaxIdleConns = 10
				c.ConnMaxLifetime = 2 * time.Hour
				c.ConnMaxIdleTime = time.Hour
				c.MigrationsPath = migrationsDir
				c.AutoMigrate = true
				c.JournalMode = "WAL"
				c.SynchronousMode = "NORMAL"
				c.CacheSize = 4000
				c.BusyTimeout = 10000
				c.ForeignKeys = true
				c.AutoVacuum = true
				c.VacuumInterval = 12 * time.Hour
				c.AnalyzeInterval = 3 * time.Hour
				c.RetentionDays = 30
				c.EnableCleanup = true
				c.BackupEnabled = true
				c.BackupInterval = 6 * time.Hour
				c.BackupPath = tempDir
				c.BackupRetention = 14
				c.Environment = "production"
				c.LogLevel = "info"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate for complex scenario testing
			tt.modifier(config)

			err := config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_ValidateEnvironmentOverrides(t *testing.T) {
	// Test that validation catches issues even when environment variables
	// could override config values (simulating the real-world scenario
	// CodeRabbit was concerned about)

	t.Run("Environment could set WAL on in-memory via config", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoMigrate = false // Disable AutoMigrate to focus on WAL validation
		config.Path = ":memory:"
		config.JournalMode = "WAL" // This could come from env var

		err := config.Validate()
		if err == nil {
			t.Error("Expected validation to catch WAL + in-memory incompatibility")
		}
		if !strings.Contains(err.Error(), "journalMode cannot be WAL when using in-memory database") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("Environment could set AutoMigrate with invalid path", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoMigrate = true
		config.MigrationsPath = "/absolutely/does/not/exist"

		err := config.Validate()
		if err == nil {
			t.Error("Expected validation to catch missing migrations path")
		}
		if !strings.Contains(err.Error(), "does not exist when AutoMigrate is enabled") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
}

func TestConfig_GetConnectionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modifier func(*Config)
		expected map[string]string // expected query parameters
		pathCheck func(string) bool // function to validate the path part
	}{
		{
			name: "basic configuration",
			modifier: func(c *Config) {
				c.Path = "test.db"
				c.ForeignKeys = true
				c.JournalMode = "WAL"
				c.SynchronousMode = "NORMAL"
				c.CacheSize = 2000
				c.BusyTimeout = 30000
			},
			expected: map[string]string{
				"_foreign_keys":  "on",
				"_journal_mode":  "WAL",
				"_synchronous":   "NORMAL",
				"_cache_size":    "-2000",
				"_busy_timeout":  "30000",
			},
			pathCheck: func(s string) bool {
				return strings.HasPrefix(s, "test.db?")
			},
		},
		{
			name: "in-memory database",
			modifier: func(c *Config) {
				c.Path = ":memory:"
				c.ForeignKeys = false
				c.JournalMode = "MEMORY"
				c.SynchronousMode = "OFF"
				c.CacheSize = 1000
				c.BusyTimeout = 0
			},
			expected: map[string]string{
				"_foreign_keys":  "off",
				"_journal_mode":  "MEMORY",
				"_synchronous":   "OFF",
				"_cache_size":    "-1000",
				"_busy_timeout":  "0",
			},
			pathCheck: func(s string) bool {
				return strings.HasPrefix(s, ":memory:?")
			},
		},
		{
			name: "path with special characters",
			modifier: func(c *Config) {
				c.Path = "my database?.db&test=1"
				c.ForeignKeys = true
				c.JournalMode = "WAL"
				c.SynchronousMode = "FULL"
				c.CacheSize = 500
				c.BusyTimeout = 5000
			},
			expected: map[string]string{
				"_foreign_keys":  "on",
				"_journal_mode":  "WAL",
				"_synchronous":   "FULL",
				"_cache_size":    "-500",
				"_busy_timeout":  "5000",
			},
			pathCheck: func(s string) bool {
				// Only ? and & should be escaped to prevent query parsing issues
				return strings.HasPrefix(s, "my database%3F.db%26test=1?")
			},
		},
		{
			name: "URL-style path",
			modifier: func(c *Config) {
				c.Path = "file:///path/to/database.db"
				c.ForeignKeys = false
				c.JournalMode = "DELETE"
				c.SynchronousMode = "NORMAL"
				c.CacheSize = 1500
				c.BusyTimeout = 10000
			},
			expected: map[string]string{
				"_foreign_keys":  "off",
				"_journal_mode":  "DELETE",
				"_synchronous":   "NORMAL",
				"_cache_size":    "-1500",
				"_busy_timeout":  "10000",
			},
			pathCheck: func(s string) bool {
				return strings.HasPrefix(s, "file:///path/to/database.db?")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := DefaultConfig()
			config.AutoMigrate = false // Disable AutoMigrate for connection string testing
			tt.modifier(config)

			connStr := config.GetConnectionString()

			// Validate the path format
			if !tt.pathCheck(connStr) {
				t.Errorf("Connection string path format check failed: %s", connStr)
			}

			// Parse query parameters from the connection string
			var values url.Values
			var err error

			// Try to parse as URL first
			if u, parseErr := url.Parse(connStr); parseErr == nil && len(u.Query()) > 0 {
				// Successfully parsed as URL with query parameters
				values = u.Query()
			} else {
				// For opaque URLs or special formats, manually extract query parameters
				if strings.Contains(connStr, "?") {
					parts := strings.SplitN(connStr, "?", 2)
					if len(parts) == 2 {
						values, err = url.ParseQuery(parts[1])
						if err != nil {
							t.Fatalf("Failed to parse query parameters: %v", err)
						}
					} else {
						values = url.Values{}
					}
				} else {
					values = url.Values{}
				}
			}
			for key, expectedValue := range tt.expected {
				actualValue := values.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}

			// Check that no extra parameters are present
			for key := range values {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("Unexpected parameter in connection string: %s=%s", key, values.Get(key))
				}
			}
		})
	}
}

func TestConfig_GetConnectionString_SpecialCases(t *testing.T) {
	t.Parallel()

	t.Run("Empty path should not cause panic", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoMigrate = false
		config.Path = ""

		// Should not panic
		connStr := config.GetConnectionString()
		if connStr == "" {
			t.Error("Connection string should not be empty even with empty path")
		}
	})

	t.Run("Malformed URL path should be handled gracefully", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoMigrate = false
		config.Path = "://invalid-url"

		// Should not panic and should produce some output
		connStr := config.GetConnectionString()
		if connStr == "" {
			t.Error("Connection string should not be empty even with malformed URL")
		}
	})

	t.Run("Parameters should be properly URL encoded", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoMigrate = false
		config.Path = "test.db"
		config.JournalMode = "WAL MODE" // Space should be encoded
		config.SynchronousMode = "FULL&EXTRA" // & should be encoded

		connStr := config.GetConnectionString()

		u, err := url.Parse(connStr)
		if err != nil {
			t.Fatalf("Failed to parse connection string: %v", err)
		}

		values := u.Query()
		if values.Get("_journal_mode") != "WAL MODE" {
			t.Errorf("Journal mode not properly decoded: %s", values.Get("_journal_mode"))
		}
		if values.Get("_synchronous") != "FULL&EXTRA" {
			t.Errorf("Synchronous mode not properly decoded: %s", values.Get("_synchronous"))
		}
	})
}
