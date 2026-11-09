package companystore

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.Route("/company-store", func(r chi.Router) {
		read := middleware.RequireAppPolicy(h.checker, "company_store", "read")
		write := middleware.RequireAppPolicy(h.checker, "company_store", "write")

		r.With(read).Get("/stock", h.listStock)
		r.With(read).Get("/stock/{material_id}", h.getStock)
		r.With(write).Post("/stock/additions", h.addStock)
		r.With(write).Put("/stock/{material_id}/reorder-point", h.setReorderPoint)
	})
}
