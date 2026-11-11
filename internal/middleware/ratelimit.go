package middleware

import (
	"net/http"
	"os"
	"strconv"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func ClientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(
		chimiddleware.GetClientIP(r.Context()),
	), nil
}

// RateLimit buckets by canonical client IP. Hard caps can be overridden via
// env vars RATE_LIMIT_GLOBAL and RATE_LIMIT_AUTH (useful for tests).
func RateLimit(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	// Dynamic override so e2e can disable without reshaping the handler chain.
	if override := os.Getenv("RATE_LIMIT_OVERRIDE"); override != "" {
		if n, err := strconv.Atoi(override); err == nil && n > 0 {
			requestLimit = n
		}
	}
	return httprate.LimitBy(requestLimit, window, ClientIPKey)
}
