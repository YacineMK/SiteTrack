package projects

import (
	"github.com/go-chi/chi/v5"

	"github.com/YacineMK/SiteTrack/internal/middleware"
)

func (h *Handler) Routes(r chi.Router) {
	member := middleware.RequireProjectMember(h.access)
	admin := middleware.RequireProjectAdmin(h.access)

	r.Route("/projects", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
		r.With(member).Get("/{display_id}", h.get)
		r.With(admin).Patch("/{display_id}", h.update)
		r.With(admin).Delete("/{display_id}", h.delete)
		r.With(member).Get("/{display_id}/members", h.listMembers)
		r.With(admin).Patch("/{display_id}/members/{user_id}", h.updateMember)
		r.With(admin).Delete("/{display_id}/members/{user_id}", h.removeMember)
		r.With(admin).Post("/{display_id}/invitations", h.invite)
		r.With(admin).Get("/{display_id}/invitations", h.listInvitations)
		r.With(admin).Delete("/{display_id}/invitations/{invitation_id}", h.withdrawInvitation)
	})

	r.Post("/invitations/{invitation_id}/accept", h.acceptInvitation)
	r.Post("/invitations/{invitation_id}/reject", h.rejectInvitation)
}
