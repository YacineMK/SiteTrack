package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedTasksSQL = `
WITH p AS (
    SELECT id, display_id,
           (row_number() OVER (ORDER BY display_id) - 1) AS rn,
           COUNT(*) OVER () AS n
    FROM projects
), g AS (
    SELECT g, FLOOR((g - 1) * p.n / $1)::int AS prn
    FROM generate_series(1, $1) g
    CROSS JOIN (SELECT COUNT(*) AS n FROM projects) p
)
INSERT INTO tasks (id, display_id, project_id, name, planned_start_date, planned_end_date, status)
SELECT gen_random_uuid(),
       p.display_id || '-T-' || lpad((row_number() OVER (PARTITION BY p.id ORDER BY g.g))::text, 4, '0'),
       p.id,
       'Seed Task ' || g.g,
       DATE '2024-01-01' + (RANDOM() * 1000)::int,
       DATE '2024-06-01' + (RANDOM() * 700)::int,
       (ARRAY['not_started','in_progress','done'])[1 + FLOOR(RANDOM() * 3)::int]
FROM g JOIN p ON p.rn = g.prn`

func seedTasks(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedTasksSQL, count)
	return err
}
