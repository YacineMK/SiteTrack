package roles

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB *pgxpool.Pool
}

type Module struct {
	handler *Handler
}

func New(deps Dependencies) *Module {
	svc := NewService(NewPostgresRepository(deps.DB))
	return &Module{
		handler: NewHandler(svc),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
