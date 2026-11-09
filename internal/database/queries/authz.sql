-- name: CreateRole :one
INSERT INTO app_roles (name, description)
VALUES ($1, $2)
RETURNING id, name, description;

-- name: ListRoles :many
SELECT id, name, description
FROM app_roles
ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description
FROM app_roles
WHERE id = $1;

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
