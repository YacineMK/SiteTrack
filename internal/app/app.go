package app

import (
	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/server"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type App struct {
	postgres *database.Postgres
	redis    *kitredis.Redis
	server   *server.Server
	auth     *auth.Handler
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis, jwtManager *kitjwt.Manager) *App {
	authRepo := auth.NewPostgresRepository(postgres.Pool)
	authService := auth.NewService(authRepo, jwtManager, auth.NewRedisBlacklist(redis))

	a := &App{
		postgres: postgres,
		redis:    redis,
		server:   server.New(addr),
		auth:     auth.NewHandler(authService),
	}
	a.routes()

	return a
}

func (a *App) Run() error {
	return a.server.Start()
}
