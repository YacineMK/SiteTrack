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

	"github.com/YacineMK/SiteTrack/internal/app"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/kit/config"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
	a := app.New(cfg.Port, postgres, redisClient, jwtManager, cfg.CookieSecure)

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := a.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
			stop()
		}
	}()

	log.Printf("listening on %s", cfg.Port)
	<-runCtx.Done()

	log.Println("shutting down")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := a.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
