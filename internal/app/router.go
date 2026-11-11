package app

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

func (a *App) routes() {
	r := a.server.Mux()
	r.Use(middleware.RateLimit(60, 10*time.Second))

	r.Get("/health", a.health)

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

// health pings postgres and redis in parallel, returns 200 only when both
// are reachable. /health is the readiness probe for load balancers and
// monitoring, so a flaky dependency must show as a non-200.
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	type check struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	body := struct {
		Status   string `json:"status"`
		Postgres check  `json:"postgres"`
		Redis    check  `json:"redis"`
	}{Status: "ok"}

	if err := a.postgres.Pool.Ping(ctx); err != nil {
		body.Postgres = check{Status: "down", Error: err.Error()}
		body.Status = "degraded"
	} else {
		body.Postgres = check{Status: "ok"}
	}

	if err := a.redis.Client.Ping(ctx).Err(); err != nil {
		body.Redis = check{Status: "down", Error: err.Error()}
		body.Status = "degraded"
	} else {
		body.Redis = check{Status: "ok"}
	}

	code := http.StatusOK
	if body.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	jsonutil.JSON(w, code, body)
}
