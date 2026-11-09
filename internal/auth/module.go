package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	Redis        *kitredis.Redis
	JWT          *kitjwt.Manager
	CookieSecure bool
}

type Module struct {
	handler *Handler
	svc     *Service
}

func New(deps Dependencies) *Module {
	repo := NewPostgresRepository(deps.DB)
	svc := NewService(repo, deps.JWT, NewRedisBlacklist(deps.Redis))
	return &Module{
		handler: NewHandler(svc, deps.CookieSecure),
		svc:     svc,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}

func (m *Module) Middleware(next http.Handler) http.Handler {
	return m.svc.Middleware(next)
}
