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
ORDER BY display_id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountTasks :one
SELECT COUNT(*) FROM tasks WHERE project_id = $1;

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
