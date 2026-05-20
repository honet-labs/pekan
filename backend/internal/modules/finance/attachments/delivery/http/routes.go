package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/attachments", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Upload)
		r.Get("/{attachmentID}/download", h.Download)
		r.Delete("/{attachmentID}", h.Delete)
	})
}

