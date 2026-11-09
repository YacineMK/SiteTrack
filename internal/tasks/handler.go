package tasks

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

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.svc.Create(r.Context(), chi.URLParam(r, "project_display_id"), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusCreated, task)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := h.pagination(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	tasks, err := h.svc.List(r.Context(), chi.URLParam(r, "project_display_id"),
		query.Get("q"), query.Get("status"), query.Get("from_date"), query.Get("to_date"),
		query.Get("sort_by"), query.Get("sort_dir"),
		query.Get("late") == "true", query.Get("blocked") == "true",
		limit, offset)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, tasks)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.Get(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, task)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateTaskRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.svc.Update(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"), req)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, task)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.Start(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, task)
}

func (h *Handler) done(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.Done(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, task)
}

func (h *Handler) reopen(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.Reopen(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	jsonutil.JSON(w, http.StatusOK, task)
}

func (h *Handler) addDependency(w http.ResponseWriter, r *http.Request) {
	var req AddDependencyRequest
	if err := jsonutil.DecodeJSON(r, &req); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		jsonutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.svc.AddDependency(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"), req.DependsOn)
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeDependency(w http.ResponseWriter, r *http.Request) {
	err := h.svc.RemoveDependency(r.Context(), chi.URLParam(r, "project_display_id"), chi.URLParam(r, "task_display_id"), chi.URLParam(r, "depends_on_display_id"))
	if err != nil {
		h.utils.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
