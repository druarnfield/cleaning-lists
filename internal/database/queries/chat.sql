-- name: CreateScheduleSuggestion :one
INSERT INTO schedule_suggestions (user_message, llm_response, week_offset, changes_made)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetRecentSuggestions :many
SELECT * FROM schedule_suggestions
ORDER BY created_at DESC
LIMIT ?;

-- name: GetSuggestionsForWeek :many
SELECT * FROM schedule_suggestions
WHERE week_offset = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: DeleteOldSuggestions :exec
DELETE FROM schedule_suggestions
WHERE created_at < datetime('now', '-' || ? || ' days');
