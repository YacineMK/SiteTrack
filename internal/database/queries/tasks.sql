-- name: CreateTask :one
INSERT INTO tasks (id, display_id, project_id, name, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at, blocked, dependents_count;

-- name: GetTask :one
SELECT id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at, blocked, dependents_count
FROM tasks
WHERE project_id = $1 AND display_id = $2;

-- name: ListTasks :many
WITH cur AS (
    SELECT name, planned_start_date, planned_end_date, dependents_count, display_id
    FROM tasks
    WHERE project_id = sqlc.arg(project_id) AND display_id = sqlc.arg(after_id)
)
SELECT t.id, t.display_id, t.project_id, t.name, t.planned_start_date, t.planned_end_date,
       t.status, t.blocked, t.dependents_count, t.created_at, t.updated_at,
       (SELECT COALESCE(jsonb_agg(t2.display_id ORDER BY t2.display_id), '[]'::jsonb)::text
        FROM task_dependencies td JOIN tasks t2 ON t2.id = td.depends_on
        WHERE td.task_id = t.id) AS depends_on
FROM tasks t
LEFT JOIN cur ON TRUE
WHERE t.project_id = sqlc.arg(project_id)
  AND (sqlc.arg(query)::text = '' OR t.name ILIKE '%' || sqlc.arg(query)::text || '%' OR t.display_id ILIKE '%' || sqlc.arg(query)::text || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR t.planned_start_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR t.planned_end_date <= NULLIF(sqlc.arg(to_date)::text, '')::date)
  AND (NOT sqlc.arg(late_only)::boolean OR (t.status = 'not_started' AND t.planned_start_date < CURRENT_DATE) OR (t.status = 'in_progress' AND t.planned_end_date < CURRENT_DATE))
  AND (NOT sqlc.arg(blocked_only)::boolean OR t.blocked)
  AND (sqlc.arg(after_id)::text = '' OR (cur.display_id IS NOT NULL AND CASE
        WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN (t.name, t.display_id) < (cur.name, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'name' THEN (t.name, t.display_id) > (cur.name, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN (t.planned_start_date, t.display_id) < (cur.planned_start_date, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'start_date' THEN (t.planned_start_date, t.display_id) > (cur.planned_start_date, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'end_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN (t.planned_end_date, t.display_id) < (cur.planned_end_date, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'end_date' THEN (t.planned_end_date, t.display_id) > (cur.planned_end_date, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'dependents' AND sqlc.arg(sort_dir)::text = 'desc' THEN (t.dependents_count, t.display_id) < (cur.dependents_count, cur.display_id)
        WHEN sqlc.arg(sort_by)::text = 'dependents' THEN (t.dependents_count, t.display_id) > (cur.dependents_count, cur.display_id)
        WHEN sqlc.arg(sort_dir)::text = 'desc' THEN t.display_id < cur.display_id
        ELSE t.display_id > cur.display_id
      END))
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'asc' THEN t.name END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN t.name END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'asc' THEN t.planned_start_date END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN t.planned_start_date END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'end_date' AND sqlc.arg(sort_dir)::text = 'asc' THEN t.planned_end_date END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'end_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN t.planned_end_date END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'dependents' AND sqlc.arg(sort_dir)::text = 'asc' THEN t.dependents_count END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'dependents' AND sqlc.arg(sort_dir)::text = 'desc' THEN t.dependents_count END DESC,
  CASE WHEN sqlc.arg(sort_dir)::text = 'desc' THEN t.display_id END DESC,
  t.display_id ASC
LIMIT sqlc.arg(page_limit);

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks t
WHERE t.project_id = sqlc.arg(project_id)
  AND (sqlc.arg(query)::text = '' OR t.name ILIKE '%' || sqlc.arg(query)::text || '%' OR t.display_id ILIKE '%' || sqlc.arg(query)::text || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR t.planned_start_date >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR t.planned_end_date <= NULLIF(sqlc.arg(to_date)::text, '')::date)
  AND (NOT sqlc.arg(late_only)::boolean OR (t.status = 'not_started' AND t.planned_start_date < CURRENT_DATE) OR (t.status = 'in_progress' AND t.planned_end_date < CURRENT_DATE))
  AND (NOT sqlc.arg(blocked_only)::boolean OR t.blocked);

-- name: UpdateTask :one
UPDATE tasks
SET name = $3,
    planned_start_date = $4,
    planned_end_date = $5,
    updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at, blocked, dependents_count;

-- name: UpdateTaskStatus :one
UPDATE tasks
SET status = $3,
    updated_at = NOW()
WHERE project_id = $1 AND display_id = $2
RETURNING id, display_id, project_id, name, planned_start_date, planned_end_date, status, created_at, updated_at, blocked, dependents_count;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE project_id = $1 AND display_id = $2;

-- name: AddTaskDependency :execrows
WITH RECURSIVE upstream AS (
    SELECT td.depends_on AS id FROM task_dependencies td WHERE td.task_id = sqlc.arg(depends_on_id)
    UNION
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
    UNION
    SELECT td.depends_on FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
SELECT EXISTS (
    SELECT 1 FROM upstream u JOIN tasks t ON t.id = u.id WHERE t.status <> 'done'
);

-- name: TaskHasStartedDownstream :one
WITH RECURSIVE downstream AS (
    SELECT td.task_id AS id FROM task_dependencies td WHERE td.depends_on = sqlc.arg(target_id)
    UNION
    SELECT td.task_id FROM task_dependencies td JOIN downstream d ON td.depends_on = d.id
)
SELECT EXISTS (
    SELECT 1 FROM downstream d JOIN tasks t ON t.id = d.id WHERE t.status <> 'not_started'
);

-- name: ListUpstreamDependencies :many
WITH RECURSIVE upstream AS (
    SELECT td.depends_on AS id, 1 AS depth FROM task_dependencies td WHERE td.task_id = sqlc.arg(target_id)
    UNION
    SELECT td.depends_on, u.depth + 1 FROM task_dependencies td JOIN upstream u ON td.task_id = u.id
)
SELECT t.display_id, t.name, t.status, MIN(u.depth)::int AS steps
FROM upstream u JOIN tasks t ON t.id = u.id
GROUP BY t.id, t.display_id, t.name, t.status
ORDER BY steps, t.display_id;

-- name: ListDownstreamDependencies :many
WITH RECURSIVE downstream AS (
    SELECT td.task_id AS id, 1 AS depth FROM task_dependencies td WHERE td.depends_on = sqlc.arg(target_id)
    UNION
    SELECT td.task_id, d.depth + 1 FROM task_dependencies td JOIN downstream d ON td.depends_on = d.id
)
SELECT t.display_id, t.name, t.status, MIN(d.depth)::int AS steps
FROM downstream d JOIN tasks t ON t.id = d.id
GROUP BY t.id, t.display_id, t.name, t.status
ORDER BY steps, t.display_id;
