-- name: CreateCompletion :one
INSERT INTO completions (task_instance_id, completed_by, actual_mins)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetCompletion :one
SELECT * FROM completions
WHERE task_instance_id = ?
LIMIT 1;

-- name: DeleteCompletion :exec
DELETE FROM completions
WHERE task_instance_id = ?;

-- name: GetCompletionByInstance :one
SELECT * FROM completions
WHERE task_instance_id = ?
LIMIT 1;

-- name: DeleteOldCompletions :exec
DELETE FROM completions
WHERE completed_at < date('now', '-' || ? || ' days');

-- name: GetCompletionStats :many
SELECT
    ti.assigned_to,
    COUNT(ti.id) as scheduled_count,
    SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END) as completed_count,
    SUM(CASE WHEN ti.counts_toward_weekly THEN t.estimated_mins ELSE 0 END) as scheduled_mins,
    SUM(CASE WHEN c.id IS NOT NULL AND ti.counts_toward_weekly THEN t.estimated_mins ELSE 0 END) as completed_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
LEFT JOIN completions c ON c.task_instance_id = ti.id
WHERE ti.scheduled_date >= ? AND ti.scheduled_date <= ?
GROUP BY ti.assigned_to;

-- name: GetCategoryStats :many
SELECT
    t.category,
    COUNT(ti.id) as scheduled_count,
    SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END) as completed_count
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
LEFT JOIN completions c ON c.task_instance_id = ti.id
WHERE ti.scheduled_date >= ? AND ti.scheduled_date <= ?
GROUP BY t.category
ORDER BY t.category;

-- name: GetWeeklyCompletionStats :many
SELECT week_start, total_count, completed_count
FROM (
    SELECT
        ti.week_start_date as week_start,
        COUNT(ti.id) as total_count,
        SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END) as completed_count
    FROM task_instances ti
    LEFT JOIN completions c ON c.task_instance_id = ti.id
    WHERE ti.week_start_date <= date('now')
    GROUP BY ti.week_start_date
    ORDER BY ti.week_start_date DESC
    LIMIT 4
)
ORDER BY week_start ASC;

-- name: ListCompletionsByDateRange :many
SELECT c.*, ti.assigned_to, t.name
FROM completions c
JOIN task_instances ti ON c.task_instance_id = ti.id
JOIN tasks t ON ti.task_id = t.id
WHERE ti.scheduled_date >= ? AND ti.scheduled_date <= ?
ORDER BY c.completed_at DESC;