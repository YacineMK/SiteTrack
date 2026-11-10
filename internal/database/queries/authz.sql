-- name: CreateRole :one
INSERT INTO app_roles (name, description) VALUES ($1, $2)
RETURNING id, name, description;

-- name: ListRoles :many
SELECT id, name, description FROM app_roles ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description FROM app_roles WHERE id = $1;

-- name: GetRoleByName :one
SELECT id, name, description FROM app_roles WHERE name = $1;

-- name: GetPoliciesByRoleName :many
SELECT rp.resource, rp.action
FROM role_policies rp
JOIN app_roles ar ON ar.id = rp.role_id
WHERE ar.name = $1;

-- name: UpsertRole :exec
INSERT INTO app_roles (name, description) VALUES ($1, $2)
ON CONFLICT (name) DO NOTHING;

-- name: UpsertRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action)
SELECT ar.id, $2, $3 FROM app_roles ar WHERE ar.name = $1
ON CONFLICT DO NOTHING;

-- name: AssignUserRole :one
WITH updated AS (
    UPDATE users
    SET role_id = (SELECT app_roles.id FROM app_roles WHERE app_roles.name = sqlc.arg(role)),
        updated_at = NOW()
    WHERE users.id = sqlc.arg(user_id)
    RETURNING id, name, email, role_id, created_at, updated_at
)
SELECT u.id, u.name, u.email, ar.name AS app_role, u.created_at, u.updated_at
FROM updated u
JOIN app_roles ar ON ar.id = u.role_id;

-- name: AddRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action) VALUES ($1, $2, $3);

-- name: ListRolePolicies :many
SELECT role_id, resource, action FROM role_policies
WHERE role_id = $1 ORDER BY resource, action;

-- name: DeleteRolePolicy :execrows
DELETE FROM role_policies WHERE role_id = $1 AND resource = $2 AND action = $3;
