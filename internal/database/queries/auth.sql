-- name: GetUserByEmail :one
SELECT u.id, u.name, u.email, u.password_hash, ar.name AS app_role, u.created_at, u.updated_at
FROM users u
JOIN app_roles ar ON ar.id = u.role_id
WHERE u.email = $1;

-- name: GetUserByID :one
SELECT u.id, u.name, u.email, u.password_hash, ar.name AS app_role, u.created_at, u.updated_at
FROM users u
JOIN app_roles ar ON ar.id = u.role_id
WHERE u.id = $1;

-- name: CreateUser :one
WITH new_user AS (
    INSERT INTO users (id, name, email, password_hash, role_id)
    VALUES ($1, $2, $3, $4, (SELECT app_roles.id FROM app_roles WHERE app_roles.name = sqlc.arg(role)))
    RETURNING id, name, email, password_hash, role_id, created_at, updated_at
)
SELECT u.id, u.name, u.email, u.password_hash, ar.name AS app_role, u.created_at, u.updated_at
FROM new_user u
JOIN app_roles ar ON ar.id = u.role_id;

-- name: EmailExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: RoleExists :one
SELECT EXISTS(SELECT 1 FROM app_roles WHERE name = $1);
