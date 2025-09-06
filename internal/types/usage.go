package types

import "time"

// AppUsage represents usage data for a single application
type AppUsage struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Duration  int64     `json:"duration" db:"duration"` // in seconds
	IconPath  string    `json:"iconPath" db:"icon_path"`
	ExePath   string    `json:"exePath" db:"exe_path"`
	Date      time.Time `json:"date" db:"date"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// UsageData represents the complete usage data
type UsageData struct {
	TotalTime int64      `json:"totalTime"` // in seconds
	Apps      []AppUsage `json:"apps"`
}

// DailyUsage represents daily usage summary
type DailyUsage struct {
	ID        int64     `json:"id" db:"id"`
	Date      time.Time `json:"date" db:"date"`
	TotalTime int64     `json:"totalTime" db:"total_time"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// PaginatedAppUsageResult represents paginated app usage results with metadata
type PaginatedAppUsageResult struct {
	Results []AppUsage `json:"results"`
	Total   int        `json:"total"`
}

// BatchStrategy defines the strategy for batch operations
type BatchStrategy int

const (
	// BatchStrategyInsertOnly performs insert-only operations, failing on conflicts
	BatchStrategyInsertOnly BatchStrategy = iota
	// BatchStrategyUpsert performs upsert operations, updating on conflicts
	BatchStrategyUpsert
)
