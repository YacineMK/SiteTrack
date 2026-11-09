-- name: CreateDelivery :one
INSERT INTO deliveries (id, sender_type, sender_id, receiver_type, receiver_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at;

-- name: CreateDeliveryLine :exec
INSERT INTO delivery_lines (id, delivery_id, material_id, quantity)
VALUES ($1, $2, $3, $4);

-- name: GetDelivery :one
SELECT id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at
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
RETURNING id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at;

-- name: ListCompanyDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id, d.status, d.sent_at, d.received_at
FROM deliveries d
WHERE (d.sender_type = 'company' OR d.receiver_type = 'company')
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
ORDER BY d.sent_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountCompanyDeliveries :one
SELECT COUNT(*)
FROM deliveries d
WHERE (d.sender_type = 'company' OR d.receiver_type = 'company')
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text);

-- name: ListProjectDeliveries :many
SELECT d.id, d.sender_type, d.sender_id, d.receiver_type, d.receiver_id, d.status, d.sent_at, d.received_at
FROM deliveries d
WHERE ((d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid) OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text)
ORDER BY d.sent_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjectDeliveries :one
SELECT COUNT(*)
FROM deliveries d
WHERE ((d.sender_type = 'project' AND d.sender_id = sqlc.arg(project_id)::uuid) OR (d.receiver_type = 'project' AND d.receiver_id = sqlc.arg(project_id)::uuid))
  AND (sqlc.arg(status_filter)::text = '' OR d.status::text = sqlc.arg(status_filter)::text);
