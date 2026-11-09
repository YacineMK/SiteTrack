package companystore

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/company-store", func(r chi.Router) {
		r.Use(middleware.AuthorizationMiddleware("company_store_manager", "super_admin"))
		r.Get("/", h.list)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	jsonutil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
