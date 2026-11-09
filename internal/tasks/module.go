package tasks

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/projects"
	ids "github.com/YacineMK/SiteTrack/kit/idgen"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type Dependencies struct {
	DB    *pgxpool.Pool
	Redis *kitredis.Redis
}

type Module struct {
	handler *Handler
}

func New(deps Dependencies) *Module {
	svc := NewService(
		NewPostgresRepository(deps.DB),
		projects.NewPostgresRepository(deps.DB),
		ids.New(deps.Redis, deps.DB),
	)
	return &Module{handler: NewHandler(svc)}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
