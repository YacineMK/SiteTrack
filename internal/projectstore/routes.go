package projectstore

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.With(middleware.RequireProjectMember(h.access)).Get("/projects/{project_display_id}/store/items", h.list)
	r.With(middleware.RequireProjectMember(h.access)).Get("/projects/{project_display_id}/store/items/{material_id}", h.get)
}
