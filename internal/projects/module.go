package projects

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	ids "github.com/YacineMK/SiteTrack/kit/idgen"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type Dependencies struct {
	DB    *pgxpool.Pool
	Redis *kitredis.Redis
}

type Module struct {
	handler *Handler
	access  middleware.ProjectAccess
}

func New(deps Dependencies) *Module {
	repo := NewPostgresRepository(deps.DB)
	return &Module{
		handler: NewHandler(NewService(repo, ids.New(deps.Redis, deps.DB)), repo),
		access:  repo,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}

func (m *Module) Access() middleware.ProjectAccess {
	return m.access
}
