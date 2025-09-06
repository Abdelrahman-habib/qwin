-- +goose Up
-- Create app_usage table for storing individual application usage data
CREATE TABLE app_usage (
    id INTEGER PRIMARY KEY, -- Uses rowid-backed primary key for better performance
    name TEXT NOT NULL,
    duration INTEGER NOT NULL DEFAULT 0 CHECK (duration >= 0),
    icon_path TEXT,
    exe_path TEXT,
    date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance optimization
CREATE INDEX idx_app_usage_date ON app_usage(date);

-- Create unique constraint for data integrity (one record per app per day)
CREATE UNIQUE INDEX idx_app_usage_unique ON app_usage(name, date);

-- +goose Down
-- Drop the app_usage table and its indexes
DROP INDEX IF EXISTS idx_app_usage_unique;
DROP INDEX IF EXISTS idx_app_usage_date;
DROP TABLE IF EXISTS app_usage;