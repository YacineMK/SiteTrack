package app

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/companystore"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/deliveries"
	"github.com/YacineMK/SiteTrack/internal/materials"
	"github.com/YacineMK/SiteTrack/internal/middleware"
	"github.com/YacineMK/SiteTrack/internal/projects"
	"github.com/YacineMK/SiteTrack/internal/projectstore"
	"github.com/YacineMK/SiteTrack/internal/roles"
	"github.com/YacineMK/SiteTrack/internal/server"
	"github.com/YacineMK/SiteTrack/internal/tasks"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type module interface {
	Routes(chi.Router)
}

type App struct {
	server         *server.Server
	authMiddleware func(http.Handler) http.Handler
	public         []module
	protected      []module
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis, jwtManager *kitjwt.Manager, cookieSecure bool) *App {
	rolesModule := roles.New(roles.Dependencies{DB: postgres.Pool})

	authModule := auth.New(auth.Dependencies{
		DB:           postgres.Pool,
		Redis:        redis,
		JWT:          jwtManager,
		CookieSecure: cookieSecure,
	})

	a := &App{
		server:         server.New(addr),
		authMiddleware: middleware.AuthenticateMiddleware(authModule.Service()),
		public:         []module{authModule},
		protected: []module{
			materials.New(materials.Dependencies{DB: postgres.Pool}),
			companystore.New(companystore.Dependencies{DB: postgres.Pool}),
			rolesModule,
			projects.New(projects.Dependencies{DB: postgres.Pool, Redis: redis}),
			projectstore.New(projectstore.Dependencies{DB: postgres.Pool}),
			tasks.New(tasks.Dependencies{DB: postgres.Pool, Redis: redis}),
			deliveries.New(deliveries.Dependencies{DB: postgres.Pool}),
		},
	}
	a.routes()

	return a
}

func (a *App) Run() error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
