package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
)

const AccessCookieName = "access_token"

type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string) (*kitjwt.Claims, error)
}

type contextKey string

const (
	userIDKey  contextKey = "user_id"
	appRoleKey contextKey = "app_role"
)

func AuthenticateMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AccessCookieName)
			if err != nil || cookie.Value == "" {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims, err := validator.ValidateAccessToken(r.Context(), cookie.Value)
			if err != nil {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := WithContext(r.Context(), userID, claims.AppRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithContext(ctx context.Context, userID uuid.UUID, appRole string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, appRoleKey, appRole)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

func AppRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(appRoleKey).(string)
	return role, ok
}
