-- +goose Up
-- Create daily_usage table for storing daily screen time summaries
CREATE TABLE daily_usage (
    id INTEGER PRIMARY KEY, -- Uses rowid-backed primary key for better performance
    date DATE NOT NULL UNIQUE,
    total_time INTEGER NOT NULL DEFAULT 0 CHECK (total_time >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- Drop the daily_usage table
DROP TABLE IF EXISTS daily_usage;