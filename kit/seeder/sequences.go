package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

func syncSequences(ctx context.Context, pool *pgxpool.Pool, redis *kitredis.Redis) error {
	var lastProject int64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX((regexp_match(display_id, '^PRJ-([0-9]+)$'))[1]::bigint), 0)
		FROM projects`).Scan(&lastProject); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO project_id_counter (id, last_value) VALUES (TRUE, $1)
		ON CONFLICT (id) DO UPDATE SET last_value = GREATEST(project_id_counter.last_value, EXCLUDED.last_value)`, lastProject); err != nil {
		return err
	}
	if err := redis.Client.Set(ctx, "idseq:project", lastProject, 0).Err(); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO task_id_counter (project_id, last_value)
		SELECT project_id, MAX((regexp_match(display_id, '-T-([0-9]+)$'))[1]::bigint)
		FROM tasks
		GROUP BY project_id
		ON CONFLICT (project_id) DO UPDATE
		SET last_value = GREATEST(task_id_counter.last_value, EXCLUDED.last_value)`); err != nil {
		return err
	}

	rows, err := pool.Query(ctx, `
		SELECT project_id, MAX((regexp_match(display_id, '-T-([0-9]+)$'))[1]::bigint)
		FROM tasks
		GROUP BY project_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	pipe := redis.Client.Pipeline()
	for rows.Next() {
		var projectID string
		var value int64
		if err := rows.Scan(&projectID, &value); err != nil {
			return err
		}
		pipe.Set(ctx, "idseq:task:"+projectID, value, 0)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = pipe.Exec(ctx)
	return err
}
