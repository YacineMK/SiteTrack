package roles

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

type Dependencies struct {
	DB *pgxpool.Pool
}

type Module struct {
	handler *Handler
	checker middleware.PolicyChecker
}

func New(deps Dependencies) *Module {
	repo := NewPostgresRepository(deps.DB)
	return &Module{
		handler: NewHandler(NewService(repo), repo),
		checker: repo,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}

func (m *Module) Checker() middleware.PolicyChecker {
	return m.checker
}
