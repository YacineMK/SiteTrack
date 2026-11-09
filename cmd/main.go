package main

import (
	"context"
	"log"
	"time"

	"github.com/YacineMK/SiteTrack/internal/app"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/kit/config"
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

	redis, err := kitredis.New(ctx, kitredis.Config{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPassword,
		DB:              cfg.RedisDB,
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()

	a := app.New(cfg.Port, postgres, redis)

	log.Printf("listening on %s", cfg.Port)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
