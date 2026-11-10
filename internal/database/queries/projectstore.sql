-- name: ListProjectStock :many
SELECT m.id AS material_id, m.name, m.category, m.unit, s.quantity,
       (SELECT MAX(d.sent_at)::timestamptz FROM delivery_lines dl
        JOIN deliveries d ON d.id = dl.delivery_id
        WHERE dl.material_id = m.id
          AND (d.sender_id = sqlc.arg(project_id)::uuid OR d.receiver_id = sqlc.arg(project_id)::uuid))
       AS last_delivery
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = sqlc.arg(project_id)::uuid
  AND (sqlc.arg(q)::text        = '' OR m.name     ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.arg(category)::text = '' OR m.category = sqlc.arg(category))
ORDER BY
    CASE sqlc.arg(sort)::text WHEN 'quantity'      THEN s.quantity END DESC,
    CASE sqlc.arg(sort)::text WHEN 'last_delivery' THEN
        (SELECT MAX(d.sent_at)::timestamptz FROM delivery_lines dl
         JOIN deliveries d ON d.id = dl.delivery_id
         WHERE dl.material_id = m.id
           AND (d.sender_id = sqlc.arg(project_id)::uuid OR d.receiver_id = sqlc.arg(project_id)::uuid))
    END DESC,
    m.name ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjectStock :one
SELECT COUNT(*)
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = sqlc.arg(project_id)::uuid
  AND (sqlc.arg(q)::text        = '' OR m.name     ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.arg(category)::text = '' OR m.category = sqlc.arg(category));

-- name: GetProjectStock :one
SELECT m.id AS material_id, m.name, m.category, m.unit, s.quantity
FROM project_store_stock s
JOIN materials m ON m.id = s.material_id
WHERE s.project_id = $1 AND s.material_id = $2;

-- name: DeductProjectStock :execrows
UPDATE project_store_stock SET quantity = quantity - $3, updated_at = NOW()
WHERE project_id = $1 AND material_id = $2 AND quantity >= $3;

-- name: AddProjectStock :one
INSERT INTO project_store_stock (project_id, material_id, quantity) VALUES ($1, $2, $3)
ON CONFLICT (project_id, material_id) DO UPDATE
SET quantity = project_store_stock.quantity + EXCLUDED.quantity, updated_at = NOW()
RETURNING project_id, material_id, quantity, updated_at;
