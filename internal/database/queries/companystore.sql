-- name: ListCompanyStock :many
WITH stock AS (
    SELECT m.id AS material_id, m.name, m.category, m.unit,
           COALESCE(s.quantity,      0)::numeric AS quantity,
           COALESCE(s.reorder_point, 0)::numeric AS reorder_point,
           CASE
               WHEN COALESCE(s.quantity, 0) = 0                      THEN 'out_of_stock'
               WHEN COALESCE(s.quantity, 0) <= COALESCE(s.reorder_point, 0) THEN 'low'
               ELSE 'available'
           END AS status,
           COALESCE((SELECT SUM(dl.quantity) FROM delivery_lines dl
                     JOIN deliveries d ON d.id = dl.delivery_id
                     WHERE dl.material_id = m.id
                       AND d.sender_type = 'company' AND d.status = 'in_transit'), 0)::numeric AS in_transit
    FROM materials m
    LEFT JOIN company_store_stock s ON s.material_id = m.id
)
SELECT material_id, name, category, unit, quantity, reorder_point, status, in_transit
FROM stock
WHERE (sqlc.arg(q)::text        = '' OR name ILIKE '%' || sqlc.arg(q)        || '%')
  AND (sqlc.arg(category)::text = '' OR category = sqlc.arg(category))
  AND (sqlc.arg(status)::text   = '' OR status   = sqlc.arg(status))
ORDER BY
    CASE sqlc.arg(sort)::text WHEN 'quantity' THEN quantity END ASC,
    CASE sqlc.arg(sort)::text WHEN 'status'   THEN
        CASE status WHEN 'out_of_stock' THEN 0 WHEN 'low' THEN 1 ELSE 2 END
    END ASC,
    name ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountCompanyStock :one
WITH stock AS (
    SELECT m.name, m.category,
           CASE
               WHEN COALESCE(s.quantity, 0) = 0                      THEN 'out_of_stock'
               WHEN COALESCE(s.quantity, 0) <= COALESCE(s.reorder_point, 0) THEN 'low'
               ELSE 'available'
           END AS status
    FROM materials m
    LEFT JOIN company_store_stock s ON s.material_id = m.id
)
SELECT COUNT(*) FROM stock
WHERE (sqlc.arg(q)::text        = '' OR name ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.arg(category)::text = '' OR category = sqlc.arg(category))
  AND (sqlc.arg(status)::text   = '' OR status   = sqlc.arg(status));

-- name: GetCompanyStock :one
SELECT m.id AS material_id, m.name, m.category, m.unit,
       COALESCE(s.quantity,      0)::numeric AS quantity,
       COALESCE(s.reorder_point, 0)::numeric AS reorder_point,
       CASE
           WHEN COALESCE(s.quantity, 0) = 0                      THEN 'out_of_stock'
           WHEN COALESCE(s.quantity, 0) <= COALESCE(s.reorder_point, 0) THEN 'low'
           ELSE 'available'
       END AS status,
       COALESCE((SELECT SUM(dl.quantity) FROM delivery_lines dl
                 JOIN deliveries d ON d.id = dl.delivery_id
                 WHERE dl.material_id = m.id
                   AND d.sender_type = 'company' AND d.status = 'in_transit'), 0)::numeric AS in_transit
FROM materials m
LEFT JOIN company_store_stock s ON s.material_id = m.id
WHERE m.id = $1;

-- name: AddCompanyStock :one
INSERT INTO company_store_stock (material_id, quantity) VALUES ($1, $2)
ON CONFLICT (material_id) DO UPDATE
SET quantity = company_store_stock.quantity + EXCLUDED.quantity, updated_at = NOW()
RETURNING material_id, quantity, reorder_point, updated_at;

-- name: SetCompanyReorderPoint :one
INSERT INTO company_store_stock (material_id, reorder_point) VALUES ($1, $2)
ON CONFLICT (material_id) DO UPDATE
SET reorder_point = EXCLUDED.reorder_point, updated_at = NOW()
RETURNING material_id, quantity, reorder_point, updated_at;

-- name: DeductCompanyStock :execrows
UPDATE company_store_stock SET quantity = quantity - $2, updated_at = NOW()
WHERE material_id = $1 AND quantity >= $2;
