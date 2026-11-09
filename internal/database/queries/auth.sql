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

-- name: IncrementUserTokenVersion :exec
UPDATE users
SET token_version = token_version + 1, updated_at = NOW()
WHERE id = $1;
