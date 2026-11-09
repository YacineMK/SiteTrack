package auth

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(5, 10*time.Second))
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
			r.Get("/email-available", h.EmailAvailable)
		})
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthenticateMiddleware(h.svc))
			r.Get("/me", h.Me)
		})
	})
}
