-- name: CreateRole :one
INSERT INTO app_roles (name, description)
VALUES ($1, $2)
RETURNING id, name, description, is_system;

-- name: ListRoles :many
SELECT id, name, description, is_system
FROM app_roles
ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description, is_system
FROM app_roles
WHERE id = $1;

-- name: GetRoleByName :one
SELECT id, name, description, is_system
FROM app_roles
WHERE name = $1;

-- name: GetPoliciesByRoleName :many
SELECT rp.resource, rp.action
FROM role_policies rp
JOIN app_roles ar ON ar.id = rp.role_id
WHERE ar.name = $1;

-- name: UpsertRole :exec
INSERT INTO app_roles (name, description, is_system)
VALUES ($1, $2, TRUE)
ON CONFLICT (name) DO UPDATE SET is_system = TRUE;

-- name: UpsertRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action)
SELECT ar.id, $2, $3 FROM app_roles ar WHERE ar.name = $1
ON CONFLICT DO NOTHING;

-- name: AssignUserRole :one
UPDATE users
SET role_id = (SELECT ar.id FROM app_roles ar WHERE ar.name = sqlc.arg(role_name)),
    app_role = sqlc.arg(role_name),
    token_version = token_version + 1,
    updated_at = NOW()
WHERE users.id = sqlc.arg(user_id)
RETURNING id, name, email, app_role, token_version, created_at, updated_at;

-- name: AddRolePolicy :exec
INSERT INTO role_policies (role_id, resource, action)
VALUES ($1, $2, $3);

-- name: ListRolePolicies :many
SELECT role_id, resource, action
FROM role_policies
WHERE role_id = $1
ORDER BY resource, action;

-- name: DeleteRolePolicy :execrows
DELETE FROM role_policies
WHERE role_id = $1 AND resource = $2 AND action = $3;
