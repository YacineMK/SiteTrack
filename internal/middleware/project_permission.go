package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	"github.com/YacineMK/SiteTrack/kit/rbac"
	"github.com/YacineMK/SiteTrack/kit/reqctx"
)

// ProjectEnforcer resolves the caller's project-scope policies for a project.
// is_admin=true short-circuits the policy check (bypass).
type ProjectEnforcer interface {
	ProjectPolicies(ctx context.Context, projectID, userID uuid.UUID) (policies []rbac.Policy, isAdmin bool, err error)
}

// RequireProjectPermission enforces resource/action for the caller's role in
// the project resolved by RequireProjectMember / RequireProjectAdmin.
// Super-admins and project admins bypass the policy check.
func RequireProjectPermission(enforcer ProjectEnforcer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := reqctx.AppRole(r.Context())
			if role == "super_admin" {
				next.ServeHTTP(w, r)
				return
			}

			userID, ok := reqctx.UserID(r.Context())
			projectID, ok2 := reqctx.ProjectID(r.Context())
			if !ok || !ok2 {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			policies, isAdmin, err := enforcer.ProjectPolicies(r.Context(), projectID, userID)
			if err != nil {
				jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if isAdmin || rbac.Allows(policies, resource, action) {
				next.ServeHTTP(w, r)
				return
			}
			jsonutil.Error(w, http.StatusForbidden, "forbidden")
		})
	}
}
