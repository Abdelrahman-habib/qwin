-- Daily Usage Queries
-- These queries handle CRUD operations for daily usage summaries

-- name: CreateDailyUsage :one
INSERT INTO daily_usage (date, total_time)
VALUES (?, ?)
RETURNING *;

-- name: GetDailyUsageByDate :one
SELECT * FROM daily_usage
WHERE date = ?;

-- name: UpdateDailyUsage :one
UPDATE daily_usage
SET total_time = ?, updated_at = CURRENT_TIMESTAMP
WHERE date = ?
RETURNING *;

-- name: UpsertDailyUsage :one
INSERT INTO daily_usage (date, total_time)
VALUES (?, ?)
ON CONFLICT(date) DO UPDATE SET
    total_time = excluded.total_time,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteDailyUsage :exec
DELETE FROM daily_usage
WHERE date = ?;

-- name: GetDailyUsageByDateRange :many
SELECT * FROM daily_usage
WHERE date >= ? AND date <= ?
ORDER BY date ASC;

-- name: GetRecentDailyUsage :many
SELECT * FROM daily_usage
WHERE date >= date('now', '-' || sqlc.arg(days) || ' days')
ORDER BY date DESC;

-- name: GetAllDailyUsage :many
SELECT * FROM daily_usage
ORDER BY date DESC;

-- name: DeleteOldDailyUsage :exec
DELETE FROM daily_usage
WHERE date < ?;