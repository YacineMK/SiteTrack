package notifications

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes(r chi.Router) {
	r.Get("/notifications", h.list)
	r.Patch("/notifications/{notification_id}/read", h.markRead)
	r.Get("/notifications/stream", h.stream)
}
