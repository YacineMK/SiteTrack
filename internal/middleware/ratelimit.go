package middleware

import (
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func ClientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(
		chimiddleware.GetClientIP(r.Context()),
	), nil
}

func RateLimit(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(requestLimit, window, ClientIPKey)
}
