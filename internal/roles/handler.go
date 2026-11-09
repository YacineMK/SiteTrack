package roles

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/auth"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc   *Service
	authz *auth.AuthZ
}

func NewHandler(svc *Service, authz *auth.AuthZ) *Handler {
	return &Handler{svc: svc, authz: authz}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, role)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, roles)
}

func (h *Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	roleID, ok := roleIDParam(w, r)
	if !ok {
		return
	}

	var req CreatePolicyRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	policy, err := h.svc.CreatePolicy(r.Context(), roleID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, policy)
}

func (h *Handler) listPolicies(w http.ResponseWriter, r *http.Request) {
	roleID, ok := roleIDParam(w, r)
	if !ok {
		return
	}

	policies, err := h.svc.ListPolicies(r.Context(), roleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, policies)
}

func (h *Handler) deletePolicy(w http.ResponseWriter, r *http.Request) {
	roleID, ok := roleIDParam(w, r)
	if !ok {
		return
	}

	err := h.svc.DeletePolicy(
		r.Context(),
		roleID,
		chi.URLParam(r, "resource"),
		chi.URLParam(r, "action"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func roleIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	roleID, err := uuid.Parse(chi.URLParam(r, "role_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role id")
		return uuid.UUID{}, false
	}
	return roleID, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrRoleExists):
		writeError(w, http.StatusConflict, "role already exists")
	case errors.Is(err, ErrRoleNotFound):
		writeError(w, http.StatusNotFound, "role not found")
	case errors.Is(err, ErrPolicyExists):
		writeError(w, http.StatusConflict, "policy already exists")
	case errors.Is(err, ErrPolicyNotFound):
		writeError(w, http.StatusNotFound, "policy not found")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	jsonutil.JSON(w, status, map[string]string{"error": message})
}
