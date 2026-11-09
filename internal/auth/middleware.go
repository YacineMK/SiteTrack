package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type contextKey string

const (
	userIDKey       contextKey = "user_id"
	appRoleKey      contextKey = "app_role"
	tokenVersionKey contextKey = "token_version"
)

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(accessCookieName)
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w)
			return
		}

		claims, err := s.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		ctx := WithContext(r.Context(), userID, claims.AppRole, claims.TokenVersion)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithContext(ctx context.Context, userID uuid.UUID, appRole string, tokenVersion int32) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, appRoleKey, appRole)
	return context.WithValue(ctx, tokenVersionKey, tokenVersion)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

func AppRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(appRoleKey).(string)
	return role, ok
}

func TokenVersion(ctx context.Context) (int32, bool) {
	version, ok := ctx.Value(tokenVersionKey).(int32)
	return version, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	jsonutil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
