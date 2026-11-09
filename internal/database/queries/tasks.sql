-- name: CreateTask :one
INSERT INTO tasks (id, display_id, project_id, name, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at;

-- name: GetTask :one
SELECT id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at
FROM tasks
WHERE project_id = $1 AND display_id = $2;

-- name: ListTasks :many
SELECT id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at
FROM tasks
WHERE project_id = $1
  AND (sqlc.arg(query)::text = '' OR name ILIKE '%' || sqlc.arg(query)::text || '%' OR display_id ILIKE '%' || sqlc.arg(query)::text || '%')
ORDER BY display_id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND (sqlc.arg(query)::text = '' OR name ILIKE '%' || sqlc.arg(query)::text || '%' OR display_id ILIKE '%' || sqlc.arg(query)::text || '%');

-- name: UpdateTask :one
UPDATE tasks
SET name = $3,
    planned_start_date = $4,
    planned_end_date = $5,
    updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at;

-- name: UpdateTaskStatus :one
UPDATE tasks
SET status = $3,
    updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE project_id = $1 AND display_id = $2;

-- name: AddTaskDependency :execrows
WITH RECURSIVE upstream AS (
    SELECT td.depends_on AS id FROM task_dependencies td WHERE td.task_id = sqlc.arg(depends_on_id)
    UNION ALL
    SELECT td.depends_on FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
INSERT INTO task_dependencies (task_id, depends_on)
SELECT sqlc.arg(new_task_id), sqlc.arg(depends_on_id)
WHERE NOT EXISTS (SELECT 1 FROM upstream WHERE id = sqlc.arg(new_task_id));

-- name: RemoveTaskDependency :execrows
DELETE FROM task_dependencies WHERE task_id = $1 AND depends_on = $2;

-- name: TaskHasBlockingUpstream :one
WITH RECURSIVE upstream AS (
    SELECT td.depends_on AS id FROM task_dependencies td WHERE td.task_id = sqlc.arg(target_id)
    UNION ALL
    SELECT td.depends_on FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
SELECT EXISTS (
    SELECT 1 FROM upstream u JOIN tasks t ON t.id = u.id WHERE t.status <> 'done'
);

-- name: TaskHasStartedDownstream :one
WITH RECURSIVE downstream AS (
    SELECT td.task_id AS id FROM task_dependencies td WHERE td.depends_on = sqlc.arg(target_id)
    UNION ALL
    SELECT td.task_id FROM task_dependencies td JOIN downstream d ON td.depends_on = d.id
)
SELECT EXISTS (
    SELECT 1 FROM downstream d JOIN tasks t ON t.id = d.id WHERE t.status <> 'not_started'
);

-- name: ListUpstreamDependencies :many
WITH RECURSIVE upstream AS (
    SELECT td.depends_on AS id, 1 AS depth FROM task_dependencies td WHERE td.task_id = sqlc.arg(target_id)
    UNION ALL
    SELECT td.depends_on, u.depth + 1 FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
SELECT t.display_id, t.name, t.status, MIN(u.depth)::int AS steps
FROM upstream u JOIN tasks t ON t.id = u.id
GROUP BY t.id, t.display_id, t.name, t.status
ORDER BY steps, t.display_id;

-- name: ListDownstreamDependencies :many
WITH RECURSIVE downstream AS (
    SELECT td.task_id AS id, 1 AS depth FROM task_dependencies td WHERE td.depends_on = sqlc.arg(target_id)
    UNION ALL
    SELECT td.task_id, d.depth + 1 FROM task_dependencies td JOIN downstream d ON td.depends_on = d.id
)
SELECT t.display_id, t.name, t.status, MIN(d.depth)::int AS steps
FROM downstream d JOIN tasks t ON t.id = d.id
GROUP BY t.id, t.display_id, t.name, t.status
ORDER BY steps, t.display_id;

-- name: ListBlockedTasks :many
WITH RECURSIVE reach AS (
    SELECT td.task_id AS root, td.depends_on AS upstream
    FROM task_dependencies td
    JOIN tasks t ON t.id = td.task_id
    WHERE t.project_id = sqlc.arg(target_project_id)
    UNION ALL
    SELECT r.root, td.depends_on
    FROM task_dependencies td
    JOIN reach r ON td.task_id = r.upstream
)
SELECT DISTINCT r.root
FROM reach r
JOIN tasks t ON t.id = r.upstream
WHERE t.status <> 'done';
