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
	authz   *auth.AuthZ
	handler *Handler
}

func New(deps Dependencies) *Module {
	svc := NewService(NewPostgresRepository(deps.DB))
	authz := auth.NewAuthZ(svc)
	return &Module{
		authz:   authz,
		handler: NewHandler(svc, authz),
	}
}

func (m *Module) AuthZ() *auth.AuthZ {
	return m.authz
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
