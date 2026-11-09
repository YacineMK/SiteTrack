-- name: CheckRolePolicy :one
SELECT EXISTS (
    SELECT 1
    FROM role_policies rp
    JOIN app_roles ar ON ar.id = rp.role_id
    WHERE ar.name = sqlc.arg(role_name)
      AND rp.resource = sqlc.arg(resource)
      AND (
          rp.action = sqlc.arg(action)
          OR (sqlc.arg(action) = 'read' AND rp.action = 'write')
      )
) AS allowed;

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
