package database

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// parseBoolEnv reads an environment variable and parses it as a boolean.
// Returns the parsed value and a boolean indicating if the variable was present.
// Supports common boolean representations: true/false, 1/0, yes/no, on/off, t/f, y/n (case-insensitive).
func parseBoolEnv(key string) (bool, bool) {
	value := os.Getenv(key)
	if value == "" {
		return false, false
	}

	// First try strconv.ParseBool which handles: true/false, 1/0, t/f (case-insensitive)
	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed, true
	}

	// Handle additional common variants not supported by strconv.ParseBool
	switch value {
	case "yes", "YES", "Yes", "y", "Y", "on", "ON", "On":
		return true, true
	case "no", "NO", "No", "n", "N", "off", "OFF", "Off":
		return false, true
	default:
		return false, false
	}
}

// Config holds all database configuration options
type Config struct {
	// Database connection settings
	Path                  string        `json:"path" yaml:"path"`                                   // Database file path
	MaxConnections        int           `json:"maxConnections" yaml:"maxConnections"`               // Maximum number of open connections
	MaxIdleConns          int           `json:"maxIdleConns" yaml:"maxIdleConns"`                   // Maximum number of idle connections
	ConnMaxLifetime       time.Duration `json:"connMaxLifetime" yaml:"connMaxLifetime"`             // Maximum connection lifetime
	ConnMaxIdleTime       time.Duration `json:"connMaxIdleTime" yaml:"connMaxIdleTime"`             // Maximum connection idle time
	ForceSingleConnection bool          `json:"forceSingleConnection" yaml:"forceSingleConnection"` // Force single connection mode for SQLite

	// Migration settings
	MigrationsPath string `json:"migrationsPath" yaml:"migrationsPath"` // Path to migration files
	AutoMigrate    bool   `json:"autoMigrate" yaml:"autoMigrate"`       // Whether to run migrations automatically on startup

	// Performance settings
	JournalMode     string `json:"journalMode" yaml:"journalMode"`         // SQLite journal mode (WAL, DELETE, etc.)
	SynchronousMode string `json:"synchronousMode" yaml:"synchronousMode"` // SQLite synchronous mode (FULL, NORMAL, OFF)
	CacheSize       int    `json:"cacheSize" yaml:"cacheSize"`             // SQLite cache size in KB
	BusyTimeout     int    `json:"busyTimeout" yaml:"busyTimeout"`         // SQLite busy timeout in milliseconds
	ForeignKeys     bool   `json:"foreignKeys" yaml:"foreignKeys"`         // Enable foreign key constraints

	// Maintenance settings
	AutoVacuum      bool          `json:"autoVacuum" yaml:"autoVacuum"`           // Enable auto vacuum
	VacuumInterval  time.Duration `json:"vacuumInterval" yaml:"vacuumInterval"`   // Interval for running VACUUM
	AnalyzeInterval time.Duration `json:"analyzeInterval" yaml:"analyzeInterval"` // Interval for running ANALYZE

	// Data retention settings
	RetentionDays int  `json:"retentionDays" yaml:"retentionDays"` // Number of days to retain data (0 = no cleanup)
	EnableCleanup bool `json:"enableCleanup" yaml:"enableCleanup"` // Whether to enable automatic data cleanup

	// Backup settings
	BackupEnabled   bool          `json:"backupEnabled" yaml:"backupEnabled"`     // Enable automatic backups
	BackupInterval  time.Duration `json:"backupInterval" yaml:"backupInterval"`   // Backup interval
	BackupPath      string        `json:"backupPath" yaml:"backupPath"`           // Backup directory path
	BackupRetention int           `json:"backupRetention" yaml:"backupRetention"` // Number of backups to retain

	// Environment and runtime settings
	Environment string `json:"environment" yaml:"environment"` // Environment (development, production, test)
	LogLevel    string `json:"logLevel" yaml:"logLevel"`       // Log level for database operations
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Connection settings
		Path:                  "qwin.db",
		MaxConnections:        10,
		MaxIdleConns:          5,
		ConnMaxLifetime:       24 * time.Hour,
		ConnMaxIdleTime:       30 * time.Minute,
		ForceSingleConnection: false, // Let the service auto-detect based on journal mode

		// Migration settings
		MigrationsPath: "internal/database/migrations",
		AutoMigrate:    true,

		// Performance settings
		JournalMode:     "WAL",
		SynchronousMode: "NORMAL",
		CacheSize:       2000,  // 2MB cache
		BusyTimeout:     30000, // 30 seconds
		ForeignKeys:     true,

		// Maintenance settings
		AutoVacuum:      true,
		VacuumInterval:  24 * time.Hour, // Daily vacuum
		AnalyzeInterval: 6 * time.Hour,  // Analyze every 6 hours

		// Data retention settings
		RetentionDays: 365, // Keep data for 1 year
		EnableCleanup: true,

		// Backup settings
		BackupEnabled:   false, // Disabled by default
		BackupInterval:  24 * time.Hour,
		BackupPath:      "backups",
		BackupRetention: 7, // Keep 7 backups

		// Environment settings
		Environment: "production",
		LogLevel:    "info",
	}
}

