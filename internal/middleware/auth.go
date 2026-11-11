package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	"github.com/YacineMK/SiteTrack/kit/reqctx"
)

const AccessCookieName = "access_token"

type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string) (*kitjwt.Claims, error)
}

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

			ctx := reqctx.WithAuth(r.Context(), userID, claims.AppRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
