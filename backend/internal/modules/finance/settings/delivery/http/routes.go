package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/settings", func(r chi.Router) {
		r.Get("/channels", h.ListChannels)
		r.Put("/channels", h.UpdateChannels)

		r.Get("/templates/reminder", h.ListReminderTemplates)
		r.Put("/templates/reminder", h.UpsertReminderTemplate)

		r.Get("/roles", h.ListRoleCatalog)
		r.Post("/roles", h.CreateRole)
		r.Put("/roles/{roleID}", h.UpdateRole)
		r.Delete("/roles/{roleID}", h.DeleteRole)

		r.Get("/users", h.ListTenantUsers)
		r.Post("/users", h.CreateTenantUser)
		r.Put("/users/{membershipID}", h.UpdateTenantUser)
		r.Delete("/users/{membershipID}", h.DeleteTenantUser)

		r.Get("/users/roles", h.ListUsersRoles)
		r.Put("/users/roles/{membershipID}", h.UpdateMembershipRoles)

		r.Get("/audit-logs", h.ListAuditLogs)
	})
}
