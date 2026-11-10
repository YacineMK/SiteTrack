-- Scope: pass NULL in sqlc.narg(project_id) for app-scope rows,
-- the project uuid for project-scope rows. Same table, one query per op.

-- name: ListRoles :many
SELECT id, name, description, is_admin, is_system, created_at, updated_at
FROM roles
WHERE project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
ORDER BY is_system DESC, name;

-- name: GetRoleByID :one
SELECT id, name, description, is_admin, is_system, created_at, updated_at
FROM roles
WHERE id = $1 AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid;

-- name: GetRoleByName :one
SELECT id, name, description, is_admin, is_system, created_at, updated_at
FROM roles
WHERE name = $1 AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid;

-- name: CreateRole :one
INSERT INTO roles (id, project_id, name, description, is_admin, is_system)
VALUES (gen_random_uuid(), sqlc.narg(project_id)::uuid, $1, $2, $3, FALSE)
RETURNING id, name, description, is_admin, is_system, created_at, updated_at;

-- name: UpdateRole :one
UPDATE roles SET name = $2, description = $3, is_admin = $4, updated_at = NOW()
WHERE id = $1
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
  AND NOT is_system
RETURNING id, name, description, is_admin, is_system, created_at, updated_at;

-- name: DeleteRole :execrows
DELETE FROM roles
WHERE id = $1
  AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
  AND NOT is_system;

-- name: GetPoliciesByRoleName :many
SELECT rp.resource, rp.action
FROM role_policies rp
JOIN roles r ON r.id = rp.role_id
WHERE r.name = $1 AND r.project_id IS NULL;

-- name: AssignUserRole :one
WITH updated AS (
    UPDATE users
    SET role_id = (SELECT roles.id FROM roles WHERE roles.name = sqlc.arg(role) AND roles.project_id IS NULL),
        updated_at = NOW()
    WHERE users.id = sqlc.arg(user_id)
    RETURNING id, name, email, role_id, created_at, updated_at
)
SELECT u.id, u.name, u.email, r.name AS app_role, u.created_at, u.updated_at
FROM updated u
JOIN roles r ON r.id = u.role_id;

-- name: AddRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action) VALUES ($1, $2, $3);

-- name: UpsertAppRolePolicyByName :exec
INSERT INTO role_policies (role_id, resource, action)
SELECT r.id, $2, $3 FROM roles r WHERE r.name = $1 AND r.project_id IS NULL
ON CONFLICT DO NOTHING;

-- name: ListRolePolicies :many
SELECT role_id, resource, action FROM role_policies
WHERE role_id = $1 ORDER BY resource, action;

-- name: DeleteRolePolicy :execrows
DELETE FROM role_policies WHERE role_id = $1 AND resource = $2 AND action = $3;
