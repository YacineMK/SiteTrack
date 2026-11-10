-- name: CreateMaterial :one
INSERT INTO materials (id, name, category, unit) VALUES ($1, $2, $3, $4)
RETURNING id, name, category, unit, created_at, updated_at;

-- name: GetMaterial :one
SELECT id, name, category, unit, created_at, updated_at FROM materials WHERE id = $1;

-- name: ListMaterials :many
SELECT id, name, category, unit, created_at, updated_at
FROM materials
WHERE sqlc.arg(q)::text = '' OR name ILIKE '%' || sqlc.arg(q) || '%'
ORDER BY name, id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountMaterials :one
SELECT COUNT(*) FROM materials
WHERE sqlc.arg(q)::text = '' OR name ILIKE '%' || sqlc.arg(q) || '%';

-- name: UpdateMaterial :one
UPDATE materials SET name = $2, category = $3, unit = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, name, category, unit, created_at, updated_at;

-- name: DeleteMaterial :execrows
DELETE FROM materials WHERE id = $1;
