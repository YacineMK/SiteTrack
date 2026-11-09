package roles

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc   Service
	utils Utils
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc, utils: NewUtils()}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, role)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		h.utils.WriteServiceError(w, err)
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
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	policy, err := h.svc.CreatePolicy(r.Context(), roleID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
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
		h.utils.WriteServiceError(w, err)
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
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func roleIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	roleID, err := uuid.Parse(chi.URLParam(r, "role_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid role id")
		return uuid.UUID{}, false
	}
	return roleID, true
}
