package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedProjectsSQL = `
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
SELECT gen_random_uuid(),
       'PRJ-' || lpad(g::text, 4, '0'),
       'Seed Project ' || g,
       'Seeded project ' || g,
       DATE '2024-01-01' + ((g * 7) % 900),
       DATE '2025-01-01' + ((g * 11) % 700)
FROM generate_series(1, $1) g`

func seedProjects(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedProjectsSQL, count)
	return err
}
