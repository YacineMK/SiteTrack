-- All queries here read/write app-scope rows only (roles.project_id IS NULL).

-- name: CreateRole :one
INSERT INTO roles (name, description) VALUES ($1, $2)
RETURNING id, name, description;

-- name: ListRoles :many
SELECT id, name, description FROM roles WHERE project_id IS NULL ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description FROM roles WHERE id = $1 AND project_id IS NULL;

-- name: GetRoleByName :one
SELECT id, name, description FROM roles WHERE name = $1 AND project_id IS NULL;

-- name: GetPoliciesByRoleName :many
SELECT rp.resource, rp.action
FROM role_policies rp
JOIN roles r ON r.id = rp.role_id
WHERE r.name = $1 AND r.project_id IS NULL;

-- name: UpsertRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action)
SELECT r.id, $2, $3 FROM roles r WHERE r.name = $1 AND r.project_id IS NULL
ON CONFLICT DO NOTHING;

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

-- name: ListRolePolicies :many
SELECT role_id, resource, action FROM role_policies
WHERE role_id = $1 ORDER BY resource, action;

-- name: DeleteRolePolicy :execrows
DELETE FROM role_policies WHERE role_id = $1 AND resource = $2 AND action = $3;
