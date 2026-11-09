package deliveries

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	jsonutil "github.com/YacineMK/SiteTrack/kit/json"
)

type Handler struct {
	svc   Service
	utils Utils
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc, utils: NewUtils()}
}

func (h *Handler) createCompany(w http.ResponseWriter, r *http.Request) {
	var req CreateCompanyDeliveryRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	delivery, err := h.svc.CreateFromCompany(r.Context(), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, delivery)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectDeliveryRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	delivery, err := h.svc.CreateFromProject(r.Context(), chi.URLParam(r, "project_display_id"), req)
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

	deliveries, err := h.svc.ListCompany(r.Context(), r.URL.Query().Get("status"), limit, offset)
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

	deliveries, err := h.svc.ListProject(r.Context(), chi.URLParam(r, "project_display_id"), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, deliveries)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.svc.Get(r.Context(), chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.svc.Record(r.Context(), chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.svc.Reject(r.Context(), chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.svc.Cancel(r.Context(), chi.URLParam(r, "delivery_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, delivery)
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
