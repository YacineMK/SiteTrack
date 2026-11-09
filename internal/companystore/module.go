package companystore

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Checker middleware.PolicyChecker
}

type Module struct {
	handler *Handler
}

func New(deps Dependencies) *Module {
	return &Module{handler: NewHandler(NewService(NewPostgresRepository(deps.DB)), deps.Checker)}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
