package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedDeliveriesSQL = `
WITH p AS (
    SELECT array_agg(id) AS ids FROM projects
), u AS (
    SELECT array_agg(id) AS ids FROM users
)
INSERT INTO deliveries (id, sender_type, sender_id, receiver_type, receiver_id, status, sent_at, received_at, sent_by, received_by)
SELECT gen_random_uuid(),
       (CASE WHEN d.company_sender THEN 'company' ELSE 'project' END)::delivery_party,
       CASE WHEN d.company_sender THEN NULL ELSE d.project_id END,
       (CASE WHEN d.company_sender THEN 'project' ELSE 'company' END)::delivery_party,
       CASE WHEN d.company_sender THEN d.project_id ELSE NULL END,
       d.status::delivery_status,
       d.sent_at,
       CASE WHEN d.status = 'in_transit' THEN NULL ELSE d.sent_at + INTERVAL '2 days' END,
       u.ids[1 + FLOOR(RANDOM() * array_length(u.ids, 1))::int],
       CASE WHEN d.status = 'in_transit' THEN NULL
            ELSE u.ids[1 + FLOOR(RANDOM() * array_length(u.ids, 1))::int] END
FROM generate_series(1, $1) g
CROSS JOIN p
CROSS JOIN u
CROSS JOIN LATERAL (
    SELECT RANDOM() < 0.7 AS company_sender,
           p.ids[1 + FLOOR(RANDOM() * array_length(p.ids, 1))::int] AS project_id,
           (ARRAY['in_transit','delivered','rejected','cancelled'])[1 + FLOOR(RANDOM() * 4)::int] AS status,
           NOW() - (RANDOM() * 365)::int * INTERVAL '1 day' AS sent_at
) d`

func seedDeliveries(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedDeliveriesSQL, count)
	return err
}
