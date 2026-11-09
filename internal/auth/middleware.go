package auth

import (
	"context"
	"net/http"
	"strings"

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
		token := bearerToken(r)
		if token == "" {
			writeUnauthorized(w)
			return
		}

		claims, err := s.Authenticate(r.Context(), token)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, appRoleKey, claims.AppRole)
		ctx = context.WithValue(ctx, tokenVersionKey, claims.TokenVersion)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

func bearerToken(r *http.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func writeUnauthorized(w http.ResponseWriter) {
	jsonutil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
