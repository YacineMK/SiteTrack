package roles

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(middleware.AuthorizationMiddleware("super_admin"))

		r.Post("/", h.create)
		r.Get("/", h.list)
		r.Get("/{role_id}/policies", h.listPolicies)
		r.Post("/{role_id}/policies", h.createPolicy)
		r.Delete("/{role_id}/policies/{resource}/{action}", h.deletePolicy)
	})
}