// DevelopmentConfig returns a configuration optimized for development
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Path = "qwin_dev.db"
	config.Environment = "development"
	config.LogLevel = "debug"
	config.RetentionDays = 30    // Keep less data in development
	config.EnableCleanup = false // Don't cleanup in development
	config.BackupEnabled = false // No backups in development
	return config
}

// TestConfig returns a configuration optimized for testing
func TestConfig() *Config {
	config := DefaultConfig()
	config.Path = ":memory:" // Use in-memory database for tests
	config.Environment = "test"
	config.LogLevel = "error"
	config.AutoMigrate = true
	config.RetentionDays = 0 // No retention in tests
	config.EnableCleanup = false
	config.BackupEnabled = false
	config.VacuumInterval = 0 // Disable maintenance in tests
	config.AnalyzeInterval = 0

	// Configure in-memory-friendly pragmas
	config.JournalMode = "MEMORY"  // WAL is meaningless for in-memory databases
	config.SynchronousMode = "OFF" // No need for synchronous writes in memory
	config.CacheSize = 1000        // Smaller cache for tests
	config.BusyTimeout = 1000      // Shorter timeout for tests

	return config
}

// LoadFromEnvironment loads configuration from environment variables
func (c *Config) LoadFromEnvironment() error {
	// Database path
	if path := os.Getenv("QWIN_DB_PATH"); path != "" {
		c.Path = path
	}

	// Connection settings
	if maxConns := os.Getenv("QWIN_DB_MAX_CONNECTIONS"); maxConns != "" {
		if val, err := strconv.Atoi(maxConns); err == nil && val > 0 {
			c.MaxConnections = val
		}
	}

	if maxIdle := os.Getenv("QWIN_DB_MAX_IDLE_CONNECTIONS"); maxIdle != "" {
		if val, err := strconv.Atoi(maxIdle); err == nil && val > 0 {
			c.MaxIdleConns = val
		}
	}

	if lifetime := os.Getenv("QWIN_DB_CONN_MAX_LIFETIME"); lifetime != "" {
		if val, err := time.ParseDuration(lifetime); err == nil {
			c.ConnMaxLifetime = val
		}
	}

	if idleTime := os.Getenv("QWIN_DB_CONN_MAX_IDLE_TIME"); idleTime != "" {
		if val, err := time.ParseDuration(idleTime); err == nil {
			c.ConnMaxIdleTime = val
		}
	}

	// Migration settings
	if migrationsPath := os.Getenv("QWIN_DB_MIGRATIONS_PATH"); migrationsPath != "" {
		c.MigrationsPath = migrationsPath
	}

	if autoMigrate, present := parseBoolEnv("QWIN_DB_AUTO_MIGRATE"); present {
		c.AutoMigrate = autoMigrate
	}

	// Performance settings
	if journalMode := os.Getenv("QWIN_DB_JOURNAL_MODE"); journalMode != "" {
		c.JournalMode = journalMode
	}

	if syncMode := os.Getenv("QWIN_DB_SYNCHRONOUS_MODE"); syncMode != "" {
		c.SynchronousMode = syncMode
	}

	if cacheSize := os.Getenv("QWIN_DB_CACHE_SIZE"); cacheSize != "" {
		if val, err := strconv.Atoi(cacheSize); err == nil && val > 0 {
			c.CacheSize = val
		}
	}

	if busyTimeout := os.Getenv("QWIN_DB_BUSY_TIMEOUT"); busyTimeout != "" {
		if val, err := strconv.Atoi(busyTimeout); err == nil && val >= 0 {
			c.BusyTimeout = val
		}
	}

	if foreignKeys, present := parseBoolEnv("QWIN_DB_FOREIGN_KEYS"); present {
		c.ForeignKeys = foreignKeys
	}

	if forceSingle, present := parseBoolEnv("QWIN_DB_FORCE_SINGLE_CONNECTION"); present {
		c.ForceSingleConnection = forceSingle
	}

	// Maintenance settings
	if autoVacuum, present := parseBoolEnv("QWIN_DB_AUTO_VACUUM"); present {
		c.AutoVacuum = autoVacuum
	}

	if vacuumInterval := os.Getenv("QWIN_DB_VACUUM_INTERVAL"); vacuumInterval != "" {
		if val, err := time.ParseDuration(vacuumInterval); err == nil {
			c.VacuumInterval = val
		}
	}

	if analyzeInterval := os.Getenv("QWIN_DB_ANALYZE_INTERVAL"); analyzeInterval != "" {
		if val, err := time.ParseDuration(analyzeInterval); err == nil {
			c.AnalyzeInterval = val
		}
	}

	// Data retention settings
	if retentionDays := os.Getenv("QWIN_DB_RETENTION_DAYS"); retentionDays != "" {
		if val, err := strconv.Atoi(retentionDays); err == nil && val >= 0 {
			c.RetentionDays = val
		}
	}

	if enableCleanup, present := parseBoolEnv("QWIN_DB_ENABLE_CLEANUP"); present {
		c.EnableCleanup = enableCleanup
	}

	// Backup settings
	if backupEnabled, present := parseBoolEnv("QWIN_DB_BACKUP_ENABLED"); present {
		c.BackupEnabled = backupEnabled
	}

	if backupInterval := os.Getenv("QWIN_DB_BACKUP_INTERVAL"); backupInterval != "" {
		if val, err := time.ParseDuration(backupInterval); err == nil {
			c.BackupInterval = val
		}
	}

	if backupPath := os.Getenv("QWIN_DB_BACKUP_PATH"); backupPath != "" {
		c.BackupPath = backupPath
	}

	if backupRetention := os.Getenv("QWIN_DB_BACKUP_RETENTION"); backupRetention != "" {
		if val, err := strconv.Atoi(backupRetention); err == nil && val > 0 {
			c.BackupRetention = val
		}
	}

	// Environment settings
	if environment := os.Getenv("QWIN_ENVIRONMENT"); environment != "" {
		c.Environment = environment
	}

	if logLevel := os.Getenv("QWIN_DB_LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}

	return nil
}

// Validate validates the configuration parameters
func (c *Config) Validate() error {
	// Validate database path
	if c.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// For file-based databases, ensure directory exists
	if c.Path != ":memory:" {
		dir := filepath.Dir(c.Path)
		if dir != "." && dir != "" {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create database directory %s: %w", dir, err)
				}
			}
		}
	}

	// Validate connection settings
	if c.MaxConnections <= 0 {
		return fmt.Errorf("maxConnections must be positive, got %d", c.MaxConnections)
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("maxIdleConns cannot be negative, got %d", c.MaxIdleConns)
	}

	if c.MaxIdleConns > c.MaxConnections {
		return fmt.Errorf("maxIdleConns (%d) cannot be greater than maxConnections (%d)", c.MaxIdleConns, c.MaxConnections)
	}

	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("connMaxLifetime cannot be negative, got %v", c.ConnMaxLifetime)
	}

	if c.ConnMaxIdleTime < 0 {
		return fmt.Errorf("connMaxIdleTime cannot be negative, got %v", c.ConnMaxIdleTime)
	}

	// Validate migrations path
	if c.MigrationsPath == "" {
		return fmt.Errorf("migrationsPath cannot be empty")
	}

	// If AutoMigrate is enabled, ensure migrations path exists and is accessible
	if c.AutoMigrate {
		if _, err := os.Stat(c.MigrationsPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("migrationsPath %q does not exist when AutoMigrate is enabled", c.MigrationsPath)
			}
			return fmt.Errorf("migrationsPath %q is not accessible when AutoMigrate is enabled: %w", c.MigrationsPath, err)
		}
	}

	// Validate performance settings
	validJournalModes := map[string]bool{
		"DELETE":   true,
		"TRUNCATE": true,
		"PERSIST":  true,
		"MEMORY":   true,
		"WAL":      true,
		"OFF":      true,
	}
	// Check case-insensitive
	journalModeValid := false
	for validMode := range validJournalModes {
		if strings.EqualFold(c.JournalMode, validMode) {
			journalModeValid = true
			break
		}
	}
	if !journalModeValid {
		return fmt.Errorf("invalid journalMode: %s", c.JournalMode)
	}

	if c.IsInMemory() && strings.EqualFold(c.JournalMode, "WAL") {
		return fmt.Errorf("journalMode cannot be WAL when using in-memory database")
	}

	validSyncModes := map[string]bool{
		"OFF":    true,
		"NORMAL": true,
		"FULL":   true,
		"EXTRA":  true,
	}
	if !validSyncModes[c.SynchronousMode] {
		return fmt.Errorf("invalid synchronousMode: %s", c.SynchronousMode)
	}

	if c.CacheSize <= 0 {
		return fmt.Errorf("cacheSize must be positive, got %d", c.CacheSize)
	}

	if c.BusyTimeout < 0 {
		return fmt.Errorf("busyTimeout cannot be negative, got %d", c.BusyTimeout)
	}

	// Validate maintenance settings
	if c.VacuumInterval < 0 {
		return fmt.Errorf("vacuumInterval cannot be negative, got %v", c.VacuumInterval)
	}

	if c.AnalyzeInterval < 0 {
		return fmt.Errorf("analyzeInterval cannot be negative, got %v", c.AnalyzeInterval)
	}

	// Validate data retention settings
	if c.RetentionDays < 0 {
		return fmt.Errorf("retentionDays cannot be negative, got %d", c.RetentionDays)
	}

	// Validate backup settings
	if c.BackupEnabled {
		if c.BackupPath == "" {
			return fmt.Errorf("backupPath cannot be empty when backups are enabled")
		}

		if c.BackupInterval <= 0 {
			return fmt.Errorf("backupInterval must be positive when backups are enabled, got %v", c.BackupInterval)
		}

		if c.BackupRetention <= 0 {
			return fmt.Errorf("backupRetention must be positive when backups are enabled, got %d", c.BackupRetention)
		}

		// Ensure backup directory exists
		if _, err := os.Stat(c.BackupPath); os.IsNotExist(err) {
			if err := os.MkdirAll(c.BackupPath, 0755); err != nil {
				return fmt.Errorf("failed to create backup directory %s: %w", c.BackupPath, err)
			}
		}
	}

	// Validate environment
	validEnvironments := map[string]bool{
		"development": true,
		"test":        true,
		"production":  true,
	}
	if !validEnvironments[c.Environment] {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid logLevel: %s", c.LogLevel)
	}

	return nil
}

