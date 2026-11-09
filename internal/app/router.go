package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

func (a *App) routes() {
	r := a.server.Mux()
	r.Use(middleware.RateLimit(60, 10*time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		jsonutil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	for _, m := range a.public {
		m.Routes(r)
	}

	r.Group(func(r chi.Router) {
		r.Use(a.authMiddleware)
		for _, m := range a.protected {
			m.Routes(r)
		}
	})
}
