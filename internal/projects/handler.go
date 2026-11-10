package projects

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	"github.com/YacineMK/SiteTrack/kit/reqctx"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc    Service
	utils  Utils
	access middleware.ProjectAccess
}

func NewHandler(svc Service, access middleware.ProjectAccess) *Handler {
	return &Handler{svc: svc, utils: NewUtils(), access: access}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := reqctx.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateProjectRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, project)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	userID, ok := reqctx.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	role, _ := reqctx.AppRole(r.Context())
	q := r.URL.Query()

	projects, err := h.svc.List(r.Context(), ListParams{
		UserID: userID, SuperAdmin: role == "super_admin",
		LateOnly: q.Get("late") == "true",
		Query:    q.Get("q"), Sort: q.Get("sort"),
		Limit: limit, Offset: offset,
	})
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, projects)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	isAdmin, _ := reqctx.IsProjectAdmin(r.Context())

	project, err := h.svc.Get(r.Context(), chi.URLParam(r, "display_id"), isAdmin)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, project)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateProjectRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isAdmin, _ := reqctx.IsProjectAdmin(r.Context())

	project, err := h.svc.Update(r.Context(), chi.URLParam(r, "display_id"), isAdmin, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, project)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "display_id")); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	projectID, ok := reqctx.ProjectID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusNotFound, "project not found")
		return
	}

	members, err := h.svc.ListMembers(r.Context(), projectID)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, members)
}

func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	projectID, ok := reqctx.ProjectID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusNotFound, "project not found")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateMemberRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.SetMemberAdmin(r.Context(), projectID, userID, req); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	projectID, ok := reqctx.ProjectID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusNotFound, "project not found")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.svc.RemoveMember(r.Context(), projectID, userID); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request) {
	projectID, ok := reqctx.ProjectID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusNotFound, "project not found")
		return
	}
	actorID, ok := reqctx.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req InviteRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invitation, err := h.svc.Invite(r.Context(), projectID, chi.URLParam(r, "display_id"), actorID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, invitation)
}

func (h *Handler) withdrawInvitation(w http.ResponseWriter, r *http.Request) {
	projectID, ok := reqctx.ProjectID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusNotFound, "project not found")
		return
	}

	notificationID, err := uuid.Parse(chi.URLParam(r, "invitation_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid invitation id")
		return
	}

	if err := h.svc.WithdrawInvitation(r.Context(), projectID, notificationID); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := reqctx.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notificationID, err := uuid.Parse(chi.URLParam(r, "invitation_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid invitation id")
		return
	}

	if err := h.svc.AcceptInvitation(r.Context(), userID, notificationID); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rejectInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := reqctx.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notificationID, err := uuid.Parse(chi.URLParam(r, "invitation_id"))
	if err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid invitation id")
		return
	}

	if err := h.svc.RejectInvitation(r.Context(), userID, notificationID); err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) pagination(w http.ResponseWriter, r *http.Request) (int32, int32, bool) {
	q := r.URL.Query()
	limit, offset := int64(20), int64(0)
	if value := q.Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid limit")
			return 0, 0, false
		}
		limit = parsed
	}
	if value := q.Get("offset"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid offset")
			return 0, 0, false
		}
		offset = parsed
	}
	return int32(limit), int32(offset), true
}
