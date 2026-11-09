package auth

import (
	"context"
	"net/http"
)

type PolicyChecker interface {
	HasPolicy(ctx context.Context, role, resource, action string) (bool, error)
}

type AuthZ struct {
	policies PolicyChecker
}

func NewAuthZ(policies PolicyChecker) *AuthZ {
	return &AuthZ{policies: policies}
}

func (a *AuthZ) RequireAppRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current, ok := AppRole(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if current != role {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *AuthZ) RequireAppPolicy(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := AppRole(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			allowed, err := a.policies.HasPolicy(r.Context(), role, resource, action)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if !allowed {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
