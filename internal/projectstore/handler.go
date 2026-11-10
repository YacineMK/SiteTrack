package projectstore

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
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

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	stock, err := h.svc.List(r.Context(), chi.URLParam(r, "project_display_id"), query.Get("q"), query.Get("category"), query.Get("sort"), limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, stock)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	stock, err := h.svc.Get(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "material_id"))
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
