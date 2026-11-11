//go:build e2e

// Package e2e boots Postgres + Redis in Docker (via ory/dockertest), applies
// the migrations, and exercises every endpoint of the real HTTP handler built
// by internal/app.
//
// Run:    go test -tags=e2e ./test/e2e -count=1
package e2e

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/YacineMK/SiteTrack/internal/app"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/notifications"
	"github.com/YacineMK/SiteTrack/internal/worker"
	"github.com/YacineMK/SiteTrack/kit/config"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
	kitresend "github.com/YacineMK/SiteTrack/kit/resend"
)

var (
	ts   *httptest.Server
	cfg  config.Config
	dbpool *database.Postgres
)

const (
	pgPassword    = "sitetrack"
	redisPassword = "sitetrack"
)

func TestMain(m *testing.M) {
	// Disable rate limiting for e2e. The test suite shoots requests at a rate
	// the production 5/10s auth bucket cannot tolerate.
	os.Setenv("RATE_LIMIT_OVERRIDE", "100000")

	code, err := run(m)
	if err != nil {
		log.Printf("e2e setup: %v", err)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return 1, fmt.Errorf("dockertest pool: %w", err)
	}
	pool.MaxWait = 60 * time.Second

	pgRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17-alpine",
		Env: []string{
			"POSTGRES_USER=sitetrack",
			"POSTGRES_PASSWORD=" + pgPassword,
			"POSTGRES_DB=sitetrack_test",
		},
	}, func(c *docker.HostConfig) {
		c.AutoRemove = true
		c.RestartPolicy = docker.RestartPolicy{Name: "no"}
		c.Tmpfs = map[string]string{"/var/lib/postgresql/data": "rw"}
	})
	if err != nil {
		return 1, fmt.Errorf("start postgres: %w", err)
	}
	defer pool.Purge(pgRes)

	redisRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "7-alpine",
		Cmd:        []string{"redis-server", "--requirepass", redisPassword},
	}, func(c *docker.HostConfig) { c.AutoRemove = true })
	if err != nil {
		return 1, fmt.Errorf("start redis: %w", err)
	}
	defer pool.Purge(redisRes)

	dbURL := fmt.Sprintf("postgres://sitetrack:%s@localhost:%s/sitetrack_test?sslmode=disable",
		pgPassword, pgRes.GetPort("5432/tcp"))
	redisAddr := "localhost:" + redisRes.GetPort("6379/tcp")

	if err := waitForPostgres(dbURL, pool); err != nil {
		return 1, err
	}
	if err := waitForRedis(redisAddr); err != nil {
		return 1, err
	}
	if err := applyMigrations(dbURL); err != nil {
		return 1, fmt.Errorf("migrations: %w", err)
	}

	cfg = config.Config{
		JWTSecret:          "e2e-secret",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		CookieSecure:       false,
		InviteTTL:          time.Hour,
		AppBaseURL:         "http://localhost",
		WorkerConcurrency:  1,
		OutboxBatchSize:    5,
		OutboxPollInterval: 500 * time.Millisecond,
		OutboxLockTimeout:  time.Minute,
		OutboxMaxAttempts:  3,
		CORSAllowedOrigins: []string{"*"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pg, err := database.NewPostgres(ctx, dbURL, database.PoolConfig{MaxConns: 5, MinConns: 1})
	if err != nil {
		return 1, fmt.Errorf("pgxpool: %w", err)
	}
	defer pg.Close()
	dbpool = pg

	rc, err := kitredis.New(ctx, kitredis.Config{Addr: redisAddr, Password: redisPassword, PoolSize: 5})
	if err != nil {
		return 1, fmt.Errorf("redis: %w", err)
	}
	defer rc.Close()
	_ = rc.Client.FlushDB(ctx).Err()

	jwtManager := kitjwt.New(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	hub := notifications.NewHub()
	a := app.New("", pg, rc, jwtManager, cfg.CookieSecure, hub, cfg.InviteTTL, cfg.AppBaseURL, cfg.CORSAllowedOrigins)

	runCtx, stop := context.WithCancel(context.Background())
	defer stop()
	mail := kitresend.New("", "", "")
	w := worker.New(worker.Dependencies{
		DB:    pg.Pool,
		Redis: asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword},
		Config: worker.Config{
			Concurrency:  cfg.WorkerConcurrency,
			BatchSize:    int32(cfg.OutboxBatchSize),
			PollInterval: cfg.OutboxPollInterval,
			LockTimeout:  cfg.OutboxLockTimeout,
			MaxAttempts:  cfg.OutboxMaxAttempts,
		},
	})
	w.Handle(worker.TaskEmail, worker.NewEmailHandler(pg.Pool, mail).Handle)
	w.Handle(worker.TaskAppNotification, worker.NewAppNotificationHandler(pg.Pool, hub).Handle)
	go func() {
		if err := w.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("worker exited: %v", err)
		}
	}()

	ts = httptest.NewServer(a.Handler())
	defer ts.Close()

	return m.Run(), nil
}

func waitForPostgres(dbURL string, pool *dockertest.Pool) error {
	return pool.Retry(func() error {
		conn, err := pgx.Connect(context.Background(), dbURL)
		if err != nil {
			return err
		}
		defer conn.Close(context.Background())
		return conn.Ping(context.Background())
	})
}

func waitForRedis(addr string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		rc, err := kitredis.New(context.Background(), kitredis.Config{Addr: addr, Password: redisPassword})
		if err == nil {
			rc.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("redis never became ready")
}

func applyMigrations(dbURL string) error {
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	dir := filepath.Join(repoRoot(), "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		raw, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		up := extractUp(string(raw))
		if _, err := conn.Exec(context.Background(), up); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	return nil
}

// extractUp returns the SQL between `-- migrate:up` and `-- migrate:down`.
func extractUp(sql string) string {
	const upMarker = "-- migrate:up"
	const downMarker = "-- migrate:down"
	i := strings.Index(sql, upMarker)
	if i < 0 {
		return sql
	}
	rest := sql[i+len(upMarker):]
	if j := strings.Index(rest, downMarker); j >= 0 {
		return rest[:j]
	}
	return rest
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// test/e2e → repo root
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// mustStatus fails the test if the response status doesn't match.
func mustStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		b := make([]byte, 2048)
		n, _ := resp.Body.Read(b)
		t.Fatalf("status = %d, want %d: %s", resp.StatusCode, want, string(b[:n]))
	}
}
