package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/companystore"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/roles"
	"github.com/YacineMK/SiteTrack/internal/server"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type module interface {
	Routes(chi.Router)
}

type App struct {
	postgres       *database.Postgres
	redis          *kitredis.Redis
	server         *server.Server
	authMiddleware func(http.Handler) http.Handler
	public         []module
	protected      []module
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis, jwtManager *kitjwt.Manager, cookieSecure bool) *App {
	rolesModule := roles.New(roles.Dependencies{DB: postgres.Pool})
	authz := auth.NewAuthZ(rolesModule.Service())

	authModule := auth.New(auth.Dependencies{
		DB:           postgres.Pool,
		Redis:        redis,
		JWT:          jwtManager,
		CookieSecure: cookieSecure,
	})

	a := &App{
		postgres:       postgres,
		redis:          redis,
		server:         server.New(addr),
		authMiddleware: authModule.Middleware,
		public:         []module{authModule},
		protected: []module{
			companystore.New(companystore.Dependencies{AuthZ: authz}),
			rolesModule,
		},
	}
	a.routes()

	return a
}

func (a *App) Run() error {
	return a.server.Start()
}
