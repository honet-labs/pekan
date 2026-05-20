package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/accounts", func(r chi.Router) {
		r.Get("/", h.ListAccounts)
		r.Post("/", h.CreateAccount)
	})

	r.Route("/finance/categories", func(r chi.Router) {
		r.Get("/", h.ListCategories)
		r.Post("/", h.CreateCategory)
		r.Get("/{id}", h.GetCategory)
		r.Put("/{id}", h.UpdateCategory)
		r.Delete("/{id}", h.DeleteCategory)
	})
}

