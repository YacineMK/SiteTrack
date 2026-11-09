-- name: CreateDelivery :one
INSERT INTO deliveries (id, sender_type, sender_id, receiver_type, receiver_id, sent_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: CreateDeliveryLine :exec
INSERT INTO delivery_lines (id, delivery_id, material_id, quantity)
VALUES ($1, $2, $3, $4);

-- name: GetDelivery :one
SELECT id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by
FROM deliveries
WHERE id = $1;

-- name: GetDeliveryLines :many
SELECT dl.material_id, m.name, m.unit, dl.quantity
FROM delivery_lines dl
JOIN materials m ON m.id = dl.material_id
WHERE dl.delivery_id = $1
ORDER BY m.name;

-- name: UpdateDeliveryStatus :one
UPDATE deliveries
SET status = $2,
    received_at = NOW()
WHERE id = $1 AND status = 'in_transit'
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: CompleteDeliveryStatus :one
UPDATE deliveries
SET status = $2,
    received_at = NOW(),
    received_by = $3
WHERE id = $1 AND status = 'in_transit'
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: ListCompanyDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id, d.status, d.sent_at, d.received_at, d.sent_by, d.received_by,
       COALESCE(su.name, '')::text AS sent_by_name,
       COALESCE(ru.name, '')::text AS received_by_name
FROM deliveries d
LEFT JOIN users su ON su.id = d.sent_by
LEFT JOIN users ru ON ru.id = d.received_by
WHERE (d.sender_type = 'company' OR d.receiver_type = 'company')
  AND (sqlc.arg(direction)::text = ''
    OR (sqlc.arg(direction)::text = 'out' AND d.sender_type = 'company')
    OR (sqlc.arg(direction)::text = 'in' AND d.receiver_type = 'company'))
  AND (sqlc.arg(project_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid
    OR (d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid)
    OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR d.sent_at >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR d.sent_at < NULLIF(sqlc.arg(to_date)::text, '')::date + 1)
  AND (sqlc.arg(query)::text = '' OR d.id::text ILIKE '%' || sqlc.arg(query)::text || '%' OR EXISTS (SELECT 1 FROM delivery_lines dl JOIN materials m ON m.id = dl.material_id WHERE dl.delivery_id = d.id AND m.name ILIKE '%' || sqlc.arg(query)::text || '%') OR EXISTS (SELECT 1 FROM projects p WHERE (p.id = d.sender_id OR p.id = d.receiver_id) AND p.display_id ILIKE '%' || sqlc.arg(query)::text || '%'))
ORDER BY d.sent_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountCompanyDeliveries :one
SELECT COUNT(*)
FROM deliveries d
WHERE (d.sender_type = 'company' OR d.receiver_type = 'company')
  AND (sqlc.arg(direction)::text = ''
    OR (sqlc.arg(direction)::text = 'out' AND d.sender_type = 'company')
    OR (sqlc.arg(direction)::text = 'in' AND d.receiver_type = 'company'))
  AND (sqlc.arg(project_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid
    OR (d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid)
    OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR d.sent_at >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR d.sent_at < NULLIF(sqlc.arg(to_date)::text, '')::date + 1)
  AND (sqlc.arg(query)::text = '' OR d.id::text ILIKE '%' || sqlc.arg(query)::text || '%' OR EXISTS (SELECT 1 FROM delivery_lines dl JOIN materials m ON m.id = dl.material_id WHERE dl.delivery_id = d.id AND m.name ILIKE '%' || sqlc.arg(query)::text || '%') OR EXISTS (SELECT 1 FROM projects p WHERE (p.id = d.sender_id OR p.id = d.receiver_id) AND p.display_id ILIKE '%' || sqlc.arg(query)::text || '%'));

-- name: ListProjectDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id, d.status, d.sent_at, d.received_at, d.sent_by, d.received_by,
       COALESCE(su.name, '')::text AS sent_by_name,
       COALESCE(ru.name, '')::text AS received_by_name
FROM deliveries d
LEFT JOIN users su ON su.id = d.sent_by
LEFT JOIN users ru ON ru.id = d.received_by
WHERE ((d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid) OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(direction)::text = ''
    OR (sqlc.arg(direction)::text = 'out' AND d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid)
    OR (sqlc.arg(direction)::text = 'in' AND d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR d.sent_at >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR d.sent_at < NULLIF(sqlc.arg(to_date)::text, '')::date + 1)
  AND (sqlc.arg(query)::text = '' OR d.id::text ILIKE '%' || sqlc.arg(query)::text || '%' OR EXISTS (SELECT 1 FROM delivery_lines dl JOIN materials m ON m.id = dl.material_id WHERE dl.delivery_id = d.id AND m.name ILIKE '%' || sqlc.arg(query)::text || '%') OR EXISTS (SELECT 1 FROM projects p WHERE (p.id = d.sender_id OR p.id = d.receiver_id) AND p.display_id ILIKE '%' || sqlc.arg(query)::text || '%'))
ORDER BY d.sent_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjectDeliveries :one
SELECT COUNT(*)
FROM deliveries d
WHERE ((d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid) OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(direction)::text = ''
    OR (sqlc.arg(direction)::text = 'out' AND d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid)
    OR (sqlc.arg(direction)::text = 'in' AND d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
  AND (NULLIF(sqlc.arg(from_date)::text, '')::date IS NULL OR d.sent_at >= NULLIF(sqlc.arg(from_date)::text, '')::date)
  AND (NULLIF(sqlc.arg(to_date)::text, '')::date IS NULL OR d.sent_at < NULLIF(sqlc.arg(to_date)::text, '')::date + 1)
  AND (sqlc.arg(query)::text = '' OR d.id::text ILIKE '%' || sqlc.arg(query)::text || '%' OR EXISTS (SELECT 1 FROM delivery_lines dl JOIN materials m ON m.id = dl.material_id WHERE dl.delivery_id = d.id AND m.name ILIKE '%' || sqlc.arg(query)::text || '%') OR EXISTS (SELECT 1 FROM projects p WHERE (p.id = d.sender_id OR p.id = d.receiver_id) AND p.display_id ILIKE '%' || sqlc.arg(query)::text || '%'));
