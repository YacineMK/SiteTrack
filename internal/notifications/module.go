package notifications

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB  *pgxpool.Pool
	Hub *Hub
}

type Module struct {
	handler *Handler
}

func New(deps Dependencies) *Module {
	return &Module{handler: NewHandler(NewService(NewPostgresRepository(deps.DB)), deps.Hub)}
}

func (m *Module) Routes(r chi.Router) {
	m.handler.Routes(r)
}
