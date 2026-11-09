package tasks

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireProjectMember(h.access))

		r.Post("/projects/{project_display_id}/tasks", h.create)
		r.Get("/projects/{project_display_id}/tasks", h.list)
		r.Get("/projects/{project_display_id}/tasks/{task_display_id}", h.get)
		r.Patch("/projects/{project_display_id}/tasks/{task_display_id}", h.update)
		r.Delete("/projects/{project_display_id}/tasks/{task_display_id}", h.delete)
		r.Post("/projects/{project_display_id}/tasks/{task_display_id}/start", h.start)
		r.Post("/projects/{project_display_id}/tasks/{task_display_id}/done", h.done)
		r.Post("/projects/{project_display_id}/tasks/{task_display_id}/reopen", h.reopen)
		r.Post("/projects/{project_display_id}/tasks/{task_display_id}/dependencies", h.addDependency)
		r.Delete("/projects/{project_display_id}/tasks/{task_display_id}/dependencies/{depends_on_display_id}", h.removeDependency)
	})
}
