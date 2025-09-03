-- App Usage Queries
-- These queries handle CRUD operations for individual application usage data

-- name: CreateAppUsage :one
INSERT INTO app_usage (name, duration, icon_path, exe_path, date)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetAppUsageByID :one
SELECT * FROM app_usage
WHERE id = ?;

-- name: GetAppUsageByNameAndDate :one
SELECT * FROM app_usage
WHERE name = ? AND date = ?;

-- name: GetAppUsageByDate :many
SELECT * FROM app_usage
WHERE date = ?
ORDER BY duration DESC, name ASC;

-- name: UpdateAppUsage :one
UPDATE app_usage
SET duration = ?, icon_path = ?, exe_path = ?, updated_at = CURRENT_TIMESTAMP
WHERE name = ? AND date = ?
RETURNING *;

-- name: UpsertAppUsage :one
INSERT INTO app_usage (name, duration, icon_path, exe_path, date)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(name, date) DO UPDATE SET
    duration = excluded.duration,
    icon_path = excluded.icon_path,
    exe_path = excluded.exe_path,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteAppUsage :exec
DELETE FROM app_usage
WHERE name = ? AND date = ?;

-- name: DeleteAppUsageByID :exec
DELETE FROM app_usage
WHERE id = ?;

-- Historical data retrieval queries
-- name: GetAppUsageByDateRange :many
SELECT * FROM app_usage
WHERE date >= ? AND date <= ?
ORDER BY date DESC, duration DESC;

-- name: GetAppUsageByNameAndDateRange :many
SELECT * FROM app_usage
WHERE name = ? AND date >= ? AND date <= ?
ORDER BY date DESC;

-- name: GetTopAppsByDate :many
SELECT * FROM app_usage
WHERE date = ?
ORDER BY duration DESC
LIMIT ?;

-- name: GetTopAppsByDateRange :many
SELECT name, SUM(duration) as total_duration, COUNT(DISTINCT date) as days_used
FROM app_usage
WHERE date >= ? AND date <= ?
GROUP BY name
ORDER BY total_duration DESC
LIMIT ?;

-- name: GetRecentAppUsage :many
SELECT * FROM app_usage
WHERE date >= date('now', '-' || sqlc.arg(days) || ' days')
ORDER BY date DESC, duration DESC;

-- Single row insert operation
-- name: InsertAppUsage :exec
INSERT INTO app_usage (name, duration, icon_path, exe_path, date)
VALUES (?, ?, ?, ?, ?);

-- name: BatchUpdateAppUsage :exec
UPDATE app_usage
SET duration = duration + ?, updated_at = CURRENT_TIMESTAMP
WHERE name = ? AND date = ?;

-- name: GetAllAppsForDate :many
SELECT DISTINCT name FROM app_usage
WHERE date = ?
ORDER BY name;

-- name: GetAppUsageHistory :many
SELECT name, date, duration, icon_path, exe_path
FROM app_usage
WHERE name = ?
ORDER BY date DESC
LIMIT ?;

-- name: DeleteOldAppUsage :exec
DELETE FROM app_usage
WHERE date < ?;

-- name: GetUsageStatsByDateRange :many
SELECT 
    date,
    COUNT(*) as app_count,
    SUM(duration) as total_duration,
    AVG(duration) as avg_duration,
    MAX(duration) as max_duration
FROM app_usage
WHERE date >= ? AND date <= ?
GROUP BY date
ORDER BY date DESC;

-- name: GetAppUsageByDateRangePaginated :many
SELECT * FROM app_usage
WHERE date >= ? AND date <= ?
ORDER BY date DESC, duration DESC
LIMIT ? OFFSET ?;

-- name: GetAppUsageCountByDateRange :one
SELECT COUNT(*) FROM app_usage
WHERE date >= ? AND date <= ?;