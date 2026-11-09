package companystore

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.Route("/company-store", func(r chi.Router) {
		r.Use(middleware.AuthorizationMiddleware("company_store_manager", "super_admin"))

		r.Get("/stock", h.listStock)
		r.Get("/stock/{material_id}", h.getStock)
		r.Post("/stock/additions", h.addStock)
		r.Put("/stock/{material_id}/reorder-point", h.setReorderPoint)
	})
}
