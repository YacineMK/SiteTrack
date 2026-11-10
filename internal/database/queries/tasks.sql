-- name: CreateTask :one
INSERT INTO tasks (id, display_id, project_id, name, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date,
          status, blocked, dependents_count, created_at, updated_at;

-- name: GetTask :one
SELECT id, display_id, project_id, name, planned_start_date, planned_end_date,
       status, blocked, dependents_count, created_at, updated_at
FROM tasks
WHERE project_id = $1 AND display_id = $2;

-- name: UpdateTask :one
UPDATE tasks
SET name = $3, planned_start_date = $4, planned_end_date = $5, updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date,
          status, blocked, dependents_count, created_at, updated_at;

-- name: UpdateTaskStatus :one
UPDATE tasks SET status = $3, updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date,
          status, blocked, dependents_count, created_at, updated_at;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE project_id = $1 AND display_id = $2;

-- name: ListTasks :many
SELECT t.id, t.display_id, t.project_id, t.name, t.planned_start_date, t.planned_end_date,
       t.status, t.blocked, t.dependents_count, t.created_at, t.updated_at,
       (SELECT COALESCE(jsonb_agg(d.display_id ORDER BY d.display_id), '[]'::jsonb)::text
        FROM task_dependencies td JOIN tasks d ON d.id = td.depends_on
        WHERE td.task_id = t.id) AS depends_on
FROM tasks t
WHERE t.project_id = sqlc.arg(project_id)
  AND (sqlc.arg(q)::text = '' OR t.name ILIKE '%' || sqlc.arg(q) || '%' OR t.display_id ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status))
  AND (NOT sqlc.arg(late_only)::bool
       OR (t.status = 'not_started' AND t.planned_start_date < CURRENT_DATE)
       OR (t.status = 'in_progress' AND t.planned_end_date   < CURRENT_DATE))
  AND (NOT sqlc.arg(blocked_only)::bool OR t.blocked)
  AND (sqlc.narg(from_date)::date IS NULL OR t.planned_start_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date   IS NULL OR t.planned_end_date   <= sqlc.narg(to_date))
ORDER BY
    CASE sqlc.arg(sort)::text WHEN 'name'       THEN t.name                END ASC,
    CASE sqlc.arg(sort)::text WHEN 'start_date' THEN t.planned_start_date  END ASC,
    CASE sqlc.arg(sort)::text WHEN 'end_date'   THEN t.planned_end_date    END ASC,
    CASE sqlc.arg(sort)::text WHEN 'dependents' THEN t.dependents_count    END DESC,
    t.display_id ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks t
WHERE t.project_id = sqlc.arg(project_id)
  AND (sqlc.arg(q)::text = '' OR t.name ILIKE '%' || sqlc.arg(q) || '%' OR t.display_id ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status))
  AND (NOT sqlc.arg(late_only)::bool
       OR (t.status = 'not_started' AND t.planned_start_date < CURRENT_DATE)
       OR (t.status = 'in_progress' AND t.planned_end_date   < CURRENT_DATE))
  AND (NOT sqlc.arg(blocked_only)::bool OR t.blocked)
  AND (sqlc.narg(from_date)::date IS NULL OR t.planned_start_date >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date   IS NULL OR t.planned_end_date   <= sqlc.narg(to_date));

-- cycle check: refuses the insert if new_task is already upstream of depends_on.
-- name: AddTaskDependency :execrows
WITH RECURSIVE upstream(id) AS (
    SELECT sqlc.arg(depends_on_id)::uuid
    UNION
    SELECT td.depends_on FROM task_dependencies td JOIN upstream ON td.task_id = upstream.id
)
INSERT INTO task_dependencies (task_id, depends_on)
SELECT sqlc.arg(new_task_id), sqlc.arg(depends_on_id)
WHERE NOT EXISTS (SELECT 1 FROM upstream WHERE id = sqlc.arg(new_task_id));

-- name: RemoveTaskDependency :execrows
DELETE FROM task_dependencies WHERE task_id = $1 AND depends_on = $2;

-- name: TaskHasBlockingUpstream :one
WITH RECURSIVE upstream(id) AS (
    SELECT td.depends_on FROM task_dependencies td WHERE td.task_id = $1
    UNION
    SELECT td.depends_on FROM task_dependencies td JOIN upstream ON td.task_id = upstream.id
)
SELECT EXISTS (SELECT 1 FROM upstream u JOIN tasks t ON t.id = u.id WHERE t.status <> 'done');

-- name: TaskHasStartedDownstream :one
WITH RECURSIVE downstream(id) AS (
    SELECT td.task_id FROM task_dependencies td WHERE td.depends_on = $1
    UNION
    SELECT td.task_id FROM task_dependencies td JOIN downstream ON td.depends_on = downstream.id
)
SELECT EXISTS (SELECT 1 FROM downstream d JOIN tasks t ON t.id = d.id WHERE t.status <> 'not_started');

-- name: ListUpstreamDependencies :many
WITH RECURSIVE upstream(id, depth) AS (
    SELECT td.depends_on, 1 FROM task_dependencies td WHERE td.task_id = $1
    UNION
    SELECT td.depends_on, u.depth + 1 FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
SELECT t.display_id, t.name, t.status, MIN(u.depth)::int AS steps
FROM upstream u JOIN tasks t ON t.id = u.id
GROUP BY t.id
ORDER BY steps, t.display_id;

-- name: ListDownstreamDependencies :many
WITH RECURSIVE downstream(id, depth) AS (
    SELECT td.task_id, 1 FROM task_dependencies td WHERE td.depends_on = $1
    UNION
    SELECT td.task_id, d.depth + 1 FROM task_dependencies td JOIN downstream d ON td.depends_on = d.id
)
SELECT t.display_id, t.name, t.status, MIN(d.depth)::int AS steps
FROM downstream d JOIN tasks t ON t.id = d.id
GROUP BY t.id
ORDER BY steps, t.display_id;
