-- name: GetSetting :one
SELECT value FROM settings
WHERE key = ?
LIMIT 1;

-- name: UpdateSetting :exec
UPDATE settings
SET value = ?, updated_at = CURRENT_TIMESTAMP
WHERE key = ?;

-- name: CreateSetting :exec
INSERT INTO settings (key, value)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP;