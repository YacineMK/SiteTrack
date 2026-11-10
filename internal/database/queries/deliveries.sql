-- name: CreateDelivery :one
INSERT INTO deliveries (id, sender_type, sender_id, receiver_type, receiver_id, sent_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: CreateDeliveryLine :exec
INSERT INTO delivery_lines (id, delivery_id, material_id, quantity) VALUES ($1, $2, $3, $4);

-- name: GetDelivery :one
SELECT id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by
FROM deliveries WHERE id = $1;

-- name: GetDeliveryLines :many
SELECT dl.delivery_id, dl.material_id, m.name, m.unit, dl.quantity
FROM delivery_lines dl JOIN materials m ON m.id = dl.material_id
WHERE dl.delivery_id = ANY($1::uuid[])
ORDER BY dl.delivery_id, m.name;

-- name: UpdateDeliveryStatus :one
UPDATE deliveries SET status = $2, received_at = NOW()
WHERE id = $1 AND status = 'in_transit'
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: CompleteDeliveryStatus :one
UPDATE deliveries SET status = $2, received_at = NOW(), received_by = $3
WHERE id = $1 AND status = 'in_transit'
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by;

-- name: ListCompanyDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id,
       d.status, d.sent_at, d.received_at, d.sent_by, d.received_by,
       sp.display_id AS sender_display_id,
       rp.display_id AS receiver_display_id,
       su.name       AS sent_by_name,
       ru.name       AS received_by_name
FROM deliveries d
LEFT JOIN projects sp ON sp.id = d.sender_id
LEFT JOIN projects rp ON rp.id = d.receiver_id
LEFT JOIN users    su ON su.id = d.sent_by
LEFT JOIN users    ru ON ru.id = d.received_by
WHERE (d.sender_type = 'company' OR d.receiver_type = 'company')
  AND (sqlc.narg(project_id)::uuid IS NULL
       OR d.sender_id = sqlc.narg(project_id) OR d.receiver_id = sqlc.narg(project_id))
  AND (sqlc.arg(direction)::text = ''
       OR (sqlc.arg(direction) = 'out' AND d.sender_type   = 'company')
       OR (sqlc.arg(direction) = 'in'  AND d.receiver_type = 'company'))
  AND (sqlc.arg(status)::text   = '' OR d.status::text = sqlc.arg(status))
  AND (sqlc.narg(from_date)::date IS NULL OR d.sent_at >=  sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date   IS NULL OR d.sent_at <   sqlc.narg(to_date) + 1)
  AND (sqlc.narg(after_sent_at)::timestamptz IS NULL
       OR (d.sent_at, d.id) < (sqlc.narg(after_sent_at), sqlc.narg(after_id)::uuid))
ORDER BY d.sent_at DESC, d.id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListProjectDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id,
       d.status, d.sent_at, d.received_at, d.sent_by, d.received_by,
       sp.display_id AS sender_display_id,
       rp.display_id AS receiver_display_id,
       su.name       AS sent_by_name,
       ru.name       AS received_by_name
FROM deliveries d
LEFT JOIN projects sp ON sp.id = d.sender_id
LEFT JOIN projects rp ON rp.id = d.receiver_id
LEFT JOIN users    su ON su.id = d.sent_by
LEFT JOIN users    ru ON ru.id = d.received_by
WHERE (d.sender_id = sqlc.arg(project_id)::uuid OR d.receiver_id = sqlc.arg(project_id)::uuid)
  AND (sqlc.arg(direction)::text = ''
       OR (sqlc.arg(direction) = 'out' AND d.sender_id   = sqlc.arg(project_id)::uuid)
       OR (sqlc.arg(direction) = 'in'  AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status)::text   = '' OR d.status::text = sqlc.arg(status))
  AND (sqlc.narg(from_date)::date IS NULL OR d.sent_at >=  sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date   IS NULL OR d.sent_at <   sqlc.narg(to_date) + 1)
  AND (sqlc.narg(after_sent_at)::timestamptz IS NULL
       OR (d.sent_at, d.id) < (sqlc.narg(after_sent_at), sqlc.narg(after_id)::uuid))
ORDER BY d.sent_at DESC, d.id DESC
LIMIT sqlc.arg(page_limit);
