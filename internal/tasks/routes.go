package tasks

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Post("/projects/{project_display_id}/tasks", h.create)
	r.Get("/projects/{project_display_id}/tasks", h.list)
	r.Get("/projects/{project_display_id}/tasks/{task_display_id}", h.get)
	r.Patch("/projects/{project_display_id}/tasks/{task_display_id}", h.update)
	r.Delete("/projects/{project_display_id}/tasks/{task_display_id}", h.delete)
}
