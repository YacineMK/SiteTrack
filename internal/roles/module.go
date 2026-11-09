package roles

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YacineMK/SiteTrack/internal/auth"
)

type Dependencies struct {
	DB *pgxpool.Pool
}

type Module struct {
	svc     *Service
	handler *Handler
}

func New(deps Dependencies) *Module {
	svc := NewService(NewPostgresRepository(deps.DB))
	return &Module{
		svc:     svc,
		handler: NewHandler(svc, auth.NewAuthZ(svc)),
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
