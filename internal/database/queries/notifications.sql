-- name: CreateNotification :one
INSERT INTO notifications (id, user_id, type, title, body, data)
VALUES ($1, $2, $3, $4, $5, sqlc.arg(data)::jsonb)
RETURNING id, user_id, type, title, body, data, read_at, created_at;

-- name: GetNotification :one
SELECT id, user_id, type, title, body, data, read_at, created_at
FROM notifications
WHERE id = $1;

-- name: ListNotifications :many
SELECT id, user_id, type, title, body, data, read_at, created_at
FROM notifications
WHERE user_id = $1
  AND (sqlc.arg(unread_only)::boolean = false OR read_at IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountNotifications :one
SELECT COUNT(*)
FROM notifications
WHERE user_id = $1
  AND (sqlc.arg(unread_only)::boolean = false OR read_at IS NULL);

-- name: MarkNotificationRead :one
UPDATE notifications
SET read_at = COALESCE(read_at, NOW())
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, type, title, body, data, read_at, created_at;

-- name: LockNotification :one
SELECT id, user_id, type, title, body, data, read_at, created_at
FROM notifications
WHERE id = $1
FOR UPDATE;

-- name: DeleteNotification :execrows
DELETE FROM notifications WHERE id = $1;
