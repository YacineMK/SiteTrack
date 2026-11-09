-- name: CreateProject :one
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: GetProjectByDisplayID :one
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at
FROM projects
WHERE display_id = $1;

-- name: ListProjects :many
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at
FROM projects
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountProjects :one
SELECT COUNT(*) FROM projects;

-- name: SearchProjects :many
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       ts_rank_cd(p.search, websearch_to_tsquery('english', sqlc.arg(query))) AS rank
FROM projects p
WHERE p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%'
ORDER BY rank DESC, p.created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountSearchProjects :one
SELECT COUNT(*)
FROM projects p
WHERE p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%';

-- name: UpdateProject :one
UPDATE projects
SET name = $2,
    description = $3,
    planned_start_date = $4,
    planned_end_date = $5,
    updated_at = NOW()
WHERE display_id = $1
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE display_id = $1;
