package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedProjectCreatorsSQL = `
WITH p AS (
    SELECT id, (row_number() OVER (ORDER BY display_id) - 1) AS rn, COUNT(*) OVER () AS n FROM projects
), u AS (
    SELECT id, (row_number() OVER (ORDER BY email) - 1) AS rn, COUNT(*) OVER () AS n FROM users
)
INSERT INTO project_members (project_id, user_id, is_admin)
SELECT p.id, u.id, TRUE
FROM p JOIN u ON u.rn = p.rn % u.n
ON CONFLICT DO NOTHING`

const seedProjectMembersSQL = `
WITH p AS (
    SELECT id, (row_number() OVER (ORDER BY display_id) - 1) AS rn FROM projects
), u AS (
    SELECT id, (row_number() OVER (ORDER BY email) - 1) AS rn, COUNT(*) OVER () AS n FROM users
)
INSERT INTO project_members (project_id, user_id, is_admin)
SELECT p.id, u.id, FALSE
FROM p
CROSS JOIN generate_series(1, $1) s
JOIN u ON u.rn = (p.rn * 37 + s * 13) % u.n
ON CONFLICT DO NOTHING`

func seedProjectMembers(ctx context.Context, pool *pgxpool.Pool, maxMembers int) error {
	if maxMembers <= 0 {
		return nil
	}
	if _, err := pool.Exec(ctx, seedProjectCreatorsSQL); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, seedProjectMembersSQL, maxMembers)
	return err
}
