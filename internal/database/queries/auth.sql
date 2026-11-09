-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, app_role, token_version, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, app_role, token_version, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, app_role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, password_hash, app_role, token_version, created_at, updated_at;

-- name: EmailExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);
