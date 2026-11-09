package app

import (
	"net/http"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

func (a *App) routes() {
	r := a.server.Mux()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		jsonutil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.auth.Routes(r)
}
