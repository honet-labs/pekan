package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/finance/reports", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/transactions", h.CreateTransactionsReport)
		r.Route("/{reportID}", func(r chi.Router) {
			r.Get("/", h.GetByID)
			r.Get("/download", h.Download)
			r.Delete("/", h.Delete)
		})
	})
}
