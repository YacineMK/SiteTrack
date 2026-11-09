package materials

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Route("/materials", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
		r.Get("/{material_id}", h.get)
		r.Patch("/{material_id}", h.update)
		r.Delete("/{material_id}", h.delete)
	})
}
