package deliveries

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	storeManagers := middleware.AuthorizationMiddleware("company_store_manager", "super_admin")

	r.With(storeManagers).Post("/company-store/deliveries", h.createCompany)
	r.With(storeManagers).Get("/company-store/deliveries", h.listCompany)

	r.Post("/projects/{project_display_id}/deliveries", h.createProject)
	r.Get("/projects/{project_display_id}/deliveries", h.listProject)

	r.Get("/deliveries/{delivery_id}", h.get)
	r.Post("/deliveries/{delivery_id}/record", h.record)
	r.Post("/deliveries/{delivery_id}/reject", h.reject)
	r.Post("/deliveries/{delivery_id}/cancel", h.cancel)
}
