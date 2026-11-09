-- name: ListProjectStock :many
SELECT m.id AS material_id, m.name, m.category, m.unit, s.quantity
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = $1
  AND (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%')
ORDER BY m.name
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjectStock :one
SELECT COUNT(*)
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = $1
  AND (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%');

-- name: GetProjectStock :one
SELECT m.id AS material_id, m.name, m.category, m.unit, s.quantity
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = $1 AND s.material_id = $2;

-- name: DeductProjectStock :execrows
UPDATE project_store_stock
SET quantity = quantity - $3,
    updated_at = NOW()
WHERE project_id = $1 AND material_id = $2 AND quantity >= $3;

-- name: AddProjectStock :one
INSERT INTO project_store_stock (project_id, material_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, material_id) DO UPDATE
SET quantity = project_store_stock.quantity + EXCLUDED.quantity,
    updated_at = NOW()
RETURNING project_id, material_id, quantity, updated_at;
