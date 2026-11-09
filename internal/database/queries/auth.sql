-- name: GetUserByEmail :one
SELECT u.id, u.name, u.email, u.password_hash, ar.name AS app_role, u.token_version, u.created_at, u.updated_at
FROM users u
JOIN app_roles ar ON ar.id = u.role_id
WHERE u.email = $1;

-- name: GetUserByID :one
SELECT u.id, u.name, u.email, u.password_hash, ar.name AS app_role, u.token_version, u.created_at, u.updated_at
FROM users u
JOIN app_roles ar ON ar.id = u.role_id
WHERE u.id = $1;

-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, app_role, role_id)
VALUES ($1, $2, $3, $4, $5, (SELECT id FROM app_roles WHERE name = $5))
RETURNING id, name, email, password_hash, app_role, token_version, created_at, updated_at;

-- name: EmailExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);
