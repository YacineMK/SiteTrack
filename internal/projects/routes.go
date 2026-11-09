package projects

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Route("/projects", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
		r.Get("/search", h.search)
		r.Get("/{display_id}", h.get)
		r.Patch("/{display_id}", h.update)
		r.Delete("/{display_id}", h.delete)
	})
}
