package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/YacineMK/SiteTrack/internal/app"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/notifications"
	"github.com/YacineMK/SiteTrack/internal/worker"
	"github.com/YacineMK/SiteTrack/kit/config"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	"github.com/YacineMK/SiteTrack/kit/observability"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
	kitresend "github.com/YacineMK/SiteTrack/kit/resend"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownOtel, err := observability.Setup(ctx, "sitetrack-api", os.Getenv("APP_ENV"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownOtel(shutdownCtx)
	}()

	postgres, err := database.NewPostgres(ctx, cfg.DatabaseURL, database.PoolConfig{
		MaxConns:        cfg.DBMaxConns,
		MinConns:        cfg.DBMinConns,
		MaxConnLifetime: cfg.DBMaxConnLifetime,
		MaxConnIdleTime: cfg.DBMaxConnIdleTime,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer postgres.Close()


	redisClient, err := kitredis.New(ctx, kitredis.Config{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPassword,
		DB:              cfg.RedisDB,
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
	})
	if err != nil {
		postgres.Close()
		log.Fatal(err)
	}
	defer redisClient.Close()

	jwtManager := kitjwt.New(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	notificationHub := notifications.NewHub()
	a := app.New(cfg.Port, postgres, redisClient, jwtManager, cfg.CookieSecure, notificationHub, cfg.InviteTTL, cfg.AppBaseURL, cfg.CORSAllowedOrigins)

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mailClient := kitresend.New(cfg.ResendAPIKey, cfg.ResendFrom, cfg.ResendBaseURL)

	outboxWorker := worker.New(worker.Dependencies{
		DB: postgres.Pool,
		Redis: asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		},
		Config: worker.Config{
			Concurrency:  cfg.WorkerConcurrency,
			BatchSize:    int32(cfg.OutboxBatchSize),
			PollInterval: cfg.OutboxPollInterval,
			LockTimeout:  cfg.OutboxLockTimeout,
			MaxAttempts:  cfg.OutboxMaxAttempts,
		},
	})
	outboxWorker.Handle(worker.TaskEmail, worker.NewEmailHandler(postgres.Pool, mailClient).Handle)
	outboxWorker.Handle(worker.TaskAppNotification, worker.NewAppNotificationHandler(postgres.Pool, notificationHub).Handle)

	go func() {
		if err := outboxWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			stop()
		}
	}()

	go func() {
		if err := a.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
		}
	}()

	<-runCtx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	a.Shutdown(shutdownCtx)
}
