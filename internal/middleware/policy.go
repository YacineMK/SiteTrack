package middleware

import (
	"context"
	"net/http"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type PolicyChecker interface {
	HasPolicy(ctx context.Context, role, resource, action string) (bool, error)
}

func RequireAppPolicy(checker PolicyChecker, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := AppRole(r.Context())
			if !ok {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			allowed, err := checker.HasPolicy(r.Context(), role, resource, action)
			if err != nil {
				jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if !allowed {
				jsonutil.Error(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
