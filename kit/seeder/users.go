package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/kit/hash"
)

const seedUsersSQL = `
INSERT INTO users (id, name, email, password_hash, app_role, role_id)
SELECT gen_random_uuid(), 'Seed User ' || g, 'seed.user' || g || '@example.com', $1, 'user',
       (SELECT id FROM app_roles WHERE name = 'user')
FROM generate_series(1, $2) g
ON CONFLICT (email) DO NOTHING`

func seedUsers(ctx context.Context, pool *pgxpool.Pool, count int) error {
	if count <= 0 {
		return nil
	}
	passwordHash, err := hash.Hash("Password123!")
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, seedUsersSQL, passwordHash, count)
	return err
}
