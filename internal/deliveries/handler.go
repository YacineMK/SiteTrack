package deliveries

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/internal/middleware"
	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc     Service
	utils   Utils
	checker middleware.PolicyChecker
	access  middleware.ProjectAccess
}

func NewHandler(svc Service, checker middleware.PolicyChecker, access middleware.ProjectAccess) *Handler {
	return &Handler{svc: svc, utils: NewUtils(), checker: checker, access: access}
}

func (h *Handler) createCompany(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateCompanyDeliveryRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	delivery, err := h.svc.CreateFromCompany(r.Context(), actorID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, delivery)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserID(r.Context())
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateProjectDeliveryRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	delivery, err := h.svc.CreateFromProject(r.Context(), chi.URLParam(r, "project_display_id"), actorID, req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, delivery)
}

func (h *Handler) listCompany(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	deliveries, err := h.svc.ListCompany(r.Context(), query.Get("status"), query.Get("q"), query.Get("direction"), query.Get("from_date"), query.Get("to_date"), query.Get("project"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, deliveries)
}

func (h *Handler) listProject(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	deliveries, err := h.svc.ListProject(r.Context(), chi.URLParam(r, "project_display_id"), query.Get("status"), query.Get("q"), query.Get("direction"), query.Get("from_date"), query.Get("to_date"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, deliveries)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actorID, role, ok := deliveryActor(r)
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	delivery, err := h.svc.Get(r.Context(), actorID, role, chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	actorID, role, ok := deliveryActor(r)
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	delivery, err := h.svc.Record(r.Context(), actorID, role, chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	actorID, role, ok := deliveryActor(r)
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	delivery, err := h.svc.Reject(r.Context(), actorID, role, chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	actorID, role, ok := deliveryActor(r)
	if !ok {
		jsonutil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	delivery, err := h.svc.Cancel(r.Context(), actorID, role, chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func deliveryActor(r *http.Request) (uuid.UUID, string, bool) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		return uuid.UUID{}, "", false
	}
	role, _ := middleware.AppRole(r.Context())
	return userID, role, true
}

func (h *Handler) pagination(w http.ResponseWriter, r *http.Request) (int32, int32, bool) {
	query := r.URL.Query()

	limit := int64(20)
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid limit")
			return 0, 0, false
		}
		limit = parsed
	}

	offset := int64(0)
	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			jsonutil.Error(w, http.StatusBadRequest, "invalid offset")
			return 0, 0, false
		}
		offset = parsed
	}

	return int32(limit), int32(offset), true
}
