package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

const (
	accessCookieName  = middleware.AccessCookieName
	refreshCookieName = "refresh_token"
)

type Handler struct {
	svc          *Service
	cookieSecure bool
}

func NewHandler(svc *Service, cookieSecure bool) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			writeError(w, http.StatusConflict, "registration failed")
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonutil.JSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.setSessionCookies(w, session)
	jsonutil.JSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeUnauthorized(w)
		return
	}

	session, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			writeUnauthorized(w)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.setSessionCookies(w, session)
	jsonutil.JSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var accessToken, refreshToken string
	if cookie, err := r.Cookie(accessCookieName); err == nil {
		accessToken = cookie.Value
	}
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		refreshToken = cookie.Value
	}

	if accessToken == "" && refreshToken == "" {
		writeUnauthorized(w)
		return
	}

	if err := h.svc.Logout(r.Context(), accessToken, refreshToken); err != nil {
		if !errors.Is(err, ErrUnauthorized) {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			writeUnauthorized(w)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	jsonutil.JSON(w, http.StatusOK, user)
}

func (h *Handler) setSessionCookies(w http.ResponseWriter, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    session.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.AccessExpiresAt,
		MaxAge:   int(time.Until(session.AccessExpiresAt).Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    session.RefreshToken,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.RefreshExpiresAt,
		MaxAge:   int(time.Until(session.RefreshExpiresAt).Seconds()),
	})
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func authResponse(session Session) AuthResponse {
	return AuthResponse{
		ExpiresIn: int64(time.Until(session.AccessExpiresAt).Seconds()),
		User:      session.User,
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	jsonutil.JSON(w, status, map[string]string{"error": message})
}
