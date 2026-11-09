package auth

import (
	"errors"
	"net/http"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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

	auth, err := h.svc.Login(r.Context(), req)
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

	jsonutil.JSON(w, http.StatusOK, auth)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	auth, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			writeUnauthorized(w)
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonutil.JSON(w, http.StatusOK, auth)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), bearerToken(r), req); err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			writeUnauthorized(w)
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
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

func writeError(w http.ResponseWriter, status int, message string) {
	jsonutil.JSON(w, status, map[string]string{"error": message})
}
