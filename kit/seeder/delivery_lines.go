package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedDeliveryLinesSQL = `
WITH d AS (
    SELECT id, row_number() OVER (ORDER BY id) AS rn FROM deliveries
), m AS (
    SELECT array_agg(id) AS ids FROM materials
)
INSERT INTO delivery_lines (id, delivery_id, material_id, quantity)
SELECT gen_random_uuid(),
       d.id,
       m.ids[1 + ((d.rn * 7 + s) % array_length(m.ids, 1))::int],
       (RANDOM() * 100)::numeric(14,3)
FROM d
CROSS JOIN m
CROSS JOIN LATERAL generate_series(1, ($1::int / $2::int) + CASE WHEN d.rn <= ($1::int % $2::int) THEN 1 ELSE 0 END) s`

func seedDeliveryLines(ctx context.Context, pool *pgxpool.Pool, lines, deliveries int) error {
	if lines <= 0 || deliveries <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedDeliveryLinesSQL, lines, deliveries)
	return err
}
