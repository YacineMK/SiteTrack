package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/database/sqlc"
)

type Config struct {
	Concurrency  int
	BatchSize    int32
	PollInterval time.Duration
	LockTimeout  time.Duration
	MaxAttempts  int
}

type Dependencies struct {
	DB     *pgxpool.Pool
	Redis  asynq.RedisClientOpt
	Config Config
}

type Worker struct {
	queries  *sqlc.Queries
	client   *asynq.Client
	server   *asynq.Server
	mux      *asynq.ServeMux
	config   Config
	workerID string
}

func New(deps Dependencies) *Worker {
	return &Worker{
		queries: sqlc.New(deps.DB),
		client:  asynq.NewClient(deps.Redis),
		server: asynq.NewServer(deps.Redis, asynq.Config{
			Concurrency: deps.Config.Concurrency,
			Queues:      map[string]int{QueueName: 1},
		}),
		mux:      asynq.NewServeMux(),
		config:   deps.Config,
		workerID: workerID(),
	}
}

func (w *Worker) Handle(taskType string, handler asynq.HandlerFunc) {
	w.mux.HandleFunc(taskType, handler)
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return err
	}

	go w.dispatch(ctx)

	<-ctx.Done()
	w.server.Shutdown()
	w.client.Close()
	return nil
}
