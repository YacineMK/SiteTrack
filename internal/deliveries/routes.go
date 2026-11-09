package deliveries

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.With(middleware.RequireAppPolicy(h.checker, "company_store_deliveries", "write")).Post("/company-store/deliveries", h.createCompany)
	r.With(middleware.RequireAppPolicy(h.checker, "company_store_deliveries", "read")).Get("/company-store/deliveries", h.listCompany)

	member := middleware.RequireProjectMember(h.access)
	r.With(member).Post("/projects/{project_display_id}/deliveries", h.createProject)
	r.With(member).Get("/projects/{project_display_id}/deliveries", h.listProject)

	r.Get("/deliveries/{delivery_id}", h.get)
	r.Post("/deliveries/{delivery_id}/record", h.record)
	r.Post("/deliveries/{delivery_id}/reject", h.reject)
	r.Post("/deliveries/{delivery_id}/cancel", h.cancel)
}
