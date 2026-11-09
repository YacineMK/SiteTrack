-- name: ListCompanyStock :many
SELECT m.id AS material_id, m.name, m.category, m.unit,
       COALESCE(s.quantity, 0)::numeric AS quantity,
       COALESCE(s.reorder_point, 0)::numeric AS reorder_point,
       COALESCE((SELECT SUM(dl.quantity)
                 FROM delivery_lines dl
                 JOIN deliveries d ON d.id = dl.delivery_id
                 WHERE dl.material_id = m.id AND d.sender_type = 'company' AND d.status = 'in_transit'), 0)::numeric AS in_transit
FROM materials m
LEFT JOIN company_store_stock s ON s.material_id = m.id
WHERE (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%')
ORDER BY m.name
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountCompanyStock :one
SELECT COUNT(*)
FROM materials m
WHERE (sqlc.arg(query)::text = '' OR m.name ILIKE '%' || sqlc.arg(query)::text || '%' OR m.category ILIKE '%' || sqlc.arg(query)::text || '%');

-- name: GetCompanyStock :one
SELECT m.id AS material_id, m.name, m.category, m.unit,
       COALESCE(s.quantity, 0)::numeric AS quantity,
       COALESCE(s.reorder_point, 0)::numeric AS reorder_point,
       COALESCE((SELECT SUM(dl.quantity)
                 FROM delivery_lines dl
                 JOIN deliveries d ON d.id = dl.delivery_id
                 WHERE dl.material_id = m.id AND d.sender_type = 'company' AND d.status = 'in_transit'), 0)::numeric AS in_transit
FROM materials m
LEFT JOIN company_store_stock s ON s.material_id = m.id
WHERE m.id = $1;

-- name: AddCompanyStock :one
INSERT INTO company_store_stock (material_id, quantity)
VALUES ($1, $2)
ON CONFLICT (material_id) DO UPDATE
SET quantity = company_store_stock.quantity + EXCLUDED.quantity,
    updated_at = NOW()
RETURNING material_id, quantity, reorder_point, updated_at;

-- name: SetCompanyReorderPoint :one
INSERT INTO company_store_stock (material_id, quantity, reorder_point)
VALUES ($1, 0, $2)
ON CONFLICT (material_id) DO UPDATE
SET reorder_point = EXCLUDED.reorder_point,
    updated_at = NOW()
RETURNING material_id, quantity, reorder_point, updated_at;

-- name: DeductCompanyStock :execrows
UPDATE company_store_stock
SET quantity = quantity - $2,
    updated_at = NOW()
WHERE material_id = $1 AND quantity >= $2;
