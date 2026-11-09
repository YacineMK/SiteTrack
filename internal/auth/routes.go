package auth

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Group(func(r chi.Router) {
			r.Use(h.svc.Middleware)
			r.Get("/me", h.Me)
			r.Post("/refresh", h.Refresh)
			r.Post("/logout", h.Logout)
		})
	})
}
