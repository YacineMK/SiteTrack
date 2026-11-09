package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedMaterialsSQL = `
INSERT INTO materials (id, name, category, unit)
SELECT gen_random_uuid(),
       'Seed Material ' || g,
       (ARRAY['binder','aggregate','steel','timber','electrical'])[1 + (g % 5)],
       (ARRAY['kg','bag','m','unit'])[1 + (g % 4)]
FROM generate_series(1, $1) g`

func seedMaterials(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedMaterialsSQL, count)
	return err
}
