package roles

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

type Dependencies struct {
	DB     *pgxpool.Pool
	Access middleware.ProjectAccess
}

type Module struct {
	handler *Handler
	checker middleware.PolicyChecker
	access  middleware.ProjectAccess
}

func New(deps Dependencies) *Module {
	repo := NewPostgresRepository(deps.DB)
	return &Module{
		handler: NewHandler(NewService(repo), repo),
		checker: repo,
		access:  deps.Access,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r, m.access)
}

func (m *Module) Checker() middleware.PolicyChecker {
	return m.checker
}
