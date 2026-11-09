-- name: CreateProject :one
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: GetProjectByDisplayID :one
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at
FROM projects
WHERE display_id = $1;

-- name: ListProjects :many
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin
FROM projects p
WHERE sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id))
ORDER BY p.created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountProjects :one
SELECT COUNT(*)
FROM projects p
WHERE sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id));

-- name: SearchProjects :many
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       ts_rank_cd(p.search, websearch_to_tsquery('english', sqlc.arg(query))) AS rank,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin
FROM projects p
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%')
ORDER BY rank DESC, p.created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountSearchProjects :one
SELECT COUNT(*)
FROM projects p
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%');

-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, is_admin)
VALUES ($1, $2, $3);

-- name: GetProjectMembership :one
SELECT is_admin FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectMembers :many
SELECT pm.user_id, u.name, u.email, pm.is_admin, pm.created_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at, u.name;

-- name: SetProjectMemberAdmin :execrows
UPDATE project_members SET is_admin = $3 WHERE project_id = $1 AND user_id = $2;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: LockProjectAdmins :many
SELECT user_id FROM project_members WHERE project_id = $1 AND is_admin FOR UPDATE;

-- name: AddProjectMemberIfAbsent :exec
INSERT INTO project_members (project_id, user_id, is_admin)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: FindUserByEmail :one
SELECT id, name, email FROM users WHERE email = $1;

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