// GetConnectionString builds the SQLite connection string with all options
// Uses net/url for proper URL encoding of query parameters only
func (c *Config) GetConnectionString() string {
	// Create URL values for SQLite parameters
	values := url.Values{}

	// Add foreign keys setting
	if c.ForeignKeys {
		values.Set("_foreign_keys", "on")
	} else {
		values.Set("_foreign_keys", "off")
	}

	// Add journal mode
	values.Set("_journal_mode", c.JournalMode)

	// Add synchronous mode
	values.Set("_synchronous", c.SynchronousMode)

	// Add cache size (pass negative value so SQLite interprets it as KB)
	values.Set("_cache_size", fmt.Sprintf("%d", -c.CacheSize))

	// Add busy timeout
	values.Set("_busy_timeout", fmt.Sprintf("%d", c.BusyTimeout))

	// Build connection string: path + "?" + encoded query parameters
	// We need to escape ONLY the characters that would break query string parsing
	path := c.Path
	if strings.ContainsAny(path, "?&") {
		// Escape only the problematic characters that would break URL parsing
		path = strings.ReplaceAll(path, "?", "%3F")
		path = strings.ReplaceAll(path, "&", "%26")
	}
	
	return path + "?" + values.Encode()
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	return &Config{
		Path:                  c.Path,
		MaxConnections:        c.MaxConnections,
		MaxIdleConns:          c.MaxIdleConns,
		ConnMaxLifetime:       c.ConnMaxLifetime,
		ConnMaxIdleTime:       c.ConnMaxIdleTime,
		ForceSingleConnection: c.ForceSingleConnection,
		MigrationsPath:        c.MigrationsPath,
		AutoMigrate:           c.AutoMigrate,
		JournalMode:           c.JournalMode,
		SynchronousMode:       c.SynchronousMode,
		CacheSize:             c.CacheSize,
		BusyTimeout:           c.BusyTimeout,
		ForeignKeys:           c.ForeignKeys,
		AutoVacuum:            c.AutoVacuum,
		VacuumInterval:        c.VacuumInterval,
		AnalyzeInterval:       c.AnalyzeInterval,
		RetentionDays:         c.RetentionDays,
		EnableCleanup:         c.EnableCleanup,
		BackupEnabled:         c.BackupEnabled,
		BackupInterval:        c.BackupInterval,
		BackupPath:            c.BackupPath,
		BackupRetention:       c.BackupRetention,
		Environment:           c.Environment,
		LogLevel:              c.LogLevel,
	}
}

// IsInMemory returns true if the database is configured to use in-memory storage
func (c *Config) IsInMemory() bool {
	return c.Path == ":memory:"
}

// IsDevelopment returns true if the environment is set to development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTest returns true if the environment is set to test
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}

// IsProduction returns true if the environment is set to production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// ConfigForEnvironment returns a configuration optimized for the given environment
func ConfigForEnvironment(env string) *Config {
	switch env {
	case "development":
		return DevelopmentConfig()
	case "test":
		return TestConfig()
	default:
		config := DefaultConfig()
		// For production, use a path in current directory
		config.Path = filepath.Join(".", "qwin.db")
		return config
	}
}
