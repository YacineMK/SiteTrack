package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedInvitationsSQL = `
WITH p AS (
    SELECT array_agg(id) AS ids, array_agg(display_id) AS names FROM projects
), u AS (
    SELECT array_agg(id) AS ids FROM users
)
INSERT INTO notifications (id, user_id, type, title, body, data)
SELECT gen_random_uuid(),
       u.ids[1 + FLOOR(RANDOM() * array_length(u.ids, 1))::int],
       'app_invite',
       'Project invitation',
       'You are invited to a project',
       jsonb_build_object(
           'project_id', p.ids[x.pi]::text,
           'project_display_id', p.names[x.pi],
           'role', 'member',
           'expires_at', NOW() + INTERVAL '1 hour'
       )
FROM generate_series(1, $1) g
CROSS JOIN p
CROSS JOIN u
CROSS JOIN LATERAL (
    SELECT 1 + FLOOR(RANDOM() * array_length(p.ids, 1))::int AS pi
) x`

func seedInvitations(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, seedInvitationsSQL, count)
	return err
}
