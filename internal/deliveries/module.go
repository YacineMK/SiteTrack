package deliveries

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	"github.com/YacineMK/SiteTrack/internal/projects"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Checker middleware.PolicyChecker
	Access  middleware.ProjectAccess
}

type Module struct {
	handler *Handler
}

func New(deps Dependencies) *Module {
	svc := NewService(NewPostgresRepository(deps.DB), projects.NewPostgresRepository(deps.DB))
	return &Module{handler: NewHandler(svc, deps.Checker, deps.Access)}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
