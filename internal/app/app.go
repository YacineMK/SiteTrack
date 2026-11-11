package app

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/auth"
	"github.com/YacineMK/SiteTrack/internal/companystore"
	"github.com/YacineMK/SiteTrack/internal/database"
	"github.com/YacineMK/SiteTrack/internal/deliveries"
	"github.com/YacineMK/SiteTrack/internal/materials"
	"github.com/YacineMK/SiteTrack/internal/middleware"
	"github.com/YacineMK/SiteTrack/internal/notifications"
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
	postgres       *database.Postgres
	redis          *kitredis.Redis
}

func New(addr string, postgres *database.Postgres, redis *kitredis.Redis, jwtManager *kitjwt.Manager, cookieSecure bool, notificationHub *notifications.Hub, inviteTTL time.Duration, appBaseURL string, corsOrigins []string) *App {
	projectsModule := projects.New(projects.Dependencies{DB: postgres.Pool, Redis: redis, InviteTTL: inviteTTL, AppBaseURL: appBaseURL})
	rolesModule := roles.New(roles.Dependencies{DB: postgres.Pool, Access: projectsModule.Access()})

	authModule := auth.New(auth.Dependencies{
		DB:           postgres.Pool,
		Redis:        redis,
		JWT:          jwtManager,
		CookieSecure: cookieSecure,
	})

	a := &App{
		server:         server.New(addr, corsOrigins),
		authMiddleware: middleware.AuthenticateMiddleware(authModule.Service()),
		postgres:       postgres,
		redis:          redis,
		public:         []module{authModule},
		protected: []module{
			materials.New(materials.Dependencies{DB: postgres.Pool}),
			companystore.New(companystore.Dependencies{DB: postgres.Pool, Checker: rolesModule.Checker()}),
			rolesModule,
			projectsModule,
			projectstore.New(projectstore.Dependencies{DB: postgres.Pool, Access: projectsModule.Access()}),
			tasks.New(tasks.Dependencies{DB: postgres.Pool, Redis: redis, Access: projectsModule.Access()}),
			deliveries.New(deliveries.Dependencies{DB: postgres.Pool, Checker: rolesModule.Checker(), Access: projectsModule.Access()}),
			notifications.New(notifications.Dependencies{DB: postgres.Pool, Hub: notificationHub}),
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

func (a *App) Handler() http.Handler {
	return a.server.Mux()
}
