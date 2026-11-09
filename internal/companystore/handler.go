package companystore

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/auth"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Dependencies struct {
	AuthZ *auth.AuthZ
}

type Handler struct {
	authz *auth.AuthZ
}

func New(deps Dependencies) *Handler {
	return &Handler{authz: deps.AuthZ}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/company-store", func(r chi.Router) {
		r.Use(h.authz.RequireAppPolicy("company_store", "read"))
		r.Get("/", h.list)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	jsonutil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
