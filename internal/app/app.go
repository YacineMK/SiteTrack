package app

import (
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/server"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type App struct {
	postgres *database.Postgres
	redis    *kitredis.Redis
	server   *server.Server
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis) *App {
	a := &App{
		postgres: postgres,
		redis:    redis,
		server:   server.New(addr),
	}
	a.routes()

	return a
}

func (a *App) Run() error {
	return a.server.Start()
}
