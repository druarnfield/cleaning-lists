-- name: CreateTask :one
INSERT INTO tasks (name, category, frequency, estimated_mins, default_assignee)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY category, name;

-- name: ListTasksByCategory :many
SELECT * FROM tasks
WHERE category = ?
ORDER BY name;

-- name: UpdateTask :one
UPDATE tasks
SET name = ?, category = ?, frequency = ?, estimated_mins = ?, default_assignee = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = ?;

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks;

-- name: GetCategories :many
SELECT DISTINCT category FROM tasks
ORDER BY category;