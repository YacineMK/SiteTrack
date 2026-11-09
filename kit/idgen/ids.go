package ids

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

const (
	projectKey    = "idseq:project"
	taskKeyPrefix = "idseq:task:"
)

const nextScript = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local floor = tonumber(ARGV[1])
if current < floor then
  current = floor
end
current = current + 1
redis.call('SET', KEYS[1], current)
return current
`

type ID struct {
	UUID      uuid.UUID
	DisplayID string
}

type Generator struct {
	redis *kitredis.Redis
	db    *pgxpool.Pool
}

func New(redis *kitredis.Redis, db *pgxpool.Pool) *Generator {
	return &Generator{redis: redis, db: db}
}

func (g *Generator) NewProject(ctx context.Context) (ID, error) {
	seq, err := g.nextSequence(ctx)
	if err != nil {
		return ID{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}

	if err := g.persistSequence(ctx, seq); err != nil {
		return ID{}, err
	}

	return ID{UUID: id, DisplayID: ProjectDisplayID(uint64(seq))}, nil
}

func (g *Generator) NewTask(ctx context.Context, projectID uuid.UUID, projectDisplayID string) (ID, error) {
	seq, err := g.nextTaskSequence(ctx, projectID)
	if err != nil {
		return ID{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}

	if err := g.persistTaskSequence(ctx, projectID, seq); err != nil {
		return ID{}, err
	}

	return ID{UUID: id, DisplayID: TaskDisplayID(projectDisplayID, uint64(seq))}, nil
}

func (g *Generator) nextSequence(ctx context.Context) (int64, error) {
	exists, err := g.redis.Client.Exists(ctx, projectKey).Result()
	if err != nil {
		return 0, err
	}
	if exists > 0 {
		return g.redis.Client.Incr(ctx, projectKey).Result()
	}

	var floor int64
	err = g.db.QueryRow(ctx, "SELECT last_value FROM project_id_counter WHERE id = TRUE").Scan(&floor)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	return g.redis.Client.Eval(ctx, nextScript, []string{projectKey}, floor).Int64()
}

func (g *Generator) persistSequence(ctx context.Context, seq int64) error {
	_, err := g.db.Exec(ctx, `
INSERT INTO project_id_counter (id, last_value)
VALUES (TRUE, $1)
ON CONFLICT (id) DO UPDATE SET last_value = GREATEST(project_id_counter.last_value, EXCLUDED.last_value)
`, seq)
	return err
}

func (g *Generator) nextTaskSequence(ctx context.Context, projectID uuid.UUID) (int64, error) {
	key := taskKeyPrefix + projectID.String()

	exists, err := g.redis.Client.Exists(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if exists > 0 {
		return g.redis.Client.Incr(ctx, key).Result()
	}

	var floor int64
	err = g.db.QueryRow(ctx, "SELECT last_value FROM task_id_counter WHERE project_id = $1", projectID).Scan(&floor)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	return g.redis.Client.Eval(ctx, nextScript, []string{key}, floor).Int64()
}

func (g *Generator) persistTaskSequence(ctx context.Context, projectID uuid.UUID, seq int64) error {
	_, err := g.db.Exec(ctx, `
INSERT INTO task_id_counter (project_id, last_value)
VALUES ($1, $2)
ON CONFLICT (project_id) DO UPDATE SET last_value = GREATEST(task_id_counter.last_value, EXCLUDED.last_value)
`, projectID, seq)
	return err
}

func ProjectDisplayID(seq uint64) string {
	return fmt.Sprintf("PRJ-%04d", seq)
}

func TaskDisplayID(projectDisplayID string, seq uint64) string {
	return fmt.Sprintf("%s-T-%04d", projectDisplayID, seq)
}
