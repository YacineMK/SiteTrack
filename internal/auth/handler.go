package auth

import (
	"errors"
	"net/http"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	"github.com/YacineMK/SiteTrack/kit/reqctx"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

const (
	accessCookieName  = middleware.AccessCookieName
	refreshCookieName = "refresh_token"
)

type Handler struct {
	svc          Service
	cookieSecure bool
	utils        Utils
}

func NewHandler(svc Service, cookieSecure bool) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure, utils: NewUtils()}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			jsonutil.Error(w, http.StatusConflict, "registration failed")
		case errors.Is(err, ErrInvalidInput):
			jsonutil.Error(w, http.StatusBadRequest, "invalid input")
		default:
			jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonutil.JSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			jsonutil.Error(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, ErrInvalidInput):
			jsonutil.Error(w, http.StatusBadRequest, "invalid input")
		default:
			jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.utils.SetSessionCookies(w, session, h.cookieSecure)
	jsonutil.JSON(w, http.StatusOK, h.utils.AuthResponse(session))
}

func (h *Handler) EmailAvailable(w http.ResponseWriter, r *http.Request) {
	available, err := h.svc.EmailAvailable(r.Context(), r.URL.Query().Get("email"))
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			jsonutil.Error(w, http.StatusBadRequest, "invalid email")
			return
		}
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	jsonutil.JSON(w, http.StatusOK, map[string]bool{"available": available})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.utils.WriteUnauthorized(w)
		return
	}

	session, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			h.utils.WriteUnauthorized(w)
			return
		}
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.utils.SetSessionCookies(w, session, h.cookieSecure)
	jsonutil.JSON(w, http.StatusOK, h.utils.AuthResponse(session))
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
		h.utils.WriteUnauthorized(w)
		return
	}

	if err := h.svc.Logout(r.Context(), accessToken, refreshToken); err != nil {
		if !errors.Is(err, ErrUnauthorized) {
			jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	h.utils.ClearSessionCookies(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := reqctx.UserID(r.Context())
	if !ok {
		h.utils.WriteUnauthorized(w)
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			h.utils.WriteUnauthorized(w)
			return
		}
		jsonutil.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	jsonutil.JSON(w, http.StatusOK, user)
}
