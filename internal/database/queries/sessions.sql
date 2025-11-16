-- name: CreateSession :exec
INSERT INTO sessions (token, username, expires_at)
VALUES (?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions
WHERE token = ? AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: DeleteUserSessions :exec
DELETE FROM sessions
WHERE username = ?;