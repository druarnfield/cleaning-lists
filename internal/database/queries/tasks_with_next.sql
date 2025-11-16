-- name: GetTasksWithNextScheduled :many
SELECT
    t.id,
    t.name,
    t.category,
    t.frequency,
    t.default_assignee,
    t.estimated_mins,
    t.created_at,
    t.updated_at,
    COALESCE((SELECT scheduled_date FROM task_instances
     WHERE task_id = t.id
       AND scheduled_date >= date('now')
       AND id NOT IN (SELECT task_instance_id FROM completions)
     ORDER BY scheduled_date
     LIMIT 1), NULL) as next_scheduled_date,
    COALESCE((SELECT assigned_to FROM task_instances
     WHERE task_id = t.id
       AND scheduled_date >= date('now')
       AND id NOT IN (SELECT task_instance_id FROM completions)
     ORDER BY scheduled_date
     LIMIT 1), NULL) as next_assigned_to
FROM tasks t
ORDER BY t.name;