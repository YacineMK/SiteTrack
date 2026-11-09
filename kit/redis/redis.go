package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *goredis.Client
}

type Config struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func New(ctx context.Context, cfg Config) (*Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &Redis{Client: client}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
