-- name: CreateInstance :one
INSERT INTO task_instances (
    task_id, scheduled_date, original_scheduled_date, assigned_to,
    week_start_date, brought_forward, brought_forward_by, counts_toward_weekly
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetInstance :one
SELECT ti.*, t.name, t.category, t.estimated_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.id = ?
LIMIT 1;

-- name: GetInstanceWithTask :one
SELECT ti.*, t.name as task_name, t.category as task_category, t.estimated_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.id = ?
LIMIT 1;

-- name: ListInstancesByWeek :many
SELECT ti.*, t.name as task_name, t.category as task_category, t.estimated_mins,
       c.id as completion_id, c.completed_by, c.completed_at
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
LEFT JOIN completions c ON c.task_instance_id = ti.id
WHERE ti.week_start_date = ?
ORDER BY ti.assigned_to, ti.scheduled_date, t.name;

-- name: ListInstancesByDateRange :many
SELECT ti.*, t.name, t.category, t.estimated_mins,
       c.id as completion_id, c.completed_by, c.completed_at
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
LEFT JOIN completions c ON c.task_instance_id = ti.id
WHERE ti.scheduled_date >= ? AND ti.scheduled_date <= ?
ORDER BY ti.scheduled_date, ti.assigned_to, t.name;

-- name: ListFutureInstances :many
SELECT ti.*, t.name, t.category, t.estimated_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.scheduled_date > ?
  AND ti.brought_forward = FALSE
  AND (ti.assigned_to = ? OR t.default_assignee = 'Both')
ORDER BY ti.scheduled_date, t.name;

-- name: CountInstances :one
SELECT COUNT(*) FROM task_instances;

-- name: DeleteOldInstances :exec
DELETE FROM task_instances
WHERE scheduled_date < date('now', '-90 days');

-- name: GetNextInstanceForTask :one
SELECT ti.*, t.name, t.category, t.estimated_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.task_id = ?
  AND ti.scheduled_date > ?
  AND ti.brought_forward = FALSE
ORDER BY ti.scheduled_date
LIMIT 1;

-- name: CheckDuplicateInstance :one
SELECT COUNT(*) FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.task_id = ?
  AND ti.week_start_date = ?
  AND ti.brought_forward = FALSE
  AND (
    -- For 'Both' tasks, check assigned_to matches
    (LOWER(t.default_assignee) = 'both' AND ti.assigned_to = ?)
    OR
    -- For other tasks, just check task and week (ignore assigned_to)
    (LOWER(t.default_assignee) != 'both' OR t.default_assignee IS NULL)
  );

-- name: ListTaskInstances :many
SELECT * FROM task_instances
ORDER BY week_start_date, scheduled_date;

-- name: UpdateTaskInstanceAssignment :exec
UPDATE task_instances
SET assigned_to = ?
WHERE id = ?;

-- name: DeleteInstancesInDateRange :exec
DELETE FROM task_instances
WHERE week_start_date >= ? AND week_start_date <= ?;

-- name: DeleteInstance :exec
DELETE FROM task_instances
WHERE id = ?;