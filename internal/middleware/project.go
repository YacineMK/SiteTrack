package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type ProjectAccess interface {
	ProjectAccess(ctx context.Context, displayID string, userID uuid.UUID) (uuid.UUID, bool, bool, bool, error)
}

const (
	projectIDKey    contextKey = "project_id"
	projectAdminKey contextKey = "project_admin"
)

func RequireProjectMember(access ProjectAccess) func(http.Handler) http.Handler {
	return requireProject(access, false)
}

func RequireProjectAdmin(access ProjectAccess) func(http.Handler) http.Handler {
	return requireProject(access, true)
}

func requireProject(access ProjectAccess, adminOnly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserID(r.Context())
			if !ok {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			displayID := chi.URLParam(r, "project_display_id")
			if displayID == "" {
				displayID = chi.URLParam(r, "display_id")
			}

			projectID, exists, member, isAdmin, err := access.ProjectAccess(r.Context(), displayID, userID)
			if err != nil {
				jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if !exists {
				jsonutil.Error(w, http.StatusNotFound, "project not found")
				return
			}

			role, _ := AppRole(r.Context())
			superAdmin := role == "super_admin"
			if !superAdmin && !member {
				jsonutil.Error(w, http.StatusNotFound, "project not found")
				return
			}
			if adminOnly && !superAdmin && !isAdmin {
				jsonutil.Error(w, http.StatusForbidden, "forbidden")
				return
			}

			ctx := context.WithValue(r.Context(), projectIDKey, projectID)
			ctx = context.WithValue(ctx, projectAdminKey, isAdmin || superAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ProjectID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(projectIDKey).(uuid.UUID)
	return id, ok
}

func ProjectAdmin(ctx context.Context) (bool, bool) {
	admin, ok := ctx.Value(projectAdminKey).(bool)
	return admin, ok
}
