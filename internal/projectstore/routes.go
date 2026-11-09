package projectstore

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Get("/projects/{project_display_id}/store/items", h.list)
	r.Get("/projects/{project_display_id}/store/items/{material_id}", h.get)
}
