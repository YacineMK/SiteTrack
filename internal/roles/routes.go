package roles

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.With(middleware.RequireAppPolicy(h.checker, "roles", "read")).Get("/resources", h.resources)
	r.With(middleware.RequireAppPolicy(h.checker, "users", "write")).Patch("/users/{user_id}/role", h.assignRole)

	r.Route("/roles", func(r chi.Router) {
		r.With(middleware.RequireAppPolicy(h.checker, "roles", "read")).Get("/", h.list)
		r.With(middleware.RequireAppPolicy(h.checker, "roles", "read")).Get("/{role_id}/policies", h.listPolicies)
		r.With(middleware.RequireAppPolicy(h.checker, "roles", "write")).Post("/", h.create)
		r.With(middleware.RequireAppPolicy(h.checker, "roles", "write")).Post("/{role_id}/policies", h.createPolicy)
		r.With(middleware.RequireAppPolicy(h.checker, "roles", "write")).Delete("/{role_id}/policies/{resource}/{action}", h.deletePolicy)
	})
}
