package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/dashboard", func(r chi.Router) {
		r.Get("/summary", h.Summary)
		r.Get("/series", h.Series)
		r.Get("/top-categories", h.TopCategories)
	})
}

