package roles

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
	"github.com/YacineMK/SiteTrack/kit/rbac"
	"github.com/YacineMK/SiteTrack/kit/reqctx"
)

type Handler struct {
	svc     Service
	utils   Utils
	checker middleware.PolicyChecker
}

func NewHandler(svc Service, checker middleware.PolicyChecker) *Handler {
	return &Handler{svc: svc, utils: NewUtils(), checker: checker}
}

// scope returns the project uuid from the request context when a project
// route resolved it, else nil (= app scope).
func scope(r *http.Request) *uuid.UUID {
	if id, ok := reqctx.ProjectID(r.Context()); ok {
		return &id
	}
	return nil
}

func (h *Handler) resources(w http.ResponseWriter, r *http.Request) {
	jsonutil.JSON(w, http.StatusOK, rbac.Resources)
}

func (h *Handler) assignRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req AssignRoleRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.AssignRole(r.Context(), userID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, user)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), scope(r), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, role)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	roleID, ok := roleIDParam(w, r)
	if !ok {
		return
	}
	var req UpdateRoleRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	role, err := h.svc.UpdateRole(r.Context(), scope(r), roleID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}
	jsonutil.JSON(w, http.StatusOK, role)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	roleID, ok := roleIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteRole(r.Context(), scope(r), roleID); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context(), scope(r))
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

	policy, err := h.svc.CreatePolicy(r.Context(), scope(r), roleID, req)
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

	policies, err := h.svc.ListPolicies(r.Context(), scope(r), roleID)
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
		scope(r),
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
