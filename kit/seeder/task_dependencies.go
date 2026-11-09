package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedTaskDependenciesSQL = `
WITH ranked AS (
    SELECT id, project_id,
           row_number() OVER (PARTITION BY project_id ORDER BY display_id) AS rn
    FROM tasks
), edges AS (
    SELECT t.id AS task_id, p.id AS depends_on
    FROM ranked t
    JOIN ranked p ON p.project_id = t.project_id AND p.rn = t.rn - 1
    WHERE t.rn > 1
    UNION ALL
    SELECT t.id, p.id
    FROM ranked t
    JOIN ranked p ON p.project_id = t.project_id AND p.rn = t.rn - 2
    WHERE t.rn > 2
)
INSERT INTO task_dependencies (task_id, depends_on)
SELECT task_id, depends_on FROM edges LIMIT $1`

func seedTaskDependencies(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedTaskDependenciesSQL, count)
	return err
}
