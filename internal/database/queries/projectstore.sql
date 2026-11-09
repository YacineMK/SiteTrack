-- name: ListProjectStock :many
SELECT m.id AS material_id, m.name, m.category, m.unit, s.quantity, ld.last_delivery
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
LEFT JOIN (
  SELECT dl.material_id,
         COALESCE(MAX(d.sent_at), '0001-01-01 00:00:00+00'::timestamptz)::timestamptz AS last_delivery
  FROM delivery_lines dl
  JOIN deliveries d ON d.id = dl.delivery_id
  WHERE (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid)
     OR (d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid)
  GROUP BY dl.material_id
) ld ON ld.material_id = m.id
WHERE s.project_id = sqlc.arg(project_id)::uuid
  AND (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%')
  AND (sqlc.arg(category)::text = '' OR m.category = sqlc.arg(category)::text)
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'asc' THEN m.name END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN m.name END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'quantity' AND sqlc.arg(sort_dir)::text = 'asc' THEN s.quantity END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'quantity' AND sqlc.arg(sort_dir)::text = 'desc' THEN s.quantity END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'last_delivery' AND sqlc.arg(sort_dir)::text = 'asc' THEN ld.last_delivery END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'last_delivery' AND sqlc.arg(sort_dir)::text = 'desc' THEN ld.last_delivery END DESC,
  m.name ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjectStock :one
SELECT COUNT(*)
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = sqlc.arg(project_id)::uuid
  AND (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%')
  AND (sqlc.arg(category)::text = '' OR m.category = sqlc.arg(category)::text);

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
