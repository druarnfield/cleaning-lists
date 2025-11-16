-- name: GetUser :one
SELECT * FROM users
WHERE username = ? LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?, is_password_set = TRUE, updated_at = CURRENT_TIMESTAMP
WHERE username = ?;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY username;

-- name: ResetUserPassword :exec
UPDATE users
SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
WHERE username = ?;