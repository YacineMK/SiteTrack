package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/server"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type module interface {
	Routes(chi.Router)
}

type App struct {
	postgres *database.Postgres
	redis    *kitredis.Redis
	server   *server.Server
	modules  []module
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis, jwtManager *kitjwt.Manager, cookieSecure bool) *App {
	a := &App{
		postgres: postgres,
		redis:    redis,
		server:   server.New(addr),
		modules: []module{
			auth.New(auth.Dependencies{
				DB:           postgres.Pool,
				Redis:        redis,
				JWT:          jwtManager,
				CookieSecure: cookieSecure,
			}),
		},
	}
	a.routes()

	return a
}

func (a *App) Run() error {
	return a.server.Start()
}
