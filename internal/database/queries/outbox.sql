-- name: CreateOutboxEvent :exec
INSERT INTO outbox (id, kind, payload)
VALUES ($1, $2, sqlc.arg(payload)::jsonb);

-- name: ClaimOutboxBatch :many
UPDATE outbox
SET status = 'processing',
    locked_at = NOW(),
    locked_by = sqlc.arg(worker_id)::text,
    attempts = attempts + 1
WHERE id IN (
    SELECT id FROM outbox
    WHERE (status = 'pending' AND next_attempt_at <= NOW())
       OR (status = 'processing' AND locked_at < NOW() - make_interval(secs => sqlc.arg(lock_seconds)::int))
    ORDER BY next_attempt_at
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
)
RETURNING id, kind, payload, status, attempts, next_attempt_at, locked_at, locked_by, last_error, created_at, processed_at;

-- name: MarkOutboxEnqueued :execrows
UPDATE outbox
SET status = 'enqueued',
    last_error = NULL
WHERE id = $1 AND status = 'processing';

-- name: CompleteOutbox :execrows
UPDATE outbox
SET status = 'done',
    processed_at = NOW(),
    last_error = NULL,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;

-- name: FailOutbox :execrows
UPDATE outbox
SET status = sqlc.arg(status),
    next_attempt_at = sqlc.arg(next_attempt_at),
    last_error = sqlc.arg(last_error)::text,
    locked_at = NULL,
    locked_by = NULL
WHERE id = $1;
