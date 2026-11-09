package projects

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

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := h.svc.Create(r.Context(), req)
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

	projects, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, projects)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	projects, err := h.svc.Search(r.Context(), r.URL.Query().Get("q"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, projects)
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

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	project, err := h.svc.Get(r.Context(), chi.URLParam(r, "display_id"))
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

	project, err := h.svc.Update(r.Context(), chi.URLParam(r, "display_id"), req)
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
