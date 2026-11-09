package companystore

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

func (h *Handler) listStock(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	stock, err := h.svc.List(r.Context(), r.URL.Query().Get("q"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, stock)
}

func (h *Handler) getStock(w http.ResponseWriter, r *http.Request) {
	stock, err := h.svc.Get(r.Context(), chi.URLParam(r, "material_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, stock)
}

func (h *Handler) addStock(w http.ResponseWriter, r *http.Request) {
	var req AddStockRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stock, err := h.svc.Add(r.Context(), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, stock)
}

func (h *Handler) setReorderPoint(w http.ResponseWriter, r *http.Request) {
	var req SetReorderPointRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stock, err := h.svc.SetReorderPoint(r.Context(), chi.URLParam(r, "material_id"), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, stock)
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
